package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/magiconair/properties"
	"github.com/whatap/golib/util/dateutil"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/golib/util/stringutil"
	"github.com/whatap/kube/tools/util/logutil"
)

type Config struct {
	Port                                   string
	Cycle                                  int32
	KubeConfigPath                         string
	HostPathPrefix                         string
	CollectNfsDiskEnabled                  bool
	PathSysBlock                           string
	Debug                                  bool
	KubeMasterUrl                          string
	Test                                   bool
	TestContainerId                        string
	MasterAgentHost                        string
	MasterAgentPort                        int32
	ConfBaseAgentHost                      string
	ConfBaseAgentPort                      int32
	WhatapJavaAgentPath                    string
	WhatapPythonAgentPath                  string
	WhatapPhpAgentPath                     string
	WhatapGoAgentPath                      string
	WhatapDotnetAgentPath                  string
	WhatapExecutableJavaPath               string
	InspectWhatapAgentPathFromProc         bool
	CollectVolumeDetailEnabled             bool
	InjectContainerIdToApmAgentEnabled     bool
	Version                                string
	LogSysOut                              bool
	UseCachedMountPointEnabled             bool
	CollectProcessPssEnabled               bool
	CollectProcessPssTargetList            []string
	CollectKubeNodeProcessMetricEnabled    bool
	CollectKubeNodeProcessMetricTargetList []string
	CollectProcessFD                       bool
	CollectProcessIO                       bool
	CgroupVersion                          string
	IsRuntimeDocker                        bool
	IsRuntimeContainerd                    bool
	IsRuntimeCrio                          bool
	Runtime                                string
}

func checkDockerEnabled() bool {
	fi, err := os.Stat("/var/run/docker.sock")
	if err != nil && os.IsNotExist(err) {
		return false
	}

	if fi.Mode().IsDir() {
		return false
	}

	return true
}

func checkContainerdEnabled() bool {
	fi, err := os.Stat("/run/containerd/containerd.sock")
	if err != nil && os.IsNotExist(err) {
		return false
	}

	if fi.Mode().IsDir() {
		return false
	}

	return true
}

func checkCrioEnabled() bool {
	fi, err := os.Stat("/var/run/crio/crio.sock")
	if err != nil && os.IsNotExist(err) {
		return false
	}

	if fi.Mode().IsDir() {
		return false
	}

	return true
}

