package session

import (
	"log"
	gonet "net"
	"time"

	"github.com/whatap/go-api/common/io"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/net"
	"whatap.io/aws/ecs/config"
	"whatap.io/aws/ecs/secure"
)

const (
	READ_MAX    = 8 * 1024 * 1024
	NET_TIMEOUT = 30 * time.Second
	LOG_LIMIT   = 6
	CAFE        = 0xcafe
)

var (
	AGENT_VERSION = "0.0.1"
	BUILDNO       = "001"
)

func keyReset() []byte {

	conf := config.GetConfig()
	dout := io.NewDataOutputX()
	secu := secure.GetSecurityMaster()
	secu.WaitForInit()

	msg := dout.WriteText("hello").WriteText(conf.ONAME).WriteInt(getMyAddr()).ToByteArray()
	if conf.CypherLevel > 0 {
		msg = secu.Encrypt(msg)
	}
	dout = io.NewDataOutputX()
	dout.WriteByte(net.NETSRC_AGENT_JAVA_EMBED)

	var trkey int32 = 0
	if conf.CypherLevel == 128 {
		dout.WriteByte(byte(net.NET_KEY_RESET))
	} else {
		dout.WriteByte(byte(net.NET_KEY_EXTENSION))

		if conf.CypherLevel == 0 {
			trkey = 0
		} else {
			b0 := byte(1)
			b1 := byte(conf.CypherLevel / 8)
			trkey = io.ToInt([]byte{byte(b0), byte(b1), byte(0), byte(0)}, 0)
		}
	}
	dout.WriteLong(conf.PCODE)
	dout.WriteInt(conf.OID)
	dout.WriteInt(trkey)
	dout.WriteIntBytes(msg)

	return dout.ToByteArray()
}

func readKeyResetEx(conn gonet.Conn) []byte {
	//conn.SetReadDeadline(time.Now().Add(NET_TIMEOUT))
	buflen := 22
	buf := make([]byte, buflen)
	nbytethistime, err := conn.Read(buf)
	if buflen > nbytethistime || err != nil {

		return []byte{}
	}
	in := io.NewDataInputX(buf)

	_ = in.ReadByte()
	_ = in.ReadByte()
	pcode := in.ReadLong()
	_ = in.ReadInt()
	_ = in.ReadInt()
	datasize := in.ReadInt()
	if datasize > 1024 {
		return []byte{}
	}
	data := make([]byte, datasize)
	//conn.SetReadDeadline(time.Now().Add(NET_TIMEOUT))
	nbytethistime, derr := conn.Read(data)
	if datasize > int32(nbytethistime) || derr != nil {
		return []byte{}
	}

	secu := secure.GetSecurityMaster()

	if pcode != secu.PCODE {
		return []byte{}
	} else {
		return data
	}
}

func InitSecureSession() bool {
	conf := config.GetConfig()
	secure.CypherLevel = int(conf.CypherLevel)

	secure.GetSecurityMaster().WaitForInit()

	secure.GetSecuritySession()

	net.AddOpenHandler(func(sess *net.TcpSession) {
		client := sess.GetClient()
		keyreset := keyReset()

		nbyteSent, err := client.Write(keyreset)
		if len(keyreset) > nbyteSent || err != nil {
			log.Println("ERROR", "key reset request ", len(keyreset), "/", nbyteSent, " err:", err)
			client.Close()

			return
		}
		data := readKeyResetEx(client)
		if data == nil || len(data) < 1 {
			log.Println("ERROR", "key reset response len(data):", len(data), " err:", err)
			client.Close()

			return
		}
		secure.UpdateNetCypherKey(data)
	})
	net.GetTcpSession()

	startReceiver()

	return true
}
