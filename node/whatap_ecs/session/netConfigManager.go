package session

import (
	"gitlab.whatap.io/hsnam/focus-agent/whatap/net"
	"whatap.io/aws/ecs/config"
)

func InitWhatapNet() {

	config.AddObserver(onConfigChange)
}

func onConfigChange(conf *config.Config) {
	net.WhatapHost = conf.WhatapHost
	net.WhatapPort = conf.WhatapPort
	net.PCODE = conf.PCODE
	net.LicenseHash64 = conf.LicenseHash64
}
