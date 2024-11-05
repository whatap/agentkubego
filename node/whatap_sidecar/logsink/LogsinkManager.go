package logsink

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"log"
	"time"

	whatapio "github.com/whatap/golib/io"
	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/lang/value"
	whataphash "github.com/whatap/golib/util/hash"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/dateutil"

	corev1 "k8s.io/api/core/v1"
	"whatap.io/k8s/sidecar/config"
	"whatap.io/k8s/sidecar/kube"
	"whatap.io/k8s/sidecar/session"
)

const (
	CONTAINER_LOG_CATEOGORY = "container_stdout"
	ZIP_MOD_DEFULAT_GZIP    = 1
	queueTimeout            = 1 * time.Minute
	batchSize               = 60
)

var (
	Pcode           int64
	Oid             int32
	OnodeName       string
	containerLookup = map[string]bool{}
	containers      []*ContainerLogInfo
	sendBuffer      chan pack.Pack
	sendBufferSize  = 1024
)

type ContainerLogInfo struct {
	cid       string
	ns        string
	podName   string
	cName     string
	isStarted bool
	lineNo    int64
}

func (this *ContainerLogInfo) start() {
	this.isStarted = true
	this.lineNo = 1
	go this.poll()
}

func (this *ContainerLogInfo) poll() {
	conf := config.GetConfig()
	cli, err := kube.GetKubeClient()
	if err != nil {
		log.Println(err)
		return
	}

	since := int64(1)

	podLogOpts := corev1.PodLogOptions{Follow: true, Container: this.cName, SinceSeconds: &since}
	req := cli.CoreV1().Pods(this.ns).GetLogs(this.podName, &podLogOpts)
	// log.Println("ContainerLogInfo.poll ns:", this.ns, " pod:", this.podName, podLogOpts)
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		log.Println("LogSink podLogOpts:", podLogOpts)
		log.Println("LogSink podns:", this.ns)
		log.Println("LogSink podname:", this.podName)
		log.Println("LogSink", err)
		return
	}

	// log.Println("ContainerLogInfo.poll step -1")

	defer podLogs.Close()

	// linebuf := make([]byte, 1024)
	scanner := bufio.NewScanner(podLogs)
	// log.Println("ContainerLogInfo.poll step -1.1")
	for scanner.Scan() {
		// log.Println("ContainerLogInfo.poll step -1.1.1")
		// nbyteThisTime, err := podLogs.Read(linebuf)
		// if nbyteThisTime < 1 || err != nil {
		// 	log.Println("ContainerLogInfo.poll step -1.2 ", err)
		// 	break
		// }
		// log.Println("ContainerLogInfo.poll step -1.1.2")
		// log.Println("ContainerLogInfo.poll step -2 ns:", this.ns, " pod:", this.podName, " c:", this.cName, " linebuf:", nbyteThisTime)
		if !conf.LogsinkEnabled {
			this.isStarted = false
			return
		}
		line := scanner.Text()
		length := len(line)
		if length > 0 {

			this.lineNo += int64(length)
			// log.Println("ContainerLogInfo.poll step -6 ", line)
			logsinkPack := createLogSink(this.ns, this.podName, this.cName, this.cid, OnodeName)
			logsinkPack.Line = this.lineNo
			logsinkPack.Content = line

			// log.Println("ContainerLogInfo.poll step -6.1 ", dateutil.DateTime(logsinkPack.Time), "pcode: ", logsinkPack.Pcode, " oid: ", logsinkPack.Oid, " category: ", logsinkPack.Category)
			// log.Println("ContainerLogInfo.poll step -6.2 ", logsinkPack.ToString())
			// send(logsinkPack)
			if logsinkPack != nil && len(sendBuffer) < (sendBufferSize-1) {
				sendBuffer <- logsinkPack
			}

		}
		// log.Println("ContainerLogInfo.poll step -7")
		time.Sleep(time.Second * 10)
	}
}

