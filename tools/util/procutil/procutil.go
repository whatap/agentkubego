package procutil

import (
	"bufio"
	"github.com/whatap/golib/util/panicutil"
	whatap_config "github.com/whatap/kube/cadvisor/pkg/config"
	"github.com/whatap/kube/tools/util/logutil"
	"os"
	"path"
	"strconv"
	"strings"
)

func ParseBytes(src string) int64 {
	words := strings.Fields(src)
	if len(words) == 2 {
		val, _ := strconv.ParseInt(words[0], 10, 64)
		val *= toBytes(words[1])

		return val
	} else if len(words) == 2 {
		val, _ := strconv.ParseInt(words[0], 10, 64)

		return val
	}

	return 0
}

func toBytes(unit string) int64 {
	var ret int64
	switch unit {
	case "TB":
		ret = 1000000000000
	case "tB", "TiB":
		ret = 0x10000000000
	case "GB":
		ret = 1000000000
	case "gB", "GiB":
		ret = 0x40000000
	case "MB":
		ret = 1000000
	case "mB", "MiB":
		ret = 0x100000
	case "kB":
		ret = 1000
	case "KB", "KiB":
		ret = 0x400
	default:
		ret = 1
	}
	return ret
}

// ParseKeyValue ParseKeyValue
func ParseKeyValue(pid string, contentfile string, callback func(key string, val int64)) {

	fpath := strings.Join([]string{whatap_config.GetConfig().HostPathPrefix, "/proc", pid, contentfile}, "/")
	_, e := os.Stat(fpath)
	if e != nil {
		panicutil.Debug("ParseKeyValue", e)
		return
	}
	file, err := os.Open(fpath)
	if err != nil {
		panicutil.Debug("ParseKeyValue", e)
		return
	}
	defer func(file *os.File) {
		closeErr := file.Close()
		if closeErr != nil {
			logutil.Errorf("WHA-PU-ERR-001", "closeErr=%v", closeErr)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		if len(words) == 3 {
			if words[0][len(words[0])-1] == ':' {
				val, _ := strconv.ParseInt(words[1], 10, 64)
				val *= toBytes(words[2])
				callback(words[0][:len(words[0])-1], val)
			}
		}
	}
}

func PopulateFileValues(prefix string, filename string, callback func(tokens []string)) {
	calculated_path := path.Join(prefix, filename)

	f, err := os.Open(calculated_path)
	if err != nil {
		// fmt.Println(err)
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		if len(words) > 0 {
			callback(words)
		}
	}
}

func PopulateFileKeyValues(prefix string, filename string, sep string, callback func(tokens []string)) {
	calculated_path := path.Join(prefix, filename)

	f, err := os.Open(calculated_path)
	if err != nil {
		// fmt.Println(err)
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Split(line, sep)
		if len(words) > 0 {
			var strippedWords []string
			for _, word := range words {
				strippedWord := strings.Trim(word, " ")
				strippedWord = strings.Trim(strippedWord, " ")
				strippedWord = strings.Trim(strippedWord, "\t")
				strippedWords = append(strippedWords, strippedWord)
			}
			callback(strippedWords)
		}
	}
}
