package osinfo

import (
	"sync"

	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/dateutil"
)

const (
	TTL = 1000 * 60
)

var rateCacheLock sync.Mutex
var cacheMap = make(map[string]interface{})

//LoadCache LoadCache
func LoadCache(key string) interface{} {
	rateCacheLock.Lock()
	defer rateCacheLock.Unlock()
	return cacheMap[key]
}

//SaveCache SaveCache
func SaveCache(key string, v interface{}) {
	rateCacheLock.Lock()
	defer rateCacheLock.Unlock()
	cacheMap[key] = v
}

var intCacheMap = map[string][]int64{}

func ContainesKey(key string) (ret bool) {
	ret = false
	_, ret = intCacheMap[key]

	return
}
func PutIntValue(key string, v int32) {
	rateCacheLock.Lock()
	defer rateCacheLock.Unlock()
	intCacheMap[key] = []int64{int64(v), dateutil.Now()}
}

func GetIntValue(key string) (ret int32) {
	now := dateutil.Now()
	if v, ok := intCacheMap[key]; ok {
		if now-v[1] < TTL {
			ret = int32(v[0])
		}

	}
	return
}
