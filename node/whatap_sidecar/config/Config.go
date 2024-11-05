package config

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/whatap/golib/lang/value"
	"github.com/whatap/golib/util/hash"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/lang/license"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/dateutil"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/properties"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/stringutil"
)

const ()

var (
	PROTECTED_KEYS []string = []string{"createdtime", "oid"}
	observers      []func(*Config)
)

type Config struct {
	Enabled        bool
	License        string
	WhatapHost     []string
	WhatapPort     int32
	CypherLevel    int32
	EncryptLevel   int32
	Okind          string
	ONAME          string
	ONODE          string
	PCODE          int64
	OID            int32
	LicenseHash64  int64
	ConfbaseHost   string
	ConfbasePort   int32
	LogsinkEnabled bool
}

var (
	LF                               = "\n"
	conf      *Config                = nil
	mutex                            = sync.Mutex{}
	prop      *properties.Properties = nil
	saveMutex                        = sync.Mutex{}
)

func GetConfig() *Config {
	mutex.Lock()
	defer mutex.Unlock()
	if conf != nil {
		return conf
	}

	conf = new(Config)
	initWhatapConf()
	reload()
	go run()

	return conf
}
func run() {
	for {
		time.Sleep(3000 * time.Millisecond)
		reload()
	}
}

var last_file_time int64 = -1
var last_check int64 = 0

func reload() {
	now := dateutil.Now()
	if now < last_check+3000 {
		return
	}

	last_check = now

	path := GetConfFile()

	stat, err := os.Stat(path)
	if stat == nil || os.IsNotExist(err) {
		if err != nil {
			log.Println("Config File stat failed ", path, err.Error())
		} else {
			log.Println("Config File stat failed ", path)
		}

		if last_file_time == 0 {
			return
		}
		if prop == nil {
			last_file_time = 0
			prop = properties.NewProperties()
			apply()
		}

		return
	}
	if err != nil {
		log.Println("Config", err)
		return
	}

	new_time := stat.ModTime().Unix()
	if last_file_time == new_time {

		return
	}

	last_file_time = new_time
	prop = properties.MustLoadFile(path, properties.UTF8)

	apply()

}

func initWhatapConf() {
	path := GetConfFile()
	stat, err := os.Stat(path)
	if stat == nil || os.IsNotExist(err) {
		f, ferr := os.Create(path)
		if ferr == nil {
			f.Close()
		} else {
			log.Println("Config init ", ferr)
		}
	}
}

func GetConfFile() string {
	home := os.Getenv("WHATAP_HOME")
	if home == "" {
		home = "."
	}
	confName := os.Getenv("WHATAP_CONFIG")

	if confName == "" {
		confName = "whatap.conf"
	}
	return filepath.Join(home, confName)
}

func GetWhatapHome() string {
	home := os.Getenv("WHATAP_HOME")
	if home == "" {
		home = "."
	}

	return home
}
func apply() {
	conf.Enabled = getBoolean("enabled", true)
	if !conf.Enabled {
		log.Fatal("Received Agent Status Closed redeived. Shutting down...")
		os.Exit(0)
	}
	conf.License = getValueDefault("license", os.Getenv("WHATAP_LICENSE"))
	conf.WhatapHost = getStringArrayDefault("whatap.server.host", "/:,", os.Getenv("WHATAP_HOST"))
	conf.WhatapPort = getIntDefault("whatap.server.port", os.Getenv("WHATAP_PORT"), 6600)
	conf.CypherLevel = getIntDefault("cypher_level", os.Getenv("WHATAP_CYPHER_LEVEL"), 128)
	conf.EncryptLevel = getIntDefault("encrypt_level", os.Getenv("WHATAP_ENCRYPT_LEVEL"), 2)

	conf.ONODE = getValueDefault("onode", os.Getenv("NODE_NAME"))
	conf.ONAME = getValueDefault("oname", conf.ONODE)
	conf.OID = hash.HashStr(conf.ONODE)
	conf.LicenseHash64 = hash.Hash64Str(conf.License)
	conf.PCODE, _ = license.Parse(conf.License)

	conf.Okind = getValueDefault("okind", os.Getenv("WHATAP_OKIND"))
	conf.ConfbaseHost = getValueDefault("confbase_agent_host", os.Getenv("WHATAP_CONFBASE_AGENT_HOST"))
	if len(conf.ConfbaseHost) == 0 {
		conf.ConfbaseHost = "whatap-master-agent"
	}
	conf.ConfbasePort = getIntDefault("confbase_agent_port", os.Getenv("WHATAP_CONFBASE_AGENT_PORT"), 6800)
	conf.LogsinkEnabled = getBoolean("logsink_enabled", false)

	for _, observer := range observers {
		observer(conf)
	}
}

func getIntDefault(k string, kdef string, idef int32) int32 {
	v := getValueDefault(k, kdef)
	if v == "" {
		return idef
	}
	value, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return int32(idef)
	}
	return int32(value)
}

func getValue(key string) string {
	value, ok := prop.Get(key)
	if ok == false {
		value = os.Getenv(key)
	}
	return strings.TrimSpace(value)
}

func getValueDefault(key string, def string) string {
	value, ok := prop.Get(key)
	if ok == false {
		return def
	}
	return strings.TrimSpace(value)
}

