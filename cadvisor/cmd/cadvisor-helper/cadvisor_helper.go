package main

import (
	"flag"
	"github.com/whatap/kube/cadvisor/router"
	pods_scraper "github.com/whatap/kube/cadvisor/scraper/pods"
	"github.com/whatap/kube/tools/util/logutil"
)

func main() {
	portFlag := flag.Int("port", 6801, "string, whatap-node-helper port")
	//testTypeFlag := flag.String("t", "", "string , image - get agent path, spec - spec")
	//containerID := flag.String("id", "", "string , id - agent container id")
	//flag.Parse()
	//
	//// Check for test flags and run appropriate test
	//switch *testTypeFlag {
	//case "image":
	//	hack.RunImageTest(*containerID)
	//case "spec":
	//	hack.RunSpecTest(*containerID)
	//}

	// Main application logic
	logutil.Infof("run cadvisor", "port=%v\n", *portFlag)
	pods_scraper.RunPodInformer()
	router.Route()
}