func init() {
	printWhatap := fmt.Sprint("\n" +
		" _      ____       ______WHATAP-KUBER-AGENT\n" +
		"| | /| / / /  ___ /_  __/__ ____\n" +
		"| |/ |/ / _ \\/ _ `// / / _ `/ _ \\\n" +
		"|__/|__/_//_/\\_,_//_/  \\_,_/ .__/\n" +
		"                          /_/\n" +
		"Just Tap, Always Monitoring\n")
	fmt.Print(printWhatap)
	whatapConfig := GetConfig()
	whatapConfig.IsRuntimeDocker = checkDockerEnabled()
	whatapConfig.IsRuntimeContainerd = checkContainerdEnabled()
	whatapConfig.IsRuntimeCrio = checkCrioEnabled()
	switch {
	case whatapConfig.IsRuntimeDocker:
		whatapConfig.Runtime = "docker"
	case whatapConfig.IsRuntimeContainerd:
		whatapConfig.Runtime = "containerd"
	case whatapConfig.IsRuntimeCrio:
		whatapConfig.Runtime = "crio"
	default:
		whatapConfig.Runtime = "unknown"
	}
	// 구성 정보 출력
	fmt.Printf("-DEBUG: %v\n", whatapConfig.Debug)
	fmt.Printf("-Runtime: %v\n", whatapConfig.Runtime)
	fmt.Printf("-HostPathPrefix: %v\n", whatapConfig.HostPathPrefix)
	fmt.Printf("-KubeConfigPath: %v\n", whatapConfig.KubeConfigPath)
	fmt.Printf("-KubeMasterUrl: %v\n", whatapConfig.KubeMasterUrl)
	fmt.Printf("-MasterAgentHost: %v\n", whatapConfig.MasterAgentHost)
	fmt.Printf("-MasterAgentPort: %v\n", whatapConfig.MasterAgentPort)
	fmt.Printf("-ConfBaseAgentHost: %v\n", whatapConfig.ConfBaseAgentHost)
	fmt.Printf("-ConfBaseAgentPort: %v\n", whatapConfig.ConfBaseAgentPort)
	fmt.Printf("-VERSION: %v\n", whatapConfig.Version)
	fmt.Printf("-PORT: %v\n", whatapConfig.Port)
	fmt.Printf("-CYCLE: %vs\n", whatapConfig.Cycle)
	fmt.Printf("-ConfFilePath: %v\n", GetConfFilePath())
	fmt.Printf("-Test: %v\n", whatapConfig.Test)
	fmt.Printf("-TestContainerId: %v\n", whatapConfig.TestContainerId)
	fmt.Printf("-LogSysOut: %v\n", whatapConfig.LogSysOut)
	fmt.Printf("-InjectContainerIdToApmAgentEnabled: %v\n", whatapConfig.InjectContainerIdToApmAgentEnabled)
	fmt.Printf("-UseCachedMountPointEnabled: %v\n", whatapConfig.UseCachedMountPointEnabled)
	fmt.Printf("-CollectVolumeDetailEnabled: %v\n", whatapConfig.CollectVolumeDetailEnabled)
	fmt.Printf("-CollectNfsDiskEnabled: %v\n", whatapConfig.CollectNfsDiskEnabled)
	fmt.Printf("-CollectProcessIO: %v\n", whatapConfig.CollectProcessIO)
	fmt.Printf("-CollectProcessFD: %v\n", whatapConfig.CollectProcessFD)
	fmt.Printf("-CollectKubeNodeProcessMetricEnabled: %v\n", whatapConfig.CollectKubeNodeProcessMetricEnabled)
	fmt.Printf("-CollectKubeNodeProcessMetricTargetList: %v\n", whatapConfig.CollectKubeNodeProcessMetricTargetList)
	fmt.Printf("-InspectWhatapAgentPathFromProc: %v\n", whatapConfig.InspectWhatapAgentPathFromProc)
	fmt.Printf("-DefaultJavaAgentPath: %v\n", whatapConfig.WhatapJavaAgentPath)
	fmt.Printf("-DefaultPythonAgentPath: %v\n", whatapConfig.WhatapPythonAgentPath)
	fmt.Printf("-DefaultPhpAgentPath: %v\n", whatapConfig.WhatapPhpAgentPath)
	fmt.Printf("-DefaultGoAgentPath: %v\n", whatapConfig.WhatapGoAgentPath)
	fmt.Printf("-DefaultDotnetAgentPath: %v\n", whatapConfig.WhatapDotnetAgentPath)
	fmt.Printf("-ExecutableJavaPath: %v\n", whatapConfig.WhatapExecutableJavaPath)
	fmt.Printf("-IsRuntimeDocker: %v\n", whatapConfig.IsRuntimeDocker)
	fmt.Printf("-IsRuntimeContainerd: %v\n", whatapConfig.IsRuntimeContainerd)
	fmt.Printf("-IsRuntimeCrio: %v\n", whatapConfig.IsRuntimeCrio)
	fmt.Println("===========================================================================================")
}

var conf *Config = nil
var mutex = sync.Mutex{}
var prop *properties.Properties = nil
var AppType int16 = 3

// whatap.server.host 같은 경우 dot(.) 문자 때문에 환경변수로 인식이 안되는 경우 지정 이름 사용.
var envKeys = map[string]string{
	"accesskey":          "WHATAP_ACCESSKEY",
	"license":            "WHATAP_LICENSE",
	"whatap.server.host": "WHATAP_SERVER_HOST",
	"whatap.server.port": "WHATAP_SERVER_PORT",
}

var logLevelMap = map[string]int{
	"ERROR": logutil.LOG_LEVEL_ERROR,
	"WARN":  logutil.LOG_LEVEL_WARN,
	"INFO":  logutil.LOG_LEVEL_INFO,
	"DEBUG": logutil.LOG_LEVEL_DEBUG,
}

func GetConfig() *Config {

	mutex.Lock()
	defer mutex.Unlock()
	if conf != nil {
		return conf
	}
	conf = new(Config)
	//init
	prop = properties.NewProperties()
	apply()

	reload()
	go run()

	return conf
}
func run() {
	for {
		// DEBUG goroutine log
		//logutil.Println("Config.run()")

		time.Sleep(3000 * time.Millisecond)
		reload()
	}
}

