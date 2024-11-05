package cache

type ContainerCache struct {
	Tags   map[string]interface{}
	Fields map[string]interface{}
}

var (
	ecsContainerCache map[string]ContainerCache = map[string]ContainerCache{}
)

func SetContainerCache(runtimeId string, tags map[string]interface{}, fields map[string]interface{}) {
	ecsContainerCache[runtimeId] = ContainerCache{Tags: tags, Fields: fields}
}

func GetContainerCache(runtimeId string) (bool, ContainerCache) {
	c, ok := ecsContainerCache[runtimeId]

	return ok, c

}
