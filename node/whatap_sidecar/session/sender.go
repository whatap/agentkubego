package session

import (
	"log"

	"github.com/whatap/golib/io"
	"github.com/whatap/golib/lang/pack"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/agent/secure"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/net"
)

func Send(p pack.Pack) bool {
	b := pack.ToBytesPack(p)
	// secu := secure.GetSecurityMaster()
	// b = secu.Cypher.Hide(b)
	// return sendSecure(net.NET_SECURE_HIDE, b)
	return sendSecure(0, b)
}

func SendHide(p pack.Pack) bool {
	b := pack.ToBytesPack(p)
	// secu := secure.GetSecurityMaster()
	// b = secu.Cypher.Hide(b)
	// return sendSecure(net.NET_SECURE_HIDE, b)
	return sendSecure(0, b)
}

func SendEncrypted(p pack.Pack) bool {
	b := pack.ToBytesPack(p)
	return sendSecure(0, b)
	// secu := secure.GetSecurityMaster()
	// b = secu.Cypher.Encrypt(b)
	// return sendSecure(net.NET_SECURE_CYPHER, b)
}

func sendSecure(code byte, b []byte) (ret bool) {
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

	session := net.GetTcpSession()
	client := session.GetClient()
	buflen := len(sendbuf)
	nbyteleft := buflen
	for 0 < nbyteleft {
		// client.SetWriteDeadline(time.Now().Add(NET_TIMEOUT))
		nbytethistime, err := client.Write(sendbuf[buflen-nbyteleft : buflen])
		if err != nil {
			log.Println("sendSecure err:", err)
			session.Close()
			return false
		}
		nbyteleft -= nbytethistime
	}

	// client.SetWriteDeadline(time.Time{})

	return true

}

func SendOneway(pcode int64, licenseHash64 int64, p pack.Pack) bool {
	b := pack.ToBytesPack(p)

	session := net.GetTcpSession()
	client := session.GetClient()
	out := io.NewDataOutputX()
	out.WriteByte(net.NETSRC_AGENT_ONEWAY)
	out.WriteByte(0)
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
