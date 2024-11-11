package main

import (
	"github.com/whatap/kube/controlplane/pkg/config"
	"github.com/whatap/kube/controlplane/router"
	"github.com/whatap/kube/controlplane/scraper/etcd"
	"github.com/whatap/kube/controlplane/scraper/kube-apiserver"
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

	if config.Conf.CollectKubeSchedulerMonitoringEnabled {
		log.Println("kube-scheduler monitoring config is enabled. kube-scheduler metrics scraper started...")
		etcd.Do()
	} else {
		log.Println("kube-scheduler monitoring config is disabled.")
	}
	/*
		응답기
	*/
	router.Route()
}
