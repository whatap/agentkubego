package net

import (
	"fmt"
	"net"
	"time"

	"github.com/whatap/go-api/common/io"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/logutil"
)

const (
	READ_MAX = 8 * 1024 * 1024
)

var (
	WhatapDest           int
	WhatapHost           []string
	WhatapPort           int32
	PCODE                int64
	LicenseHash64        int64
	TcpSoTimeout         int = 120000
	TcpConnectionTimeout int = 5000
)

type TcpSession struct {
	tcp  net.Conn
	dest int
}

var (
	session           *TcpSession
	openEventHandlers []func(session *TcpSession)
)

func GetTcpSession() *TcpSession {
	if session != nil {
		return session
	}
	session = new(TcpSession)
	session.open()
	go func() {
		time.Sleep(1000 * time.Millisecond)
		for {
			for session.isOpen() {
				time.Sleep(5000 * time.Millisecond)
			}
			for session.open() == false {
				time.Sleep(5000 * time.Millisecond)
			}
		}
	}()
	return session
}

var host = ""
var port = int32(6600)

func (this *TcpSession) open() (ret bool) {
	if this.isOpen() {
		return true
	}

	WhatapDest += 1
	if WhatapDest >= len(WhatapHost) {
		WhatapDest = 0
	}

	host = WhatapHost[WhatapDest]
	port = WhatapPort

	tcpClient, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), time.Duration(TcpConnectionTimeout)*time.Millisecond)
	if err != nil {
		logutil.Errorln("TCP", "Connect error. "+host+":", port)
		this.Close()
		return false
	}
	logutil.Printf("TCP", "Connected %s:%d", host, port)
	this.tcp = tcpClient
	handleOpenEvent(this)
	return true
}

func AddOpenHandler(h1 func(session *TcpSession)) {
	openEventHandlers = append(openEventHandlers, h1)
}

func handleOpenEvent(session *TcpSession) {
	for _, onOpenEvent := range openEventHandlers {
		onOpenEvent(session)
	}
}

func (this *TcpSession) isOpen() bool {
	return this.tcp != nil
}

func (this *TcpSession) GetLocalAddr() net.Addr {
	return this.tcp.LocalAddr()
}

func (this *TcpSession) GetClient() net.Conn {
	return this.tcp
}

func (this *TcpSession) Send(b []byte) (ret bool) {

	if this.tcp == nil {
		return false
	}

	err := session.tcp.SetWriteDeadline(time.Now().Add(time.Duration(TcpSoTimeout) * time.Millisecond))
	if err != nil {
		logutil.Errorln("TCP", " SetWriteDeadline failed:", err)
	}

	out := io.NewDataOutputX()
	out.WriteByte(NETSRC_AGENT_ONEWAY)
	out.WriteByte(0)
	out.WriteLong(PCODE)
	out.WriteLong(LicenseHash64)
	out.WriteIntBytes(b)

	sendbuf := out.ToByteArray()

	total := len(sendbuf)
	left := total
	for i := 0; 0 < left && i < 3; i++ {
		n, err := this.tcp.Write(sendbuf[total-left : total])
		if err != nil {
			logutil.Errorln("TCP", err)
			this.Close()
			return false
		}
		left -= n
	}
	if left > 0 {
		logutil.Errorln("TCP", "All data was not sent. (tot=", total, " left=", left, " bytes")
		this.Close()
		return false
	}
	return true
}

func (this *TcpSession) Close() {
	//logutil.Println("WA181 Close", string(debug.Stack()))
	if this.tcp != nil {
		defer func() {
			recover()
			this.tcp = nil
		}()
		this.tcp.Close()
		logutil.Printf("TCP", "Closed %s:%d", host, port)
	}
	this.tcp = nil
}

func Shutdown() {
	if session != nil {
		session.Close()
	}
}
