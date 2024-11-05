// +build linux

package osinfo

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func filterRelease(filename string, prefixtoken string) string {
	var pkgrelease string = ""
	b, err := ioutil.ReadFile(filename)
	if err == nil {
		lsbrelease := string(b)
		for _, token := range strings.Split(lsbrelease, "\n") {
			if strings.HasPrefix(token, prefixtoken) {
				token = token[len(prefixtoken):]
				token = strings.Replace(token, "\"", "", -1)
				pkgrelease = token

			}
		}
	}

	return pkgrelease
}

//GetOsDetail GetOsDetail
func GetOsRelease() string {
	var pkgrelease string
	if exists("/etc/lsb-release") {
		pkgrelease = filterRelease("/etc/lsb-release", "DISTRIB_DESCRIPTION=")
	}

	if len(pkgrelease) < 1 && exists("/etc/centos-release") {
		b, err := ioutil.ReadFile("/etc/centos-release")
		if err == nil {
			pkgrelease = string(b)
		}
	}
	if len(pkgrelease) < 1 && exists("/etc/os-release") {
		pkgrelease = filterRelease("/etc/os-release", "PRETTY_NAME=")
		if len(pkgrelease) < 1 {
			pkgrelease = filterRelease("/etc/os-release", "NAME=")
		}
	}
	if len(pkgrelease) < 1 && exists("/etc/redhat-release") {
		b, err := ioutil.ReadFile("/etc/redhat-release")
		if err == nil {
			pkgrelease = string(b)
		}
	}
	if len(pkgrelease) < 1 {
		pkgrelease = "unknown distribution"
	}
	return pkgrelease
}

func exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

//Getuptime Getuptime
func GetUptime() (int64, error) {
	sysinfo := syscall.Sysinfo_t{}

	if err := syscall.Sysinfo(&sysinfo); err != nil {
		return 0, err
	}
	return int64(sysinfo.Uptime), nil
}

func GetOpenFileDescriptorCount() (int16, error) {
	pid := os.Getpid()
	fdpath := filepath.Join("/proc", fmt.Sprintf("%d", pid), "fd")
	files, e := ioutil.ReadDir(fdpath)
	if e == nil {
		return int16(len(files)), e
	} else {
		return int16(0), e
	}
}
