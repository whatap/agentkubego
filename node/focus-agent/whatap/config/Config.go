package config

type Config struct {
	VERSION  string
	PCODE    int64
	OID      int32
	ONAME    string
	Category string

	OKIND string
	ONODE string

	License       string
	LicenseHash64 int64

	WhatapHost []string
	WhatapPort int32
	WhatapDest int

	Enabled bool

	TcpSoTimeout         int32 // TODO
	TcpConnectionTimeout int32 // TODO

	NetSendMaxBytes   int32
	NetSendBufferSize int32
	NetSendQueue1Size int32
	NetSendQueue2Size int32

	QueueLogEnabled           bool
	QueueYieldEnabled         bool
	QueueTcpEnabled           bool
	QueueTcpSenderThreadCount int32

	TagCounterEnabled bool
	Interval          int

	OneTime bool
	Silent  bool

	// SM
	WmiEnabled           bool
	NativeAPIEnabled     bool
	ProcessFallback      bool
	ServerProcessFDCheck bool

	PidFile string

	CypherLevel  int32
	EncryptLevel int32
	IP           string
}

var conf *Config = nil

func GetConfig() *Config {
	if conf != nil {
		return conf
	}
	conf = new(Config)
	apply(conf)
	return conf
}

func apply(conf *Config) {

	conf.VERSION = "1.1.1"
	conf.Enabled = true

	conf.QueueLogEnabled = false
	conf.QueueYieldEnabled = false

	conf.QueueTcpEnabled = true
	conf.QueueTcpSenderThreadCount = 2

	conf.TcpSoTimeout = 120000
	conf.TcpConnectionTimeout = 5000

	conf.NetSendMaxBytes = 5 * 1024 * 1024
	conf.NetSendBufferSize = 1024
	conf.NetSendQueue1Size = 256
	conf.NetSendQueue2Size = 512

	conf.TagCounterEnabled = true
	conf.Interval = 5

	conf.OneTime = false

	// SM
	conf.WmiEnabled = true
	conf.NativeAPIEnabled = false
	conf.ProcessFallback = false

	conf.ServerProcessFDCheck = true
	conf.CypherLevel = 128
	conf.EncryptLevel = 2
}
