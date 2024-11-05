package session

import (
	"fmt"
	"log"
	"math"
	gonet "net"
	"time"

	"github.com/whatap/go-api/common/io"
	"github.com/whatap/go-api/common/lang/pack"
	"github.com/whatap/go-api/common/util/dateutil"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/net"
	"whatap.io/aws/ecs/config"
	whatapecsio "whatap.io/aws/ecs/io"
	"whatap.io/aws/ecs/secure"
)

var (
	readObservers []func(pack.Pack)
)

func startReceiver() {

	go listen()

}
func listen() {

	for {

		session := net.GetTcpSession()
		if session == nil {
			time.Sleep(1000 * time.Millisecond)
			continue
		}
		var client gonet.Conn
		for {
			client = session.GetClient()
			if client != nil {
				break
			}
			time.Sleep(1000 * time.Millisecond)
		}
		//client.SetWriteDeadline(time.Now().Add(NET_TIMEOUT * 10))
		reader := whatapecsio.NewNetReadHelper(client)
		err := readPack(reader, onPackReceive)
		if err != nil {
			log.Println("session reader error:", err)
			time.Sleep(1000 * time.Millisecond)
			continue
		}

		time.Sleep(1 * time.Millisecond)
	}
}

func AddReadObserver(observer func(pack.Pack)) {
	if observer != nil {
		readObservers = append(readObservers, observer)
	}
}

func onPackReceive(code byte, payload []byte, transferkey int32) {
	if code == net.NET_TIME_SYNC {
		in := io.NewDataInputX(payload)
		_ = in.ReadLong()
		serverTime := in.ReadLong()
		now := dateutil.Now()

		if math.Abs(float64(serverTime-now)) > 1000 {
			newDelta := dateutil.GetDelta() + serverTime - now

			if math.Abs(float64(newDelta-dateutil.GetDelta())) > 1000 {
				dateutil.SetDelta(newDelta)
			}

		}
		return
	}

	conf := config.GetConfig()
	secuSession := secure.GetSecuritySession()
	if conf.CypherLevel > 0 {
		if transferkey != secuSession.TRANSFER_KEY {
			return
		}
		switch net.GetSecureMask(code) {
		case net.NET_SECURE_HIDE:
			if secuSession.Cypher != nil {
				payload = secuSession.Cypher.Hide(payload)
			}
		case net.NET_SECURE_CYPHER:
			if secuSession.Cypher != nil {
				payload = secuSession.Cypher.Decrypt(payload)
			}
		default:
			payload = nil
		}
	}
	if payload != nil {
		p := pack.ToPack(payload)
		if p != nil {
			for _, listener := range readObservers {
				listener(p)
			}
		}

	}
}

func readPack(reader *whatapecsio.NetReadHelper, callback func(byte, []byte, int32)) error {
	_, err := reader.ReadByte()
	if err != nil {

		return err
	}

	code, err := reader.ReadByte()
	if err != nil {

		return err
	}
	pcode, err := reader.ReadLong()
	if err != nil {

		return err
	}

	oid, err := reader.ReadInt()
	if err != nil {

		return err
	}
	transfer_key, err := reader.ReadInt()
	if err != nil {

		return err
	}
	data, err := reader.ReadIntBytesLimit(READ_MAX)
	if err != nil {

		return err
	}

	conf := config.GetConfig()
	if pcode != conf.PCODE || oid != conf.OID {

		return fmt.Errorf("invalid pcode:  ", pcode, "oid:", oid)
	}

	callback(code, data, transfer_key)

	return nil
}
