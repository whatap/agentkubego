package client

import (
	"crypto/tls"
	"github.com/whatap/kube/controlplane/pkg/config"
	"net/http"
)

var schedulerClient *http.Client

func init() {
	if config.Conf.CollectKubeSchedulerMonitoringEnabled {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS13, // TLS 1.3 사용, 필요에 따라 버전을 조정
				InsecureSkipVerify: true,
			},
		}
		schedulerClient = &http.Client{Transport: transport}
	}
}

func GetSchedulerClient() *http.Client {
	return schedulerClient
}
