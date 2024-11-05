package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

var Conf *Configure
var loadConfigMap map[string]string

type Configure struct {
	Debug bool
	Port  string
	Cycle int

	CollectControlPlaneMonitoringEnabled bool

	//kube-apiserver client config
	KubeApiserverMonitoringEnabled bool
	KubeConfigPath                 string
	KubeMasterUrl                  string
	KubeClientTlsVerify            bool

	//etcd config
	EtcdMonitoringEnabled bool
	EtcdHosts             []string
	EtcdMetricsEndpoint   string
	EtcdPort              string
	EtcdCaCertPath        string
	EtcdClientCertPath    string
	EtcdClientKeyPath     string
}

func init() {
	loadConfigMap = make(map[string]string)
	configLoad()

	Conf = &Configure{
		Debug: getBool("debug", "false"),
		Port:  getString("port", "9496"),
		Cycle: getInt("cycle", "5"),

		CollectControlPlaneMonitoringEnabled: getBool("collect_control_plane_monitoring_enabled", "false"),

		KubeApiserverMonitoringEnabled: getBool("kube_apiserver_monitoring_enabled", getString("collect_control_plane_monitoring_enabled", "false")),
		KubeConfigPath:                 getString("kube_config_path", ""),
		KubeMasterUrl:                  getString("kube_master_url", ""),
		KubeClientTlsVerify:            getBool("kube_client_tls_verify", "true"),

		EtcdMonitoringEnabled: getBool("etcd_monitoring_enabled", "false"),
		EtcdHosts:             getStringList("etcd_hosts", []string{}),
		EtcdMetricsEndpoint:   getString("etcd_metrics_endpoint", "/metrics"),
		EtcdPort:              getString("etcd_port", "2379"),
		EtcdCaCertPath:        getString("etcd_ca_cert_path", "/etc/kubernetes/pki/etcd/ca.crt"),
		EtcdClientCertPath:    getString("etcd_client_cert_path", "/etc/kubernetes/pki/etcd/server.crt"),
		EtcdClientKeyPath:     getString("etcd_client_key_path", "/etc/kubernetes/pki/etcd/server.key"),
	}

	printConf()
}

func printConf() {
	log.Println("===== Print Configure =====")
	log.Println("------ control plane helper standard config ------")
	log.Println("debug:", strconv.FormatBool(Conf.Debug))
	log.Println("port:", Conf.Port)
	log.Println("cycle:", strconv.FormatInt(int64(Conf.Cycle), 10))
	log.Println("collect_control_plane_monitoring_enabled:", strconv.FormatBool(Conf.CollectControlPlaneMonitoringEnabled))

	log.Println()
	log.Println("------ kube-apiserver config ------")
	log.Println("kube_apiserver_monitoring_enabled:", strconv.FormatBool(Conf.KubeApiserverMonitoringEnabled))
	log.Println("kube_config_path:", Conf.KubeConfigPath)
	log.Println("kube_master_url:", Conf.KubeMasterUrl)
	log.Println("kube_client_tls_verify:", strconv.FormatBool(Conf.KubeClientTlsVerify))

	log.Println()
	log.Println("------ etcd config ------")
	log.Println("etcd_monitoring_enabled:", Conf.EtcdMonitoringEnabled)
	log.Println("etcd_urls:", Conf.EtcdHosts)
	log.Println("etcd_metrics_endpoint:", Conf.EtcdMetricsEndpoint)
	log.Println("etcd_port:", Conf.EtcdPort)
	log.Println("etcd_ca_cert_path:", Conf.EtcdCaCertPath)
	log.Println("etcd_client_cert_path:", Conf.EtcdClientCertPath)
	log.Println("etcd_client_key_path:", Conf.EtcdClientKeyPath)
	log.Println()
}

func getStringList(key string, defaultList []string) []string {
	value, ok := getValue(key)
	if ok {
		if len(value) == 0 {
			return []string{}
		}
		valueList := strings.Split(value, ",")
		return valueList
	} else {
		logSetDefault(key, defaultList)
		return defaultList
	}
}

func getInt(key string, defaultValue string) int {
	value := getValueWithDefault(key, defaultValue)
	atoi, err := strconv.Atoi(value)
	if err != nil {
		defaultVal, err := strconv.Atoi(defaultValue)
		if err != nil {
			loadParseFail(key, value, defaultValue)
			return 0
		}
		return defaultVal
	}
	return atoi
}

func getBool(key string, defaultValue string) bool {
	value := getValueWithDefault(key, defaultValue)
	parseBool, err := strconv.ParseBool(value)
	if err != nil {
		defaultVal, err := strconv.ParseBool(defaultValue)
		if err != nil {
			loadParseFail(key, value, defaultValue)
			return false
		}
		return defaultVal
	}
	return parseBool
}

func getFloat(key string, defaultValue string) float64 {
	value := getValueWithDefault(key, defaultValue)
	float, err := strconv.ParseFloat(value, 64)
	if err != nil {
		defaultVal, err := strconv.ParseFloat(defaultValue, 64)
		if err != nil {
			loadParseFail(key, value, defaultValue)
			return 0
		}
		return defaultVal
	}
	return float
}

func getString(key string, defaultValue string) string {
	return getValueWithDefault(key, defaultValue)
}

func getValueWithDefault(key string, defaultValue string) string {
	v, ok := loadConfigMap[key]
	if ok {
		return v
	}
	logSetDefault(key, defaultValue)
	return defaultValue
}

func getValue(key string) (value string, ok bool) {
	v, ok := loadConfigMap[key]
	if ok {
		return v, true
	} else {
		return "", false
	}
}

func logSetDefault(key string, defaultValue interface{}) {
	log.Println("there is no env specified, so set to default. key=", key, ", default value=", defaultValue)
}

func loadParseFail(key string, value string, defaultValue string) {
	log.Println("config parse fail. key=", key, ", env value=", value, ",", ", default value=", defaultValue)
}

/*
system env 보다 program arguments 가 선순위
*/
func configLoad() map[string]string {
	loadFromEnv()
	loadFromArgs()
	return loadConfigMap
}

func loadFromEnv() {
	env := os.Environ()
	load(env)
}

func loadFromArgs() {
	args := os.Args
	load(args)
}

func load(env []string) {
	for _, e := range env {
		idx := strings.Index(e, "=")
		if idx == -1 {
			// = 이 포함되지 않은 env 에 대한 처리
			continue
		}
		key := e[:idx]
		cutPrefixKey := strings.TrimPrefix(key, "-")
		val := e[idx+1:]
		loadConfigMap[cutPrefixKey] = val
	}
}