func setValue(key string, v string) {
	prop.Set(key, v)
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
func getStringArray(key string, deli string) []string {
	v := getValue(key)
	if v == "" {
		return []string{}
	}
	tokens := stringutil.Tokenizer(v, deli)
	return tokens
}
func getStringArrayDefault(key string, deli string, defaultVal string) []string {
	v := getValueDefault(key, defaultVal)
	if v == "" {
		return []string{}
	}
	tokens := stringutil.Tokenizer(v, deli)
	return tokens
}

func WhatapHome() string {
	home := os.Getenv("WHATAP_HOME")
	if home == "" {
		home = "."
	}
	return home
}

func ClearKeys(prop *SimpleMap, keyprefix string) {
	var keys []string
	for k, _ := range *prop {
		if strings.HasPrefix(k, keyprefix) {
			keys = append(keys, k)
		}
	}
	for _, k := range keys {
		delete(*prop, k)
	}
}

type SimpleMap map[string]string

// ReadConfigSimple read config simple
func ReadConfigSimple(filename string) (*SimpleMap, error) {
	return readConfigSimple(filename)
}

func readConfigSimple(filename string) (*SimpleMap, error) {
	c := SimpleMap{}
	if len(filename) == 0 {
		return &c, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')
		if equal := strings.Index(line, "="); equal >= 0 {
			if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
				value := ""
				if len(line) > equal {
					value = strings.TrimSpace(line[equal+1:])
				}
				c[key] = value
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return &c, nil
}

func SetValues(keyValues *map[string]string) {
	saveMutex.Lock()
	defer saveMutex.Unlock()

	configpath := GetConfFile()
	propmap, err := readConfigSimple(configpath)
	if err != nil || len(*propmap) < 1 {
		log.Println("Config.SetValues err:", err)
		return
	}
	for _, protectedKey := range PROTECTED_KEYS {
		if _, ok := (*keyValues)[protectedKey]; ok {
			delete(*keyValues, protectedKey)
		}
	}

	for key, value := range *keyValues {
		(*propmap)[key] = value
	}

	// f, err := os.OpenFile(configpath, os.O_WRONLY|os.O_TRUNC, 0644)
	// if err != nil {
	// 	fmt.Println(keyValues, err)
	// }
	f := bytes.NewBuffer(nil)
	orderedkeys := []string{"license", "whatap.server.host", "createdtime"}
	for _, k := range orderedkeys {
		v := (*propmap)[k]
		line := fmt.Sprintf("%s=%s%s", k, v, LF)
		io.WriteString(f, line)
		delete((*propmap), k)
	}
	var keys []string
	for k := range *propmap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := (*propmap)[k]
		line := fmt.Sprintf("%s=%s%s", k, v, LF)
		io.WriteString(f, line)
	}

	buflen := f.Len()

	if buflen < 1 {
		return
	}

	tempfilepath := fmt.Sprint(configpath, ".bak")

	err = ioutil.WriteFile(tempfilepath, f.Bytes(), 0644)
	if err != nil {
		return
	}
	stat, err := os.Stat(tempfilepath)
	if stat == nil || os.IsNotExist(err) {
		return
	}
	if stat.Size() != int64(buflen) {
		return
	}

	err = writeFile(configpath, f.Bytes(), 0644)
	if err == nil {
		os.Remove(tempfilepath)
	}

}

func writeFile(filename string, buf []byte, perm os.FileMode) (writeFileErr error) {
	if len(filename) < 0 || buf == nil || len(buf) < 1 {
		writeFileErr = fmt.Errorf("writeFile invalid param", filename, buf)
		return
	}
	f, err := os.OpenFile(filename, os.O_RDWR, perm)
	if err != nil {
		writeFileErr = err
		return
	}
	defer f.Close()

	bufsize := len(buf)
	nbytesleft := bufsize
	for nbytesleft > 0 {
		nbytesthistime, err := f.Write(buf[bufsize-nbytesleft:])
		if err != nil {
			writeFileErr = err
			return
		}
		nbytesleft -= nbytesthistime
	}
	writeFileErr = f.Truncate(int64(bufsize))

	return
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

func SearchKeyBoolean(keyPrefix string) *map[string]bool {
	keyValues := map[string]bool{}
	prefixlen := len(keyPrefix)
	for _, key := range prop.Keys() {
		if strings.HasPrefix(key, keyPrefix) {
			if v, ok := prop.Get(key); ok {
				value, err := strconv.ParseBool(v)
				if err == nil {

					keyValues[key[prefixlen:]] = value
				}
			}
		}
	}

	return &keyValues
}

func SearchKeys(keyPrefix string, callback func(string)) {
	for _, key := range prop.Keys() {
		if strings.HasPrefix(key, keyPrefix) {
			if v, ok := prop.Get(key); ok {
				callback(v)
			}
		}
	}
}

func GetAllPropertiesMapValue() *value.MapValue {
	if prop != nil {
		m := value.NewMapValue()
		for _, k := range prop.Keys() {
			v, ok := prop.Get(k)
			if ok {
				m.Put(k, value.NewTextValue(v))
			}
		}
		return m
	} else {
		return nil
	}
}

func AddObserver(observer func(*Config)) {
	mutex.Lock()
	defer mutex.Unlock()

	if observer != nil {
		observers = append(observers, observer)
	}
	if conf != nil {
		observer(conf)
	}
}