var last_file_time int64 = -1
var last_check int64 = 0

func reload() {
	// 종료 되지 않도록  Recover
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA211 Recover", r) //, string(debug.Stack()))
		}
	}()

	now := dateutil.Now()
	if now < last_check+3000 {
		return
	}
	last_check = now
	path := GetConfFile()

	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		if last_file_time == -1 {
			logutil.Println("WA212", "fail to load license file")
			if f, err := os.Create(path); err != nil {
				logutil.Println("WA212-01", "create file error ", err)
				return
			} else {
				logutil.Println("WA212-02", "create file path ", f.Name())
			}
		} else if last_file_time == 0 {
			return
		}
		last_file_time = 0
		prop = properties.NewProperties()
		apply()
		logutil.Println("WA213", " Reload Config: ", GetConfFile())
		return
	}

	new_time := stat.ModTime().Unix()
	if last_file_time == new_time {
		return
	}
	last_file_time = new_time
	prop = properties.MustLoadFile(path, properties.UTF8)
	apply()

	// Observer run
	RunConfObserver()
}

func GetConfFile() string {
	home := GetWhatapHome()
	// config 파일이 WHATAP_HOME 과 다른 경로에 있을 경우 설정.
	confHome := os.Getenv("WHATAP_CONFIG_HOME")
	if confHome != "" {
		home = confHome
	}

	confName := os.Getenv("WHATAP_CONFIG")
	if confName == "" {
		confName = "whatap.conf"
	}

	return filepath.Join(home, confName)
}
func GetConfFilePath() string {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current working directory: %v\n", err)
		return ""
	}
	confFile := GetConfFile()

	return filepath.Join(wd, confFile)
}
func GetWhatapHome() string {
	home := os.Getenv("WHATAP_HOME")
	if home == "" {
		home = "."
	}
	return home
}

func apply() {
	// Recover로 구문 예외 처리
	func() {
		defer func() {
			if r := recover(); r != nil {
				logutil.Println("WA217", " Recover ", r)
			}
		}()
		conf.Port = getValueDef("port", "6801")
		conf.Cycle = getInt("cycle", 5)
		conf.KubeConfigPath = getValueDef("kube_config_path", "")
		conf.HostPathPrefix = getValueDef("host_path_prefix", "/rootfs")
		conf.PathSysBlock = getValueDef("path_sys_block", "/sys/block")
		conf.Debug = getBoolean("debug", false)
		if conf.Debug {
			logutil.SetLevel(logLevelMap["DEBUG"])
		} else {
			logutil.SetLevel(logLevelMap["INFO"])
		}
		conf.KubeMasterUrl = getValueDef("kube_master_url", "")
		conf.Test = getBoolean("test", false)
		conf.TestContainerId = getValueDef("test_container_id", "")
		conf.MasterAgentHost = getValueDef("master_agent_host", "whatap-master-agent.whatap-monitoring")
		conf.MasterAgentPort = getInt("master_agent_port", 6600)
		conf.ConfBaseAgentHost = getValueDef("confbase_agent_host", "whatap-master-agent.whatap-monitoring")
		conf.ConfBaseAgentPort = getInt("confbase_agent_port", 6800)
		conf.CollectVolumeDetailEnabled = getBoolean("collect_volume_detail_enabled", true)
		conf.CollectNfsDiskEnabled = getBoolean("collect_nfs_disk_enabled", true)
		conf.InjectContainerIdToApmAgentEnabled = getBoolean("inject_container_id_to_apm_agent_enabled", true)
		conf.WhatapJavaAgentPath = getValueDef("whatap_java_agent_path", "")
		conf.WhatapPythonAgentPath = getValueDef("whatap_python_agent_path", "")
		conf.WhatapPhpAgentPath = getValueDef("whatap_php_agent_path", "")
		conf.WhatapGoAgentPath = getValueDef("whatap_go_agent_path", "")
		conf.WhatapDotnetAgentPath = getValueDef("whatap_dotnet_agent_path", "")
		conf.WhatapExecutableJavaPath = getValueDef("whatap_executable_java_path", "")
		conf.InspectWhatapAgentPathFromProc = getBoolean("inspect_whatap_agent_path_from_proc", true)
		conf.UseCachedMountPointEnabled = getBoolean("use_cached_mount_info_enabled", true)
		conf.CollectProcessPssEnabled = getBoolean("collect_process_pss_enabled", true)
		conf.CollectProcessPssTargetList = strings.Split(getValueDef("collect_process_pss_target_list", "httpd,apache,apache2,kubelet,containerd-shim,containerd,docker,dockerd,crio"), ",")
		conf.CollectKubeNodeProcessMetricEnabled = getBoolean("collect_kube_node_process_metric_enabled", true)
		conf.CollectKubeNodeProcessMetricTargetList = strings.Split(getValueDef("collect_kube_node_process_metric_target_list", "kubelet,containerd,dockerd,crio,coredns,kube-proxy,aws-k8s-agent,kube-apiserver,etcd,kube-controller,kube-scheduler"), ",")
		conf.CollectProcessFD = getBoolean("collect_process_fd", true)
		conf.CollectProcessIO = getBoolean("collect_process_io", false)
		conf.CgroupVersion = getValueDef("cgroup_version", "")
		// sysOut 설정 log_sys_out 설정을 키면 stdOut 과 file 에 동시에 로깅을 남김
		oldLogSysOut := conf.LogSysOut
		newLogSysOut := getBoolean("log_sys_out", true)
		if oldLogSysOut != newLogSysOut {
			conf.LogSysOut = newLogSysOut
			logutil.SetLogSysOut(newLogSysOut)
		}

		conf.Version = getValueDef("version", "")
	}()
}

