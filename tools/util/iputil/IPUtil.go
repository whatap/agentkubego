package iputil

import (
	"bytes"
	"encoding/hex"
	"github.com/whatap/kube/tools/io"
	"net"
	"strconv"
	"strings"
)

func ToStringFrInt(ip int32) string {
	return ToString(io.ToBytesInt(ip))
}

func ToString(ip []byte) string {
	if ip == nil || len(ip) == 0 {
		return "0.0.0.0"
	}
	var buffer bytes.Buffer
	buffer.WriteString(strconv.Itoa(int(uint(ip[0]))))
	buffer.WriteString(".")
	buffer.WriteString(strconv.Itoa(int(uint(ip[1]))))
	buffer.WriteString(".")
	buffer.WriteString(strconv.Itoa(int(uint(ip[2]))))
	buffer.WriteString(".")
	buffer.WriteString(strconv.Itoa(int(uint(ip[3]))))
	return buffer.String()
}

func ToBytes(ip string) []byte {
	if ip == "" {
		return []byte{0, 0, 0, 0}
	}
	result := []byte{0, 0, 0, 0}
	s := strings.Split(ip, ".")
	if len(s) != 4 {
		return []byte{0, 0, 0, 0}
	}
	for i := 0; i < 4; i++ {
		if val, err := strconv.Atoi(s[i]); err == nil {
			result[i] = (byte)(val & 0xff)
		}
	}
	return result
}

func ToBytesFrInt(ip int32) []byte {
	return io.ToBytesInt(ip)
}
func ToInt(ip []byte) int32 {
	return io.ToInt(ip, 0)
}

func IsOK(ip []byte) bool {
	return ip != nil && len(ip) == 4
}

func IsNotLocal(ip []byte) bool {
	return IsOK(ip) && uint(ip[0]) != 127
}

func ParseHexString(ipport string) ([]byte, error) {
	words := strings.Split(ipport, ":")
	parsedbytes, err := hex.DecodeString(words[0])
	if err != nil {
		return nil, err
	}
	parsedLength := len(parsedbytes)
	ipbytes := make([]byte, 6)
	ipbytes[3] = parsedbytes[parsedLength-4]
	ipbytes[2] = parsedbytes[parsedLength-3]
	ipbytes[1] = parsedbytes[parsedLength-2]
	ipbytes[0] = parsedbytes[parsedLength-1]

	portbytes, err := hex.DecodeString(words[1])
	ipbytes[4] = portbytes[0]
	ipbytes[5] = portbytes[1]

	return ipbytes, nil
}
func IsIPv6(address string) bool {
	ip := net.ParseIP(address)
	return ip != nil && ip.To4() == nil
}
