package main

import (
	"github.com/whatap/kube/master/pkg/config"
	"github.com/whatap/kube/master/router"
	"github.com/whatap/kube/master/scraper/etcd"
	"github.com/whatap/kube/master/scraper/kube-apiserver"
	"log"
)

func main() {
	/*
		수집기
	*/
	if config.Conf.CollectKubeApiserverMonitoringEnabled {
		log.Println("kube-apiserver monitoring config is enabled. kube-apiserver metrics scraper started...")
		kube_apiserver.Do()
	} else {
		log.Println("kube-apiserver monitoring config is disabled.")
	}

	if config.Conf.CollectEtcdMonitoringEnabled {
		log.Println("etcd monitoring config is enabled. etcd metrics scraper started...")
		etcd.Do()
	} else {
		log.Println("etcd monitoring config is disabled.")
	}

	/*
		응답기
	*/
	router.Route()
}