func GetValue(key string) string { return getValue(key) }
func getValue(key string) string {
	envVal := os.Getenv(key)
	if envVal == "" {
		// 동일한 이름의 env 값이 없으면, 지정된 env key 이름으로 값을 가져옴.
		if v, ok := envKeys[key]; ok {
			envVal = os.Getenv(v)
		}
	}
	value, ok := prop.Get(key)
	if ok == false {
		return strings.TrimSpace(envVal)
	}

	return strings.TrimSpace(value)
}
func GetValueDef(key, def string) string { return getValueDef(key, def) }
func getValueDef(key string, def string) string {
	v := getValue(key)

	if v == "" {
		return def
	}

	return v
}
func GetBoolean(key string, def bool) bool {
	return getBoolean(key, def)
}
func getBoolean(key string, def bool) bool {
	v := getValue(key)
	if v == "" {
		return def
	}
	value, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return value
}

func GetInt(key string, def int) int32 {
	return getInt(key, def)
}
func getInt(key string, def int) int32 {
	v := getValue(key)
	if v == "" {
		return int32(def)
	}
	value, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return int32(def)
	}
	return int32(value)
}

func GetIntSet(key, defaultValue, deli string) *hmap.IntSet {
	set := hmap.NewIntSet()
	vv := stringutil.Tokenizer(GetValueDef(key, defaultValue), deli)
	if vv != nil {
		for _, x := range vv {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Continue
					}
				}()
				if xx, err := strconv.Atoi(stringutil.TrimEmpty(x)); err != nil {
					set.Put(int32(xx))
				}
			}()
		}
	}
	return set
}

func GetStringHashSet(key, defaultValue, deli string) *hmap.IntSet {
	set := hmap.NewIntSet()
	vv := stringutil.Tokenizer(GetValueDef(key, defaultValue), deli)
	if vv != nil {
		for _, x := range vv {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Continue
					}
				}()
				xx := hash.HashStr(stringutil.TrimEmpty(x))
				set.Put(xx)
			}()
		}
	}
	return set
}

func GetStringHashCodeSet(key, defaultValue, deli string) *hmap.IntSet {
	set := hmap.NewIntSet()
	vv := stringutil.Tokenizer(GetValueDef(key, defaultValue), deli)
	if vv != nil {
		for _, x := range vv {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Continue
					}
				}()
				xx := stringutil.HashCode(stringutil.TrimEmpty(x))
				set.Put(int32(xx))
			}()
		}
	}
	return set
}
func GetLong(key string, def int64) int64 {
	return getLong(key, def)
}
func getLong(key string, def int64) int64 {
	v := getValue(key)
	if v == "" {
		return def
	}
	value, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return value
}
func GetStringArray(key string, deli string) []string {
	return getStringArray(key, deli)
}
func getStringArray(key string, deli string) []string {
	v := getValue(key)
	if v == "" {
		return []string{}
	}
	tokens := stringutil.Tokenizer(v, deli)
	// trim Space
	trimTokens := make([]string, 0)
	for _, v := range tokens {
		trimTokens = append(trimTokens, strings.TrimSpace(v))
	}
	return trimTokens
}

