package secure

import (
	"sync"
	"time"

	"github.com/whatap/go-api/common/io"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/lang/license"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/crypto"
)

var (
	CypherLevel int32
	License     string
	OName       string
)

type SecurityMaster struct {
	PCODE       int64
	OID         int32
	ONAME       string
	IP          int32
	SECURE_KEY  []byte
	Cypher      *crypto.Cypher
	lastOidSent int64
	PUBLIC_IP   int32
}

type SecuritySession struct {
	TRANSFER_KEY int32
	SECURE_KEY   []byte
	HIDE_KEY     int32
	Cypher       *crypto.Cypher
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
	if CypherLevel > 0 {
		data = GetSecurityMaster().Cypher.Decrypt(data)
	}
	in := io.NewDataInputX(data)

	session.TRANSFER_KEY = in.ReadInt()
	session.SECURE_KEY = in.ReadBlob()
	session.HIDE_KEY = in.ReadInt()
	session.Cypher = crypto.NewCypher(session.SECURE_KEY, session.HIDE_KEY)
	master.PUBLIC_IP = in.ReadInt()
}

func (this *SecurityMaster) run() {
	//log.Println("SecurityMasgter.run", &conf)
	oldLic := ""
	for {
		if len(License) > 0 && License != oldLic {
			oldLic = License
			this.resetLicense(License)
		}
		time.Sleep(3000 * time.Millisecond)
	}
}

var respDict map[string]interface{}

func (this *SecurityMaster) resetLicense(lic string) {
	//log.Println("SecurityMaster.resetLicense", lic)
	pcode, security_key := license.Parse(lic)
	this.PCODE = pcode
	this.SECURE_KEY = security_key
	this.Cypher = crypto.NewCypher(this.SECURE_KEY, 0)
}

func (this *SecurityMaster) WaitForInit() {
	for this.Cypher == nil {
		time.Sleep(1000 * time.Millisecond)
	}
}

func (this *SecurityMaster) WaitForInitFor(timeoutSec float64) {
	started := time.Now()
	for this.Cypher == nil && time.Now().Sub(started).Seconds() < timeoutSec {
		time.Sleep(1000 * time.Millisecond)
	}
}
