package net

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/whatap/go-api/common/io"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/dateutil"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/logutil"
)

const (
	UDP_READ_MAX                    = 8 * 1024 * 1024
	UDP_PACKET_BUFFER               = 64 * 1024
	UDP_PACKET_BUFFER_CHUNKED_LIMIT = 48 * 1024
	UDP_PACKET_CHANNEL_MAX          = 255
	UDP_PACKET_FLUSH_TIMEOUT        = 10 * 1000

	UDP_PACKET_HEADER_SIZE = 9
	// typ pos 0
	UDP_PACKET_HEADER_TYPE_POS = 0
	// ver pos 1
	UDP_PACKET_HEADER_VER_POS = 1
	// len pos 5
	UDP_PACKET_HEADER_LEN_POS = 5

	UDP_PACKET_SQL_MAX_SIZE = 32768
)

type UdpSession struct {
	host string
	port int

	udp net.Conn
	wr  *bufio.Writer

	sendCh       chan *UdpData
	lastSendTime int64

	lock sync.Mutex
}

type UdpData struct {
	Type  byte
	Ver   int32
	Data  []byte
	Flush bool
}

//
var udpSession *UdpSession

func GetUdpSession() *UdpSession {
	if udpSession != nil {
		return udpSession
	}
	udpSession = new(UdpSession)
	udpSession.host = "127.0.0.1"
	udpSession.port = 6600
	udpSession.sendCh = make(chan *UdpData, UDP_PACKET_CHANNEL_MAX)
	udpSession.open()
	go func() {
		time.Sleep(1000 * time.Millisecond)
		for {
			for udpSession.isOpen() {
				time.Sleep(5000 * time.Millisecond)
			}
			for udpSession.open() == false {
				time.Sleep(5000 * time.Millisecond)
			}
		}
	}()
	go udpSession.send()

	return udpSession
}

func (this *UdpSession) open() (ret bool) {
	if this.isOpen() {
		return true
	}

	udpClient, err := net.DialTimeout("udp", fmt.Sprintf("%s:%d", this.host, this.port), time.Duration(TcpConnectionTimeout)*time.Millisecond)
	if err != nil {
		logutil.Errorln("UDP", "Connect error. "+this.host+":", this.port)
		this.Close()
		return false
	}
	this.udp = udpClient
	this.wr = bufio.NewWriterSize(this.udp, UDP_PACKET_BUFFER)
	logutil.Printf("UDP", "Connected %s:%d", this.host, this.port)

	return true
}
func (this *UdpSession) isOpen() bool {
	return this.udp != nil && this.wr != nil
}

func (this *UdpSession) GetLocalAddr() net.Addr {
	return this.udp.LocalAddr()
}

func (this *UdpSession) Send(t uint8, ver int32, b []byte, flush bool) bool {
	this.lock.Lock()
	defer this.lock.Unlock()

	buff := make([]byte, len(b))
	copy(buff, b)
	this.sendCh <- &UdpData{t, ver, buff, flush}
	return true
}

func (this *UdpSession) send() {
	for {
		select {
		case sendData := <-this.sendCh:
			if !this.isOpen() {
				continue
			}
			out := io.NewDataOutputX()
			out.WriteByte(sendData.Type)
			out.WriteInt(sendData.Ver)
			out.WriteIntBytes(sendData.Data)
			sendBytes := out.ToByteArray()

			if this.wr.Buffered() > 0 && this.wr.Buffered()+len(sendBytes) > UDP_PACKET_BUFFER_CHUNKED_LIMIT {
				this.lastSendTime = dateutil.Now()
				if err := this.wr.Flush(); err != nil {
					logutil.Errorln("UDP", "Error Flush ", err)
					this.Close()
					continue
				}
			}
			if n, err := this.wr.Write(sendBytes); err != nil {
				logutil.Errorln("UDP", "Error Write send=", len(sendBytes), ", n=", n, ",err=", err)
				this.Close()
				continue
			}
			// flush == true
			if this.wr.Buffered() > 0 && sendData.Flush {
				this.lastSendTime = dateutil.Now()
				if err := this.wr.Flush(); err != nil {
					logutil.Errorln("UDP", "Error Flush ", err)
					this.Close()
					continue
				}
			}
		default:
			if !this.isOpen() {
				continue
			}
			time.Sleep(1 * time.Second)
			// 시간 비교하여 발송.

			if this.wr.Buffered() > 0 && dateutil.SystemNow()-this.lastSendTime > UDP_PACKET_FLUSH_TIMEOUT {
				this.lastSendTime = dateutil.Now()
				if err := this.wr.Flush(); err != nil {
					logutil.Errorln("UDP", "Error Flush ", err)
					this.Close()
					continue
				}
			}
		}
	}
}

func (this *UdpSession) Close() {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.udp != nil {
		defer func() {
			recover()
			this.udp = nil
		}()
		this.udp.Close()
		logutil.Printf("UDP", "Closed %s:%d", host, port)
	}
	this.udp = nil
	this.wr = nil
}

func UdpShutdown() {
	if udpSession != nil {
		udpSession.Close()
	}
}
