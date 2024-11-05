package main

import (
	"bufio"
	"fmt"
	"github.com/whatap/kube/node/src/whatap/io"
	"github.com/whatap/kube/node/src/whatap/lang/value"
	"log"
	"net"
	"time"
)

const (
	infiniteLoopTimeout = time.Second * 5
)

var (
	gSession             *SimpleTcpSession
	Host                 string
	Port                 int
	TcpConnectionTimeout time.Duration = time.Second * 3
)

func collectGpuForever() {

	for {
		collectNvidia(func(tags *value.MapValue, fields *value.MapValue) {
			send(tags, fields)
		})
		time.Sleep(infiniteLoopTimeout)
	}
}

type SimpleTcpSession struct {
	client net.Conn
	wr     *bufio.Writer
}

func (this *SimpleTcpSession) close() {
	if this.client != nil {
		this.client.Close()
	}
	this.client = nil
	return
}
func (this *SimpleTcpSession) isOpen() (isopen bool) {
	isopen = this.client != nil
	return
}

func newSession() (session *SimpleTcpSession) {
	session = new(SimpleTcpSession)

	for {
		addr := fmt.Sprintf("%s:%d", Host, Port)
		client, err := net.DialTimeout("tcp", addr, TcpConnectionTimeout)
		if err != nil {
			log.Println("WA173", "Connection error. ", err)
			time.Sleep(infiniteLoopTimeout)
			continue
		}
		session.client = client
		session.wr = bufio.NewWriter(session.client)
	}

	return
}

func getSession() (session *SimpleTcpSession, sessionErr error) {
	if gSession == nil || !gSession.isOpen() {
		gSession = newSession()
	}
	return
}

func send(tags *value.MapValue, fields *value.MapValue) (sendErr error) {
	session, err := getSession()
	if err != nil {
		session.close()
		sendErr = err
		return
	}

	m := value.NewMapValue()
	m.PutString("category", "kube_nvidiasmi")
	m.PutMapValue("tags", tags)
	m.PutMapValue("fields", fields)

	dout := io.NewDataOutputX()
	dout.WriteShort(0x1111)
	dout.WriteIntBytes(m.ToByte())

	buf := dout.ToByteArray()

	session.wr.Write(buf)
	return
}
