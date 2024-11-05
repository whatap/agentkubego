package confbase

import (
	"fmt"
	"log"
	gonet "net"
	"time"

	"github.com/whatap/golib/io"
	"github.com/whatap/golib/lang/value"
	"github.com/whatap/golib/util/hash"
	whataplicense "gitlab.whatap.io/hsnam/focus-agent/whatap/lang/license"
	sidecario "whatap.io/k8s/sidecar/io"
)

const (
	READ_MAX    = 8 * 1024 * 1024
	NET_TIMEOUT = 30 * time.Second

	LOG_LIMIT = 6
	CAFE      = 0xcafe
)

var (
	Host                 string
	Port                 int32
	TcpConnectionTimeout = 5 * time.Second
	lastFileTime         int64
	observers            []func(map[string][]int64)
)

func AddObserver(observer func(map[string][]int64)) {
	observers = append(observers, observer)
}

func StartPolling() {

	go updateProjectManager()

}

func updateProjectManager() {
	for {
		conn, err := newConnection()
		if err != nil {
			log.Println("connect confbase agent ", err)
			time.Sleep(time.Second * 5)
			continue
		}
		log.Println("connect confbase step -1 conn: ", conn, " err:", err)

		err = sendRequest(conn)
		if err != nil {
			log.Println("connect confbase agent ", err)
			time.Sleep(time.Second * 5)
			continue
		}
		log.Println("connect confbase step -2")
		err = recvResponse(conn)
		if err != nil {
			log.Println("connect confbase agent ", err)
			time.Sleep(time.Second * 5)
			continue
		}
		log.Println("connect confbase step -3")
		time.Sleep(20 * time.Second)
	}
}

func recvResponse(conn gonet.Conn) error {
	reader := sidecario.NewNetReadHelper(conn)
	dataLength, err := readHeader(reader)
	if err != nil {
		return err
	}
	if dataLength > 0 {
		payload, err := reader.ReadBytes(int(dataLength))
		if err != nil {
			return err
		}
		ret := parseResponse(payload)
		// log.Println("recvRespose ret:", ret.ToString())
		if ret.GetBool("ok") && ret.ContainsKey("config") && ret.ContainsKey("filetime") {
			ns2licenseMap := ret.Get("config").(*value.MapValue)
			keys := ns2licenseMap.Keys()
			pcodeLookup := map[string][]int64{}
			for keys.HasMoreElements() {
				k := keys.NextString()
				license := ns2licenseMap.GetString(k)
				pcode, _ := whataplicense.Parse(license)
				licenseHash64 := hash.Hash64Str(license)
				pcodeLookup[k] = []int64{pcode, licenseHash64}
			}

			//taskContainer가 참조하도록 observer 추가
			for _, observer := range observers {
				observer(pcodeLookup)
			}

			lastFileTime = ret.GetLong("filetime")
		}
	}

	return nil
}

func readHeader(reader *sidecario.NetReadHelper) (int32, error) {
	_, err := reader.ReadShort()
	if err != nil {
		return 0, err
	}
	dataLength, err := reader.ReadInt()
	if err != nil {
		return 0, err
	}

	return dataLength, nil
}

func parseResponse(payload []byte) *value.MapValue {
	buf := io.NewDataInputX(payload)
	v := value.ReadValue(buf)
	m := v.(*value.MapValue)

	return m
}

func sendRequest(conn gonet.Conn) error {
	writer := sidecario.NewNetWriteHelper(conn)
	req := value.NewMapValue()

	req.PutString("cmd", "get")
	req.PutString("id", "kube_namespace")
	req.Put("filetime", value.NewDecimalValue(lastFileTime))

	payloadbuf := io.NewDataOutputX()
	payloadbuf.WriteByte(req.GetValueType())
	req.Write(payloadbuf)
	payload := payloadbuf.ToByteArray()

	outbuf := io.NewDataOutputX()
	outbuf.WriteUShort(CAFE)
	outbuf.WriteInt(int32(len(payload)))
	outbuf.WriteBytes(payload)

	return writer.WriteBytes(outbuf.ToByteArray(), NET_TIMEOUT)
}

func newConnection() (gonet.Conn, error) {

	return gonet.DialTimeout("tcp", fmt.Sprintf("%s:%d", Host, Port), TcpConnectionTimeout)

}
