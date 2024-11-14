package pods

import (
	"sync"
)

var (
	podsMap     = make(map[string]SimplePodInfo)
	podsMapLock sync.RWMutex
)

func GetPodsMap() map[string]SimplePodInfo {
	podsMapLock.RLock()
	defer podsMapLock.RUnlock()
	return podsMap
}

func UpdatePodInfo(podInfo SimplePodInfo) {
	podsMapLock.Lock()
	defer podsMapLock.Unlock()
	podsMap[podInfo.Uid] = podInfo
}
