package runtimeutil

import "os"

func CheckDockerEnabled() bool {
	fi, err := os.Stat("/var/run/docker.sock")
	if err != nil && os.IsNotExist(err) {
		return false
	}

	if fi.Mode().IsDir() {
		return false
	}

	return true
}

func CheckCrioEnabled() bool {
	fi, err := os.Stat("/var/run/crio/crio.sock")
	if err != nil && os.IsNotExist(err) {
		return false
	}

	if fi.Mode().IsDir() {
		return false
	}

	return true
}

func CheckContainerdEnabled() bool {
	fi, err := os.Stat("/run/containerd/containerd.sock")
	if err != nil && os.IsNotExist(err) {
		return false
	}

	if fi.Mode().IsDir() {
		return false
	}

	return true
}
