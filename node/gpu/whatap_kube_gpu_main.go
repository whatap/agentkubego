package main

import (
	"flag"
	"fmt"
)

func main() {
	fmt.Println("whatap_gpu")
	nodeHost := flag.String("nodehost", "localhost", "whatap node host")
	nodePort := flag.Int("nodeport", 6600, "whatap node port")
	flag.Parse()

	Host = *nodeHost
	Port = *nodePort

	if !validate() {

		return
	}

	collectGpuForever()

}
