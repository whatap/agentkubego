package sys

import (
	"strings"

	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/ansi"

	"github.com/shirou/gopsutil/net"
	//	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/logutil"
)

//type IOCountersStat struct {
//	Name        string `json:"name"`        // interface name
//	BytesSent   uint64 `json:"bytesSent"`   // number of bytes sent
//	BytesRecv   uint64 `json:"bytesRecv"`   // number of bytes received
//	PacketsSent uint64 `json:"packetsSent"` // number of packets sent
//	PacketsRecv uint64 `json:"packetsRecv"` // number of packets received
//	Errin       uint64 `json:"errin"`       // total number of errors while receiving
//	Errout      uint64 `json:"errout"`      // total number of errors while sending
//	Dropin      uint64 `json:"dropin"`      // total number of incoming packets which were dropped
//	Dropout     uint64 `json:"dropout"`     // total number of outgoing packets which were dropped (always 0 on OSX and BSD)
//	Fifoin      uint64 `json:"fifoin"`      // total number of FIFO buffers errors while receiving
//	Fifoout     uint64 `json:"fifoout"`     // total number of FIFO buffers errors while sending
//
//}

func toStr(ifn []net.InterfaceAddr) string {
	o := ""
	for _, i := range ifn {
		if strings.Contains(i.Addr, "::") {
			o = o + " [" + i.Addr + "]"
		} else {
			o = o + " [" + ansi.Green(i.Addr) + "]"
		}
	}
	return o
}
func GetSysNicList() ([]string, []string) {
	var name []string
	var addr []string

	st, err := net.Interfaces()
	if err == nil {
		for _, i := range st {
			if len(i.Addrs) > 0 {
				name = append(name, i.Name)
				addr = append(addr, toStr(i.Addrs))
			}
		}
	}
	return name, addr
}

var last *net.IOCountersStat = nil

func GetSysNet(nicName string) (*net.IOCountersStat, error) {

	n, err := net.IOCounters(true)
	if err != nil {
		return nil, err
	}

	delta := func(next *net.IOCountersStat) *net.IOCountersStat {
		if last == nil || last.Name != next.Name {
			last = next
			return nil
		} else {
			d := new(net.IOCountersStat)
			d.Name = next.Name
			d.BytesSent = next.BytesSent - last.BytesSent
			d.BytesRecv = next.BytesRecv - last.BytesRecv
			d.PacketsSent = next.PacketsSent - last.PacketsSent
			d.PacketsRecv = next.PacketsRecv - last.PacketsRecv
			d.Errin = next.Errin - last.Errin
			d.Errout = next.Errout - last.Errout
			d.Dropin = next.Dropin - last.Dropin
			d.Dropout = next.Dropout - last.Dropout
			last = next
			return d
		}
	}
	if nicName == "" {
		r := net.IOCountersStat{
			Name: "all",
		}
		for _, nic := range n {
			r.BytesRecv += nic.BytesRecv
			r.PacketsRecv += nic.PacketsRecv
			r.Errin += nic.Errin
			r.Dropin += nic.Dropin
			r.BytesSent += nic.BytesSent
			r.PacketsSent += nic.PacketsSent
			r.Errout += nic.Errout
			r.Dropout += nic.Dropout
		}
		return delta(&r), nil
	}
	for _, nic := range n {
		if nic.Name == nicName {
			return delta(&nic), nil
		}

	}

	return nil, err

}
