package cgroup

import (
	"bufio"
	"fmt"
	whatap_config "github.com/whatap/kube/node/src/whatap/config"
	"github.com/whatap/kube/node/src/whatap/util/fileutil"
	"github.com/whatap/kube/node/src/whatap/util/logutil"
	"github.com/whatap/kube/node/src/whatap/util/stringutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func parseRealPath(cgroupsPath string) (ret string) {
	if strings.Contains(cgroupsPath, "slice") {
		cgroup_realpath := "kubepods.slice"
		tokens := stringutil.Tokenizer(cgroupsPath, ":")
		if whatap_config.GetConfig().Debug {
			logutil.Infof("parseRealPath", "tokens=%v", tokens)
		}
		if len(tokens) < 3 {
			return
		}
		// fmt.Println("populateCgroupKeyValue step -2")
		if strings.Contains(cgroupsPath, "besteffort") {
			cgroup_realpath = filepath.Join(cgroup_realpath, "kubepods-besteffort.slice", tokens[0], fmt.Sprint(tokens[1], "-", tokens[2], ".scope"))
		} else if strings.Contains(cgroupsPath, "burstable") {
			cgroup_realpath = filepath.Join(cgroup_realpath, "kubepods-burstable.slice", tokens[0], fmt.Sprint(tokens[1], "-", tokens[2], ".scope"))
		} else {
			cgroup_realpath = filepath.Join(cgroup_realpath, tokens[0], fmt.Sprint(tokens[1], "-", tokens[2], ".scope"))
		}
		ret = cgroup_realpath
	} else {
		ret = cgroupsPath
	}
	if whatap_config.GetConfig().Debug {
		logutil.Infof("CGROUP", "cgroup_realpath=%v, ret=%v", cgroupsPath, ret)
	}
	return
}

func populateFileKeyValue(prefix string, filename string, callback func(key string, v []int64)) (reterr error) {
	calculated_path := filepath.Join(prefix, filename)

	f, err := os.Open(calculated_path)
	if err != nil {
		// fmt.Println(err)
		reterr = err
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		if len(words) > 1 {
			var vals []int64
			for _, word := range words[1:] {
				vals = append(vals, stringutil.ToInt64(word))
			}
			callback(words[0], vals)
		}
	}

	return
}

func populateFileValues(prefix string, filename string, callback func(tokens []string)) (reterr error) {
	calculated_path := filepath.Join(prefix, filename)

	f, err := os.Open(calculated_path)
	if err != nil {
		// fmt.Println(err)
		reterr = err
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

	return
}

func populateCgroupKeyValue(prefix string, device string, cgroupsPath string, filename string, callback func(key string, v int64)) (reterr error) {
	// fmt.Println("populateCgroupKeyValue step -1 ", prefix, device, cgroupsPath, filename)
	// /rootfs/sys/fs/cgroup/cpu/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-podf22f34c9_f190_4fe7_86f8_e11b3dc4a1a0.slice/docker-57ad75d6f2741f6f39782d91282530fcad17903b8db0d4f640ba9161083963b8.scope
	// /rootfs/sys/fs/cgroup/cpu/kubepods/besteffort/podb889ec29-d166-4ba6-b98c-bc56d99f2d69/crio-a695a8eef261faf66a26a766883e2751f9e96b79b252cd9451e8b037c3024465
	// fmt.Println("populateCgroupKeyValue step -3 ",cgroup_realpath)

	cgroup_realpath := parseRealPath(cgroupsPath)
	if !fileutil.IsExists(filepath.Join(prefix, "/sys/fs/cgroup", device, cgroup_realpath, filename)) {
		cgroup_realpath = parseRealPathEx(cgroupsPath)
	}
	//if whatap_docker.CheckDockerEnabled() && whatap_docker.CheckDockerEnabled() {
	//	cgroup_realpath = cgroupsPath
	//}
	calculated_path := filepath.Join(prefix, "/sys/fs/cgroup", device, cgroup_realpath, filename)
	// fmt.Println("populateCgroupKeyValue cgroup: ", cgroupsPath)
	// fmt.Println("populateCgroupKeyValue realPath: ", cgroup_realpath)
	// fmt.Println("populateCgroupKeyValue calculated_path: ", calculated_path)

	f, err := os.Open(calculated_path)
	if err != nil {
		// fmt.Println(err)
		reterr = err
		return
	}
	// fmt.Println("populateCgroupKeyValue step -5")
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		// fmt.Println("populateCgroupKeyValue step -6 ", line)
		words := strings.Fields(line)
		// fmt.Println("populateCgroupKeyValue step -6.1 ", words)
		switch len(words) {
		case 1:
			callback("", stringutil.ToInt64(words[0]))
		case 2:
			callback(words[0], stringutil.ToInt64(words[1]))
		default:
			// fmt.Println("invalid cgroup file:", calculated_path, " content:", line)
		}
	}
	// fmt.Println("populateCgroupKeyValue step -7")
	return
}

