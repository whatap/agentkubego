package counter

import (
	"whatap.io/k8s/sidecar/config"
)

var (
	namespacePcodeLookup map[string][]int64
	nodeNamespace        string
	nsObservers          []func(string)
	containerObservers   []func(string, string, string, string)
)

func OnNamespaceProjectChange(nsPcodeLookup map[string][]int64) {
	namespacePcodeLookup = nsPcodeLookup
}

func findNamespace(ns string, h2 func(int64, int64)) {
	if namespacePcodeLookup != nil {
		if pcodeNlicenseHash, ok := namespacePcodeLookup[ns]; ok {
			if len(pcodeNlicenseHash) == 2 {
				h2(pcodeNlicenseHash[0], pcodeNlicenseHash[1])
				return
			}

		}
	}
	// log.Println("findNamespace step -3")
	conf := config.GetConfig()
	h2(conf.PCODE, conf.LicenseHash64)
	// log.Println("findNamespace step -4")
	return
}

func AddNSObserver(observer func(string)) {
	nsObservers = append(nsObservers, observer)
}
func setNodeNamespace(ns string) {
	nodeNamespace = ns
	for _, observer := range nsObservers {
		observer(ns)
	}
}

func AddObserver(observer func(string, string, string, string)) {
	containerObservers = append(containerObservers, observer)
}

func onContainerDetected(cid string, ns string, podName string, cName string) {
	for _, observer := range containerObservers {
		observer(cid, ns, podName, cName)
	}
}