func getStringArrayDef(key string, deli string, def string) []string {
	v := getValueDef(key, def)
	if v == "" {
		return []string{}
	}
	tokens := stringutil.Tokenizer(v, deli)
	// trim Space
	trimTokens := make([]string, 0)
	for _, v := range tokens {
		trimTokens = append(trimTokens, strings.TrimSpace(v))
	}
	return trimTokens
}

func getFloat(key string, def float32) float32 {
	v := getValue(key)
	if v == "" {
		return float32(def)
	}
	value, err := strconv.ParseFloat(v, 32)
	if err != nil {
		return float32(def)
	}
	return float32(value)
}

func SetValues(keyValues *map[string]string) {
	path := GetConfFile()
	props := properties.MustLoadFile(path, properties.UTF8)
	for key, value := range *keyValues {
		props.Set(key, value)
	}

	line := ""
	if f, err := os.OpenFile(path, os.O_RDWR, 0644); err != nil {
		logutil.Println("WA215", " Error ", err)
		return
	} else {
		defer f.Close()

		r := bufio.NewReader(f)
		new_keys := props.Keys()
		old_keys := map[string]bool{}
		for {
			data, _, err := r.ReadLine()
			if err != nil { // new key
				for _, key := range new_keys {
					if old_keys[key] {
						continue
					}
					match, _ := regexp.MatchString("^\\w", key)
					if match {
						value, _ := props.Get(key)
						if strings.TrimSpace(value) != "" {
							tmp := strings.Replace(value, "\\\\", "\\", -1)
							tmp = strings.Replace(tmp, "\\", "\\\\", -1)
							line += fmt.Sprintf("%s=%s\n", key, tmp)
						}
					}
				}
				break
			}
			if strings.Index(string(data), "=") == -1 {
				line += fmt.Sprintf("%s\n", string(data))
				//io.WriteString(f, line)
			} else {
				datas := strings.Split(string(data), "=")
				key := strings.Trim(datas[0], " ")
				value := strings.Trim(datas[1], " ")
				old_keys[key] = true

				match, _ := regexp.MatchString("^\\w", key)
				if match {
					value, _ = props.Get(key)
				}
				// value 가 없는 경우 항목 추가 안함(삭제)
				if strings.TrimSpace(value) != "" {
					tmp := strings.Replace(value, "\\\\", "\\", -1)
					tmp = strings.Replace(tmp, "\\", "\\\\", -1)

					line += fmt.Sprintf("%s=%s\n", key, tmp)
				}
				//io.WriteString(f, line)
			}
		}
	}

	if f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644); err != nil {
		logutil.Println("WA216", " Error ", err)
		return
	} else {
		defer f.Close()
		io.WriteString(f, line)

		// flush
		f.Sync()
	}
}
func ToString() string {
	sb := stringutil.NewStringBuffer()
	if prop == nil {
		return ""
	}
	for _, key := range prop.Keys() {
		if v, ok := prop.Get(key); ok {
			sb.Append(key).Append("=").AppendLine(v)
		}
	}

	sb.Append("Port").Append("=").AppendLine(fmt.Sprintf("%v", conf.Port))
	sb.Append("Cycle").Append("=").AppendLine(fmt.Sprintf("%v", conf.Cycle))
	sb.Append("KubeConfigPath").Append("=").AppendLine(fmt.Sprintf("%v", conf.KubeConfigPath))
	sb.Append("Debug").Append("=").AppendLine(fmt.Sprintf("%v", conf.Debug))
	sb.Append("KuberMasterUrl").Append("=").AppendLine(fmt.Sprintf("%v", conf.KubeMasterUrl))
	sb.Append("TestContainerId").Append("=").AppendLine(fmt.Sprintf("%v", conf.TestContainerId))
	sb.Append("MasterAgentHost").Append("=").AppendLine(fmt.Sprintf("%v", conf.MasterAgentHost))
	sb.Append("MasterAgentPort").Append("=").AppendLine(fmt.Sprintf("%v", conf.MasterAgentPort))
	sb.Append("ConfBaseAgentHost").Append("=").AppendLine(fmt.Sprintf("%v", conf.ConfBaseAgentHost))
	sb.Append("ConfBaseAgentPort").Append("=").AppendLine(fmt.Sprintf("%v", conf.ConfBaseAgentPort))
	sb.Append("CollectVolumeDetailEnabled").Append("=").AppendLine(fmt.Sprintf("%v", conf.CollectVolumeDetailEnabled))
	sb.Append("CollectNfsDiskEnabled").Append("=").AppendLine(fmt.Sprintf("%v", conf.CollectNfsDiskEnabled))
	sb.Append("InjectContainerIdToApmAgentEnabled").Append("=").AppendLine(fmt.Sprintf("%v", conf.ConfBaseAgentPort))
	sb.Append("WhatapJavaAgentPath").Append("=").AppendLine(fmt.Sprintf("%v", conf.WhatapJavaAgentPath))
	sb.Append("WhatapPythonAgentPath").Append("=").AppendLine(fmt.Sprintf("%v", conf.WhatapPythonAgentPath))
	sb.Append("WhatapPhpAgentPath").Append("=").AppendLine(fmt.Sprintf("%v", conf.WhatapPhpAgentPath))
	sb.Append("WhatapGoAgentPath").Append("=").AppendLine(fmt.Sprintf("%v", conf.WhatapGoAgentPath))
	sb.Append("WhatapDotnetAgentPath").Append("=").AppendLine(fmt.Sprintf("%v", conf.WhatapDotnetAgentPath))
	sb.Append("WhatapExecutableJavaPath").Append("=").AppendLine(fmt.Sprintf("%v", conf.WhatapExecutableJavaPath))
	sb.Append("InspectWhatapAgentPathFromProc").Append("=").AppendLine(fmt.Sprintf("%v", conf.InspectWhatapAgentPathFromProc))
	sb.Append("UseCachedMountPointEnabled").Append("=").AppendLine(fmt.Sprintf("%v", conf.UseCachedMountPointEnabled))
	return sb.ToString()
}