func populateCgroupValues(prefix string, device string, cgroupsPath string, filename string, callback func(tokens []string)) (reterr error) {
	cgroup_realpath := parseRealPath(cgroupsPath)
	if !fileutil.IsExists(filepath.Join(prefix, "/sys/fs/cgroup", device, cgroup_realpath, filename)) {
		cgroup_realpath = parseRealPathEx(cgroupsPath)
	}
	calculated_path := filepath.Join(prefix, "/sys/fs/cgroup", device, cgroup_realpath, filename)
	//fmt.Println("populateCgroupValues calculated_path:",calculated_path)
	f, err := os.Open(calculated_path)
	if err != nil {
		// fmt.Println(err)
		reterr = err
		return
	}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		if len(words) > 0 {
			callback(words)
		}
	}
	return
}

type hcontparam func(name string, cgroupParent string,
	restartCount int, pid int, memoryLimit int64) error

func getParams(regEx, src string) (paramsMap map[string]string) {
	var compRegEx = regexp.MustCompile(regEx)
	match := compRegEx.FindStringSubmatch(src)
	paramsMap = make(map[string]string)

	for i, name := range compRegEx.SubexpNames() {
		if len(match) > 0 && i > 0 {
			paramsMap[name] = match[i]
		}
	}
	return
}

func getRealCgroupParent(cgroupParent string, containerId string) (ret string) {
	ret = cgroupParent
	m := getParams("(?P<prefix1>[a-zA-Z]+)\\-(?P<prefix2>[a-zA-Z]+)\\-(?P<prefix3>[a-zA-Z0-9_]+)\\.slice", cgroupParent)
	if len(m) == 3 {
		ret = filepath.Join("kubepods.slice", fmt.Sprint(m["prefix1"], "-", m["prefix2"], ".slice"), cgroupParent)
		ret = fmt.Sprint(ret, "/containerd-", containerId, ".scope")
	} else {
		ret = fmt.Sprint(ret, "/", containerId)
	}

	return
}

func CheckIfContainerIdExists() (bool, error) {
	file, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return false, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// 각 줄을 읽습니다.
		line := scanner.Text()
		// "0::/"와 일치하는지 확인
		line = strings.ReplaceAll(line, "\n", "")
		line = strings.ReplaceAll(line, "\r", "")
		line = strings.ReplaceAll(line, "\t", "")
		line = strings.TrimSpace(line)
		if line == "0::/" {
			// 일치한다면 컨테이너 ID를 찾을 수 없음
			return false, nil
		}
	}

	// 파일 읽기 중에 오류가 발생했는지 확인합니다.
	if err := scanner.Err(); err != nil {
		return false, err
	}

	// "0::/"와 일치하는 줄이 없다면 컨테이너 ID가 존재한다고 가정합니다.
	return true, nil
}
func GetMode() (mode string) {
	//[ $(stat -fc %T /sys/fs/cgroup/) = "cgroup2fs" ] && echo "unified" || ( [ -e /sys/fs/cgroup/unified/ ] && echo "hybrid" || echo "legacy")
	iterateLineFields("/proc/1/mountinfo", func(fields []string) {
		mountpoint := fields[4]
		filesystem := fields[7]

		if mountpoint == "/sys/fs/cgroup" {
			if filesystem == "cgroup2" {
				mode = "unified"
			} else {
				_, err := os.Stat("/sys/fs/cgroup/unified")
				if err != nil && os.IsNotExist(err) {
					mode = "legacy"
				} else {
					mode = "hybrid"
				}
			}
		}
	})

	return
}

func iterateLineFields(filefullpath string, callback func([]string)) (reterr error) {

	f, err := os.Open(filefullpath)
	if err != nil {
		reterr = err
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		callback(fields)
	}

	return
}

var (
	SC_CLK_TCK int64 = 100
)

func parseRealPathEx(cgroupsPath string) (ret string) {
	ret = cgroupsPath

	if strings.Contains(cgroupsPath, "slice") {
		cgroup_realpath := "system.slice"

		tokens := strings.SplitN(cgroupsPath, ":", 2)
		if len(tokens) < 2 {
			return
		}

		if strings.HasPrefix(tokens[1], "cri-containerd") {
			cgroup_realpath = filepath.Join(cgroup_realpath, "containerd.service", cgroupsPath)
		} else {
			tokens := strings.SplitN(cgroupsPath, ":", 2)
			if len(tokens) < 2 {
				return
			}
			cgroup_realpath = filepath.Join(cgroup_realpath, fmt.Sprintf("%s.service", tokens[0]), cgroupsPath)
		}

		ret = cgroup_realpath
	}

	return
}
