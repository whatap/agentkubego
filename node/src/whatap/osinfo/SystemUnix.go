//go:build darwin || freebsd
// +build darwin freebsd

package osinfo

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"os/user"
	"runtime"
)

// GetCPUSocket get cpu socket
func GetCPUSocket() (int, error) {
	return runtime.NumCPU(), nil
}

func GetCPUType() (string, error) {
	cpuinfo, err := cpu.Info()
	if err != nil {
		return "", err
	}

	if len(cpuinfo) > 0 {
		return cpuinfo[0].ModelName, nil
	}
	return "", fmt.Errorf("CPU Info not available")
}

func GetMemorySize() (int64, error) {
	vminfo, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}

	return int64(vminfo.Total), nil
}

var username_cache = make(map[int]string)

func getUserNameById(uid int) (username string, err error) {
	if _, ok := username_cache[uid]; !ok {
		userFound, lookuperr := user.LookupId(fmt.Sprint(uid))
		if lookuperr != nil {
			//username = ""
			err = lookuperr
			return
		}
		username = userFound.Username
		// fmt.Println(uid, username)
		username_cache[uid] = username
	} else {
		username = username_cache[uid]
		err = nil
	}

	return
}
