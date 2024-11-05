package session

import (
	"fmt"
	"log"
	"math"
	gonet "net"
	"time"

	"github.com/whatap/golib/io"
	"github.com/whatap/golib/lang/pack"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/agent/secure"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/net"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/dateutil"
	"whatap.io/k8s/sidecar/config"
	sidecario "whatap.io/k8s/sidecar/io"
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
		client.SetWriteDeadline(time.Now().Add(NET_TIMEOUT * 10))
		reader := sidecario.NewNetReadHelper(client)
		err := readPack(reader, onPackReceive)
		if err != nil {
			// if netErr, ok := err.(gonet.Error); ok && netErr.Timeout() {
			// 	continue
			// }

			log.Println("session reader ", err)
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
		//fmt.Println("Rceiver.run serverTime - now:", serverTime-now)
		if math.Abs(float64(serverTime-now)) > 1000 {
			newDelta := dateutil.GetDelta() + serverTime - now
			// panicutil.Debug("Receiver.run delta diff:", math.Abs(float64(newDelta-dateutil.GetDelta())))
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

func readPack(reader *sidecario.NetReadHelper, callback func(byte, []byte, int32)) error {
	_, err := reader.ReadByte()
	if err != nil {

		// log.Println("readpack step -1", err)
		return err
	}

	code, err := reader.ReadByte()
	if err != nil {
		// log.Println("readpack step -2", err)
		return err
	}
	pcode, err := reader.ReadLong()
	if err != nil {
		// log.Println("readpack step -3", err)

		return err
	}

	oid, err := reader.ReadInt()
	if err != nil {
		// log.Println("readpack step -4", err)
		return err
	}
	transfer_key, err := reader.ReadInt()
	if err != nil {
		// log.Println("readpack step -5", err)
		return err
	}
	data, err := reader.ReadIntBytesLimit(READ_MAX)
	if err != nil {
		// log.Println("readpack step -6", err)
		return err
	}

	conf := config.GetConfig()
	if pcode != conf.PCODE || oid != conf.OID {

		return fmt.Errorf("invalid pcode:  ", pcode, "oid:", oid)
	}

	callback(code, data, transfer_key)

	return nil
}
