package secure

import (
	"sync"
	"time"

	"github.com/whatap/go-api/common/io"
	"github.com/whatap/go-api/common/util/hash"
	"whatap.io/aws/ecs/config"
)

type SecurityMaster struct {
	PCODE       int64
	OID         int32
	OkindName   string
	Okind       int32
	ONAME       string
	IP          int32
	SECURE_KEY  []byte
	cypher      *Cypher
	lastOidSent int64
	PUBLIC_IP   int32
}

type SecuritySession struct {
	TRANSFER_KEY int32
	SECURE_KEY   []byte
	HIDE_KEY     int32
	Cypher       *Cypher
}

var master *SecurityMaster = nil
var session *SecuritySession = nil
var mutex = sync.Mutex{}

func GetSecurityMaster() *SecurityMaster {
	if master != nil {
		return master
	}
	mutex.Lock()
	defer mutex.Unlock()

	if master != nil {
		return master
	}
	master = new(SecurityMaster)
	go master.run()
	return master
}
func GetSecuritySession() *SecuritySession {
	if session != nil {
		return session
	}
	mutex.Lock()
	defer mutex.Unlock()
	session = &SecuritySession{}
	return session
}
func UpdateNetCypherKey(data []byte) {
	conf := config.GetConfig()
	if conf.CypherLevel > 0 {
		CypherLevel = int(conf.CypherLevel)
		data = GetSecurityMaster().cypher.Decrypt(data)
	}
	in := io.NewDataInputX(data)

	session.TRANSFER_KEY = in.ReadInt()
	session.SECURE_KEY = in.ReadBlob()
	session.HIDE_KEY = in.ReadInt()
	session.Cypher = NewCypher(session.SECURE_KEY, session.HIDE_KEY)
	master.PUBLIC_IP = in.ReadInt()
}

func (this *SecurityMaster) init() {
	conf := config.GetConfig()
	if len(conf.Okind) > 0 {
		this.OkindName = conf.Okind
		this.Okind = hash.HashStr(this.OkindName)
	}
}

func (this *SecurityMaster) run() {
	conf := config.GetConfig()
	this.init()

	oldLic := ""
	for {
		if len(conf.License) > 0 && conf.License != oldLic {
			oldLic = conf.License
			this.resetLicense(conf.License)
		}
		time.Sleep(3000 * time.Millisecond)
	}
}

var respDict map[string]interface{}

func (this *SecurityMaster) resetLicense(lic string) {
	pcode, security_key := Parse(lic)
	this.PCODE = pcode
	this.SECURE_KEY = security_key
	this.cypher = NewCypher(this.SECURE_KEY, 0)

	conf := config.GetConfig()
	conf.PCODE = this.PCODE
	config.Update()
}

func (this *SecurityMaster) WaitForInit() {
	for this.cypher == nil {
		time.Sleep(1000 * time.Millisecond)
	}
}

func (this *SecurityMaster) WaitForInitFor(timeoutSec float64) {
	started := time.Now()
	for this.cypher == nil && time.Now().Sub(started).Seconds() < timeoutSec {
		time.Sleep(1000 * time.Millisecond)
	}
}

func (this *SecurityMaster) Encrypt(data []byte) []byte {
	if this.cypher == nil {
		return data
	}

	return this.cypher.Encrypt(data)
}
