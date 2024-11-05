package client

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/whatap/kube/master/pkg/config"
	"log"
	"net/http"
	"os"
)

var client *http.Client

func init() {
	if config.Conf.EtcdMonitoringEnabled {
		// PEM 인증서와 키파일 로드
		cert, err := tls.LoadX509KeyPair(config.Conf.EtcdClientCertPath, config.Conf.EtcdClientKeyPath)
		if err != nil {
			log.Fatalf("Error loading key pair: %v", err)
		}

		// 루트 CA 로드
		caCert, err := os.ReadFile(config.Conf.EtcdCaCertPath)
		if err != nil {
			log.Fatalf("Error reading CA certificate: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		// HTTP 클라이언트 설정 구성
		tlsConfig := &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			InsecureSkipVerify: false,
		}
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		}
	}
}

func GetEtcdClient() *http.Client {
	return client
}
