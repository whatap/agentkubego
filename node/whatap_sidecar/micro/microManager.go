package micro

import (
	"fmt"
	"log"
	gonet "net"
	"runtime"
	"time"

	"github.com/whatap/golib/io"
	"github.com/whatap/golib/lang/value"
	"whatap.io/k8s/sidecar/cache"
	sidecario "whatap.io/k8s/sidecar/io"
)

var (
	LISTEN_PORT = int32(6600)
)

const (
	READ_MAX    = 8 * 1024 * 1024
	NET_TIMEOUT = 30 * time.Second
	LOG_LIMIT   = 6
	CAFE        = 0xcafe
)

func StartMicroManager() {
	go func() {
		l, err := gonet.Listen("tcp", fmt.Sprint(":", LISTEN_PORT))
		if nil != err {
			log.Println(err)
		}
		defer l.Close()

		for {
			conn, err := l.Accept()
			if nil != err {
				log.Println(err)
				continue
			}
			go func() {
				err := serveConnForever(conn)
				log.Println("micro server err:", err)
				conn.Close()
			}()
		}
	}()
}

func serveConnForever(conn gonet.Conn) error {
	reader := sidecario.NewNetReadHelper(conn)
	writer := sidecario.NewNetWriteHelper(conn)
	for {
		dataLength, err := readHeader(reader)
		if err != nil {
			return err
		}
		if dataLength > 0 {
			payload, err := reader.ReadBytes(int(dataLength))
			if err != nil {
				return err
			}
			m, err := parseRequest(payload)
			if err != nil {
				return err
			}
			err = handleResponse(m, writer)
			if err != nil {
				return err
			}
		}
	}
}

func handleResponse(m *value.MapValue, writer *sidecario.NetWriteHelper) error {
	// log.Println("handleResponse m:", m.ToString())
	cmd := m.GetString("cmd")
	isKubeMicroEnabled := m.GetBool("kube.micro")
	if cmd == "regist" && isKubeMicroEnabled {
		resp := value.NewMapValue()
		resp.PutString("node.agent.ip", "127.0.0.1")
		resp.Put("node.agent.port", value.NewDecimalValue(6600))

		if m.ContainsKey("container_id") {
			containerId := m.GetString("container_id")
			// log.Println("micro event cid:", containerId, m.ToString())

			cache.SetMicroCache(containerId, m)
			metrics := cache.GetPerfCache(containerId)

			if metrics != nil {
				//log.Println("metrics:", metrics.ToString())
				resp.Put("cpu", metrics.Get("cpu_per_quota"))
				resp.Put("cpu_sys", metrics.Get("cpu_sys"))
				resp.Put("cpu_user", metrics.Get("cpu_user"))
				resp.Put("throttled_periods", metrics.Get("cpu_throttledperiods"))
				resp.Put("throttled_time", metrics.Get("cpu_throttledtime"))
				resp.Put("cpu_period", value.NewDecimalValue(0))
				resp.Put("cpu_quota", metrics.Get("cpu_quota"))

				//{cpu_user=11.034483,cpu_user_millis=220.68967,cpu_sys=0.591133,cpu_sys_millis=11.82266,cpu_total=11.625615,cpu_total_millis=232.5123,mem_usage=49737728,mem_totalrss=47697920,blkio_rbps=NaN,blkio_riops=NaN,blkio_wbps=NaN,blkio_wiops=NaN,cpu_per_quota=58.128075,cpu_per_request=232.5123,mem_percent=12.996652,cpu_quota=400,cpu_quota_percent=20,mem_limit=367001600,cpu_request=100,mem_request=314572800,cpu_throttledperiods=26,cpu_throttledtime=2227685249,mem_failcnt=0,mem_maxusage=49737728,mem_per_request=15.16276,mem_totalcache=0,mem_totalpgfault=16071,mem_totalrss_percent=12.996652,mem_totalunevictable=0,network_rbps=+Inf,network_rdropped=0,network_rerror=0,network_riops=+Inf,network_wbps=+Inf,network_wdropped=0,network_werror=0,network_wiops=+Inf,node_cpu=15.17588,node_mem=81.97361,restart_count=0,state=114,status=Up 36.395841113s,ready=1}

				resp.Put("memory", metrics.Get("mem_totalrss"))
				resp.Put("failcnt", metrics.Get("mem_failcnt"))
				resp.Put("limit", metrics.Get("mem_limit"))
				resp.Put("maxUsage", metrics.Get("mem_maxusage"))
				resp.Put("metering", value.NewDecimalValue(int64(runtime.NumCPU())))
			}

		}

		payloadbuf := io.NewDataOutputX()
		payloadbuf.WriteByte(resp.GetValueType())
		resp.Write(payloadbuf)
		payload := payloadbuf.ToByteArray()

		outbuf := io.NewDataOutputX()
		outbuf.WriteUShort(CAFE)
		outbuf.WriteInt(int32(len(payload)))
		outbuf.WriteBytes(payload)

		writer.WriteBytes(outbuf.ToByteArray(), NET_TIMEOUT)
	}

	return nil
}

func parseRequest(payload []byte) (*value.MapValue, error) {
	buf := io.NewDataInputX(payload)
	v := value.ReadValue(buf)
	m := v.(*value.MapValue)

	if !m.ContainsKey("cmd") {
		return nil, fmt.Errorf("request has no cmd property")
	}

	return m, nil
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
