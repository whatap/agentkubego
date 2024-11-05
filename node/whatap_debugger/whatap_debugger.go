package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/namespaces"
	"github.com/whatap/kube/node/src/whatap/client"
	"github.com/whatap/kube/node/src/whatap/confbase"
	whatap_containerd "github.com/whatap/kube/node/src/whatap/util/containerd"
	"github.com/whatap/kube/node/src/whatap/util/osutil"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
)

const (
	HOSTPATH_PREFIX = "/rootfs"
)

func panicExit(err error) {
	fmt.Println(err)
}

func main() {
	var outprefix string
	flag.StringVar(&outprefix, "outprefix", osutil.GetEnv("OUT_PREFIX", "/whatap"), "out prefix")
	err := os.MkdirAll(outprefix, 0777)
	if err != nil {
		panic(err)
	}

	var outputfile string
	flag.StringVar(&outputfile, "outfile", osutil.GetEnv("OUT_FILE", "whatap_debug.out"), "out file")

	fpLog, err := os.OpenFile(outputfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer fpLog.Close()

	multiWriter := io.MultiWriter(fpLog, os.Stdout)
	log.SetOutput(multiWriter)

	var confbaseHost string
	flag.StringVar(&confbaseHost, "confbaseHost", osutil.GetEnv("WHATAP_CONFBASE", "whatap-master-agent.whatap-monitoring"), "confbase host")
	confbasePort := flag.Int("confbasePort", 6800, "confbase port")

	var nodeName string
	flag.StringVar(&nodeName, "nodeName", os.Getenv("NODE_NAME"), "node name")

	var containerId string
	flag.StringVar(&containerId, "containerId", "nonExistingContainerId", "container id")

	flag.Parse()
	var cmd string
	if len(flag.Args()) > 0 {
		cmd = flag.Args()[0]
	}
	log.Println("OUT_PREFIX:", outprefix, " OUT_FILE:", outputfile)
	log.Println("WHATAP_CONFBASE:", confbaseHost, " PORT:", *confbasePort)

	osutil.Hostname(func(hostname string) {
		log.Println("POD:", hostname)
	})

	if cmd == "run" {
		requiredEnv := map[string]bool{}
		requiredEnv["NODE_IP"] = false
		requiredEnv["NODE_NAME"] = false
		requiredEnv["POD_NAME"] = false

		osutil.GetEnvAll(func(key string, val string) {
			if _, ok := requiredEnv[key]; ok {
				requiredEnv[key] = true
			}
		})

		log.Println("=============================================================================================")
		log.Println("= ENV CHECK")
		log.Println("= ")

		for k, v := range requiredEnv {
			log.Println("= ", k, ":", v)
		}
		log.Println("=============================================================================================")

		log.Println("=============================================================================================")
		log.Println("= CONFBASE HOST:", confbaseHost, " PORT:", *confbasePort)
		log.Println("= ")
		err = confbase.GetConfig("kube_namespace", confbaseHost, *confbasePort,
			func(key string, val string) {
				log.Println("= ", key, ":", val)

			})
		if err != nil {
			log.Println("= Status : PROBLEM")
			log.Println("= Error : ", err)
		} else {
			log.Println("= Status : OK")
		}
		log.Println("=============================================================================================")

		log.Println("=============================================================================================")
		log.Println("= MASTER AGENT HOST:", confbaseHost, " PORT:", 6600)
		log.Println("= ")

		var nodeAccessIP string
		var nodeAccessPort int32
		err = confbase.GetNodeAccess(confbaseHost, 6600, nodeName, func(nodeIP string, nodePort int32) {
			log.Println("= NodeIP :", nodeIP)
			nodeAccessIP = nodeIP
			nodeAccessPort = nodePort
		})
		if err != nil {
			log.Println("= Status : PROBLEM")
			log.Println("= Error : ", err)
		} else {
			log.Println("= Status : OK")
		}
		log.Println("=============================================================================================")

		log.Println("=============================================================================================")
		log.Println("= NODE AGENT HOST:", nodeAccessIP, " PORT:", nodeAccessPort)
		log.Println("= CONTAINER ID:", containerId)

		log.Println("= ")

		err = confbase.GetContainerPerf(nodeAccessIP, int(nodeAccessPort), containerId, func(key string, val string) {
			log.Println("= ", key, ":", val)
		})
		if err != nil {
			log.Println("= Status : PROBLEM")
			log.Println("= Error : ", err)
		} else {
			log.Println("= Status : OK")
		}
		log.Println("=============================================================================================")

	} else if cmd == "containerd" {
		log.Println("=============================================================================================")
		log.Println("Loading Container")

		resp, ctx, err := loadContainerD(containerId)
		if err != nil {
			log.Println("loading ", containerId, "error :", err)
			return
		}

		spec, err := resp.Spec(ctx)
		if err != nil {
			log.Println("getting spec", containerId, "error :", err)
			return
		}
		containerJson, err := json.Marshal(spec)
		if err != nil {
			log.Println("marshaling ", containerId, "error :", err)
			return
		}
		log.Println(string(containerJson))

		cexts, errext := resp.Extensions(ctx)
		if errext != nil {
			log.Println("getting Extensions ", containerId, "error :", err)

			return
		}
		containerJson, err = json.Marshal(cexts)
		if err != nil {
			log.Println("marshaling ", containerId, "error :", err)
			return
		}
		log.Println(string(containerJson))

		task, err := resp.Task(ctx, nil)
		if err != nil {
			log.Println("getting Task ", containerId, "error :", err)
			return
		}
		status, err := task.Status(ctx)
		if err != nil {
			log.Println("getting Status ", containerId, "error :", err)
			return
		}
		log.Println(" status.Status ", status.Status)

	} else if cmd == "allContainerd" {
		log.Println("=============================================================================================")
		log.Println("Loading All Container")

		getAllContainerPerf()
	}
}

func getAllContainerPerf() {
	cli, err := client.GetKubernetesClient()
	if err != nil {
		log.Println("getting kube client error:", err)
		return
	}

	containerdcli, err := getContainerdClient()
	if err != nil {
		log.Println("getting containerd client error:", err)
		return
	}

	nodename := os.Getenv("NODE_NAME")
	listOptions := metav1.ListOptions{FieldSelector: fmt.Sprint("spec.nodeName=", nodename)}
	pods, err := cli.CoreV1().Pods("").List(context.Background(), listOptions)
	if err != nil {
		log.Println("getting pods error:", err)
		return
	}

	for _, pod := range pods.Items {
		for _, c := range pod.Status.ContainerStatuses {
			containerid := c.ContainerID[len("containerd://"):]
			log.Println("=============================================================================================")
			log.Println("POD : ", pod.Name, " Container:", c.Name, " ID:", containerid)

			resp, ctx, err := loadContainerD(containerid)
			if err != nil {
				log.Println("loading ", containerid, "error :", err)
				continue
			}

			spec, err := resp.Spec(ctx)
			if err != nil {
				log.Println("getting spec", containerid, "error :", err)
				continue
			}
			cgroupParent := spec.Linux.CgroupsPath
			getcontainerresp, err := containerdcli.TaskService().Get(ctx, &tasks.GetRequest{
				ContainerID: containerid,
			})
			if err != nil {
				log.Println("getting container pid ", containerid, "error :", err)
				continue
			}
			pid := int(getcontainerresp.Process.Pid)
			cstat, err := whatap_containerd.GetContainerStatsExRaw(HOSTPATH_PREFIX, containerid, c.Name, cgroupParent,
				0, pid, 0)
			if err != nil {
				log.Println("getting stat ", containerid, "error :", err)
				continue
			}
			log.Println("CgroupParent: ", cgroupParent)
			log.Println("Cpu Total Usage: ", cstat.CPUStats.CPUUsage.TotalUsage, " Memory Usage:", cstat.MemoryStats.Stats.Rss)
		}
	}
}

var containerdClient *containerd.Client
var containerdNamespaces []string

func getContainerdClient() (*containerd.Client, error) {
	if containerdClient == nil {
		newContainerdClient, err := containerd.New("/run/containerd/containerd.sock")
		if err != nil {
			return nil, err
		}

		containerdClient = newContainerdClient

		if nss, err := containerdClient.NamespaceService().List(context.Background()); err == nil {
			containerdNamespaces = nss

		}
	}
	return containerdClient, nil
}

func loadContainerD(containerid string) (containerd.Container, context.Context, error) {
	cli, err := getContainerdClient()
	if err != nil {
		return nil, nil, err
	}

	for _, containerdNamespace := range containerdNamespaces {
		ctx := namespaces.WithNamespace(context.Background(), containerdNamespace)

		resp, err := cli.LoadContainer(ctx, containerid)
		if err == nil {
			return resp, ctx, err
		}

	}

	return nil, nil, fmt.Errorf("container ", containerid, " not found")
}