func SearchKey(keyPrefix string) *map[string]string {
	keyValues := map[string]string{}
	for _, key := range prop.Keys() {
		if strings.HasPrefix(key, keyPrefix) {
			if v, ok := prop.Get(key); ok {
				keyValues[key] = v
			}
		}
	}

	return &keyValues
}

func FilterPrefix(keyPrefix string) map[string]string {
	keyValues := make(map[string]string)
	pp := prop.FilterPrefix(keyPrefix)
	for _, key := range pp.Keys() {
		keyValues[key] = pp.GetString(key, "")
	}
	return keyValues
}

func cutOut(val, delim string) string {
	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA217", " Recover ", r)
		}
	}()
	if val == "" {
		return val
	}
	x := strings.LastIndex(val, delim)
	if x <= 0 {
		return ""
	}
	//return val.substring(0, x);
	return val[0:x]

}

func toHashSet(key, def string) *hmap.IntSet {
	set := hmap.NewIntSet()
	vv := strings.Split(getValueDef(key, def), ",")
	if vv != nil {
		for _, x := range vv {
			func() {
				defer func() {
					if r := recover(); r != nil {
						logutil.Infoln("WA218", " Recover ", r)
					}
				}()

				x = strings.TrimSpace(x)
				if len(x) > 0 {
					xx := hash.HashStr(x)
					set.Put(xx)
				}
			}()
		}
	}
	return set
}

func toStringSet(key, def string) *hmap.StringSet {
	set := hmap.NewStringSet()
	vv := strings.Split(getValueDef(key, def), ",")
	if vv != nil {
		for _, x := range vv {
			func() {
				defer func() {
					if r := recover(); r != nil {
						logutil.Infoln("WA219", " Recover ", r)
					}
				}()
				x = strings.TrimSpace(x)
				if len(x) > 0 {
					set.Put(x)
				}
			}()
		}
	}
	return set
}
