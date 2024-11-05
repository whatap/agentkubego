package cache

import "github.com/whatap/go-api/common/lang/value"

var (
	microCache = map[string]*value.MapValue{}
)

func SetMicroCache(containerId string, m *value.MapValue) {
	microCache[containerId] = m
}

func GetMicroCache(containerId string) *value.MapValue {
	if m, ok := microCache[containerId]; ok {
		return m
	}
	return nil
}
