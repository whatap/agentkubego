package session

import (
	"log"

	"github.com/whatap/go-api/common/io"
	"github.com/whatap/go-api/common/lang/pack"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/net"
	"whatap.io/aws/ecs/config"
	"whatap.io/aws/ecs/secure"
)

const (
	SECURE_HIDE      = 0x01
	SECURE_CYPHER    = 0x02
	ONEWAY_NO_CYPHER = 0x04
	ACK              = 0xfb
	PREPARE_AGENT    = 0xfc
	KEY_EXTENSION    = 0xfd
	TIME_SYNC        = 0xfe
	KEY_RESET        = 0xff
)

func Send(p pack.Pack) bool {
	// b := pack.ToBytesPack(p)
	// secu := secure.GetSecurityMaster()
	// b = secu.Cypher.Hide(b)
	// return sendSecure(net.NET_SECURE_HIDE, b)
	//log.Println("send packType:", p.GetPackType())
	// return sendSecure(0, b)

	return SendHide(p)
}

func SendHide(p pack.Pack) bool {
	session := secure.GetSecuritySession()
	if session.Cypher == nil {
		return false
	}

	b := pack.ToBytesPack(p)
	b = session.Cypher.Hide(b)

	return sendSecure(net.NET_SECURE_HIDE, b)
}

func SendEncrypted(p pack.Pack) bool {
	b := pack.ToBytesPackECB(p, int(config.GetConfig().CypherLevel/8))
	session := secure.GetSecuritySession()
	b = session.Cypher.Encrypt(b)
	return sendSecure(net.NET_SECURE_CYPHER, b)
}

func sendSecure(code byte, b []byte) (ret bool) {
	session := net.GetTcpSession()
	client := session.GetClient()
	if client == nil {
		return false
	}
	secu := secure.GetSecurityMaster()
	secuSession := secure.GetSecuritySession()
	out := io.NewDataOutputX()
	out.WriteByte(net.NETSRC_AGENT_JAVA_EMBED)
	out.WriteByte(code)
	out.WriteLong(secu.PCODE)
	out.WriteInt(secu.OID)
	out.WriteInt(secuSession.TRANSFER_KEY)
	out.WriteIntBytes(b)
	sendbuf := out.ToByteArray()

	buflen := len(sendbuf)
	nbyteleft := buflen
	for 0 < nbyteleft {
		nbytethistime, err := client.Write(sendbuf[buflen-nbyteleft : buflen])
		if err != nil {
			log.Println("sendSecure err:", err)
			session.Close()
			return false
		}
		nbyteleft -= nbytethistime
	}

	return true

}

func SendOneway(pcode int64, licenseHash64 int64, p pack.Pack) bool {
	b := pack.ToBytesPack(p)

	session := net.GetTcpSession()
	client := session.GetClient()
	out := io.NewDataOutputX()
	out.WriteByte(net.NETSRC_AGENT_ONEWAY)
	out.WriteByte(ONEWAY_NO_CYPHER)
	out.WriteLong(pcode)
	out.WriteLong(licenseHash64)
	out.WriteIntBytes(b)

	sendbuf := out.ToByteArray()

	total := len(sendbuf)
	left := total
	for i := 0; 0 < left && i < 3; i++ {
		n, err := client.Write(sendbuf[total-left : total])
		if err != nil {
			log.Println("TCP", err)
			session.Close()
			return false
		}
		left -= n
	}
	if left > 0 {
		log.Println("TCP", "All data was not sent. (tot=", total, " left=", left, " bytes")
		session.Close()
		return false
	}

	return true
}