func createLogSink(ns string, podName string, cName string, cid string, nodeName string) *pack.LogSinkPack {
	p := pack.NewLogSinkPack()
	p.Pcode = Pcode
	p.Oid = Oid
	p.Category = CONTAINER_LOG_CATEOGORY
	p.Time = dateutil.Now()
	p.Tags.PutString("namespace", ns)
	p.Tags.PutString("pod", podName)
	p.Tags.Put("podHash", value.NewDecimalValue(int64(whataphash.HashStr(podName))))

	p.Tags.PutString("container", cName)
	p.Tags.PutString("containerId", cid)
	p.Tags.Put("containerKey", value.NewDecimalValue(int64(whataphash.HashStr(cid))))
	p.Tags.PutString("onodeName", nodeName)
	p.Tags.Put("onode", value.NewDecimalValue(int64(whataphash.HashStr(nodeName))))

	return p
}

func OnContainerDetected(cid string, ns string, podName string, cName string) {

	if _, ok := containerLookup[cid]; !ok {
		// log.Println("OnContainerDetected cid:", cid, " ns:", ns, " pod:", podName, " cname:", cName)
		containerLookup[cid] = true
		containers = append(containers, &ContainerLogInfo{ns: ns, podName: podName, cName: cName, cid: cid})
	}
}

func StartPolling() {
	// log.Println("StartPolling step -1")
	sendBuffer = make(chan pack.Pack, sendBufferSize)
	// log.Println("StartPolling step -2")
	go startLogTail()
	go startQueueListener()
	// log.Println("StartPolling step -3")
}

func startLogTail() {
	conf := config.GetConfig()
	for {
		for _, c := range containers {
			if !c.isStarted && conf.LogsinkEnabled {
				// log.Println("starting pod:", c.podName, " c:"+c.cName)
				c.start()
			}
		}
		time.Sleep(time.Second * 10)
	}
}

func startQueueListener() {

	for {
		var packsThisTime []pack.Pack
		lastPackReceive := int64(0)
		for {
			now := dateutil.Now()
			select {
			case p := <-sendBuffer:
				packsThisTime = append(packsThisTime, p)
				if len(packsThisTime) == batchSize || now-lastPackReceive > queueTimeout.Milliseconds() {
					sendZippedPacks(packsThisTime)

					packsThisTime = nil
					lastPackReceive = 0
				}
				if lastPackReceive == 0 {
					lastPackReceive = now
				}
			case <-time.After(queueTimeout):
				if len(packsThisTime) > 0 {
					sendZippedPacks(packsThisTime)
					// sendPacks(packsThisTime)
					packsThisTime = nil
					lastPackReceive = 0
				}
			}
		}
	}
}

func sendPacks(packs []pack.Pack) {
	for _, p := range packs {
		send(p)
	}
}

func sendZippedPacks(packs []pack.Pack) {
	// log.Println("sendZippedPacks step -1")
	zp := pack.NewZipPack()

	zp.Pcode = Pcode
	zp.Oid = Oid
	zp.Time = dateutil.Now()
	zp.RecountCount = len(packs)
	zp.Records = getStream(packs)

	// log.Println("sendZippedPacks step -2", zp.RecountCount)
	err := doZip(zp)
	// log.Println("sendZippedPacks step -3", err)
	if err != nil {
		// log.Println("sending log sink", err)
		return
	}
	// log.Println("sendZippedPacks step -4")
	send(zp)
}

func doZip(p *pack.ZipPack) error {
	buf := new(bytes.Buffer)

	gz := gzip.NewWriter(buf)
	_, gzipErr := gz.Write(p.Records)
	if gzipErr != nil {
		return gzipErr
	}
	gz.Close()

	p.Status = ZIP_MOD_DEFULAT_GZIP
	p.Records = buf.Bytes()

	return nil
}

func getStream(packs []pack.Pack) []byte {
	dout := whatapio.NewDataOutputX()

	for _, p := range packs {
		pack.WritePack(dout, p)
	}

	return dout.ToByteArray()
}

func send(p pack.Pack) bool {

	return session.Send(p)
}
