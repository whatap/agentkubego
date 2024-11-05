// +build linux

package osinfo

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/whatap/go-api/common/util/hash"
	"whatap.io/aws/ecs/config"
	"whatap.io/aws/ecs/lang/pack"
)

//NicTrafficRaw NicTrafficRaw
type NicTrafficRaw struct {
	DeviceID          string
	ReadCount         int64
	ReadByteCount     int64
	ReadDroppedCount  int64
	ReadErrorCount    int64
	WriteCount        int64
	WriteByteCount    int64
	WriteDroppedCount int64
	WriteErrorCount   int64
	Timestamp         int64
}

const nanotimes int64 = 1000000000

//ParseNicTraffic ParseNicTraffic
func ParseNicTraffic() ([]NetworkPerformance, error) {
	conf := config.GetConfig()
	filepath := path.Join(conf.NodeVolPrefix, "/proc/1/net/dev")

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var nicTrafficMap2 map[string]NicTrafficRaw
	loadedCache := LoadCache(filepath)
	if loadedCache != nil {
		nicTrafficMap2 = loadedCache.(map[string]NicTrafficRaw)
	}
	nicTrafficMap := make(map[string]NicTrafficRaw)
	timestamp := time.Now().UnixNano()
	scanner := bufio.NewScanner(file)
	i := 0
	j := 0
	nicPerformances := make([]NetworkPerformance, 100)
	for scanner.Scan() {
		line := scanner.Text()
		j++
		if j < 3 {
			continue
		}
		words := strings.Fields(strings.Replace(line, ":", " ", -1))
		deviceID := words[0]

		// if !isBond(deviceID) && !isPhysical(deviceID) && !isBridge(deviceID) {
		// 	continue
		// }
		// ip, macaddr, ipenabled := getIP(deviceID)
		// if !ipenabled {
		// 	continue
		// }

		readByteCount, _ := strconv.ParseInt(words[1], 10, 64)
		readCount, _ := strconv.ParseInt(words[2], 10, 64)
		readErrorCount, _ := strconv.ParseInt(words[3], 10, 64)
		readDroppedCount, _ := strconv.ParseInt(words[4], 10, 64)

		writeByteCount, _ := strconv.ParseInt(words[9], 10, 64)
		writeCount, _ := strconv.ParseInt(words[10], 10, 64)
		writeErrorCount, _ := strconv.ParseInt(words[11], 10, 64)
		writeDroppedCount, _ := strconv.ParseInt(words[12], 10, 64)

		nicTrafficIORaw := NicTrafficRaw{DeviceID: deviceID, ReadCount: readCount, ReadByteCount: readByteCount, ReadDroppedCount: readDroppedCount, ReadErrorCount: readErrorCount,
			WriteCount: writeCount, WriteByteCount: writeByteCount, WriteDroppedCount: writeDroppedCount, WriteErrorCount: writeErrorCount, Timestamp: timestamp}

		nicTrafficMap[deviceID] = nicTrafficIORaw
		if v, ok := nicTrafficMap2[deviceID]; ok {
			nicPerformances[i].Desc = deviceID
			nicPerformances[i].TrafficIn = float64(readByteCount-v.ReadByteCount) / float64(timestamp-v.Timestamp) * float64(nanotimes) * 8
			nicPerformances[i].TrafficOut = float64(writeByteCount-v.WriteByteCount) / float64(timestamp-v.Timestamp) * float64(nanotimes) * 8
			nicPerformances[i].PacketIn = float64(readCount-v.ReadCount) / float64(timestamp-v.Timestamp) * float64(nanotimes)
			nicPerformances[i].PacketOut = float64(writeCount-v.WriteCount) / float64(timestamp-v.Timestamp) * float64(nanotimes)
			nicPerformances[i].DroppedIn = float64(readDroppedCount-v.ReadDroppedCount) / float64(timestamp-v.Timestamp) * float64(nanotimes)
			nicPerformances[i].DroppedOut = float64(writeDroppedCount-v.WriteDroppedCount) / float64(timestamp-v.Timestamp) * float64(nanotimes)
			nicPerformances[i].ErrorIn = float64(readErrorCount-v.ReadErrorCount) / float64(timestamp-v.Timestamp) * float64(nanotimes)
			nicPerformances[i].ErrorOut = float64(writeErrorCount-v.WriteErrorCount) / float64(timestamp-v.Timestamp) * float64(nanotimes)

			nicPerformances[i].TrafficInAcct = readByteCount
			nicPerformances[i].TrafficOutAcct = writeByteCount

			i++
		}
	}

	SaveCache(filepath, nicTrafficMap)

	return nicPerformances[:i], nil
}

//GetNicUsage GetNicUsage
func GetNicUtil(textCallback func(int32, int32, string)) []pack.NetPerf {
	p, _ := ParseNicTraffic()
	n := make([]pack.NetPerf, len(p))
	for i := 0; i < len(p); i++ {
		//fmt.Println(p[i])
		n[i].Desc = hash.HashStr(p[i].Desc)
		textCallback(pack.TEXT_SYS_NET_DESC, n[i].Desc, p[i].Desc)
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.BigEndian, p[i].IP)
		n[i].IP = buf.Bytes()

		n[i].HwAddr = p[i].HwAddr

		n[i].TrafficIn = p[i].TrafficIn
		n[i].TrafficOut = p[i].TrafficOut
		n[i].PacketIn = p[i].PacketIn
		n[i].PacketOut = p[i].PacketOut
		n[i].ErrorOut = p[i].ErrorOut
		n[i].ErrorIn = p[i].ErrorIn
		n[i].DroppedOut = p[i].DroppedOut
		n[i].DroppedIn = p[i].DroppedIn

		n[i].TrafficInAcct = p[i].TrafficInAcct
		n[i].TrafficOutAcct = p[i].TrafficOutAcct

	}
	return n
}
