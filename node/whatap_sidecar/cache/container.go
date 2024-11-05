package cache

import "github.com/whatap/golib/lang/value"

var (
	perfCache = map[string]*value.MapValue{}
)

func SetPerfCache(containerId string, perf *value.MapValue) {
	perfCache[containerId] = perf
}

func GetPerfCache(containerId string) *value.MapValue {
	if r, ok := perfCache[containerId]; ok {
		return r
	}
	return nil
}
