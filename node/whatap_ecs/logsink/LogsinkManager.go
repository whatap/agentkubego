package logsink

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	whatapio "github.com/whatap/go-api/common/io"
	"github.com/whatap/go-api/common/lang/pack"
	"github.com/whatap/go-api/common/lang/value"
	"github.com/whatap/go-api/common/util/dateutil"
	whataphash "github.com/whatap/go-api/common/util/hash"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/stringutil"

	"whatap.io/aws/ecs/config"
	"whatap.io/aws/ecs/session"
)

const (
	CONTAINER_LOG_CATEOGORY = "containerStdout"
	ZIP_MOD_DEFULAT_GZIP    = 1
	queueTimeout            = 1 * time.Minute
	batchSize               = 60
)

var (
	Pcode           int64
	Oid             int32
	OnodeName       string
	containerLookup = map[string]bool{}
	containers      []*ContainerLogInfo
	sendBuffer      chan pack.Pack
	sendBufferSize  = 1024
)

type ContainerLogInfo struct {
	cid       string
	cName     string
	cLogpath  string
	oNode     string
	isStarted bool
	lineNo    int64
}

func (cli *ContainerLogInfo) start() {
	cli.isStarted = true
	cli.lineNo = 1
	go cli.poll()
}

func (cli *ContainerLogInfo) poll() {
	conf := config.GetConfig()

	logfile := filepath.Join(conf.NodeVolPrefix, cli.cLogpath)

	f, fileErr := os.Open(logfile)
	if fileErr != nil {
		log.Println("LM62 Error:", fileErr)
		return
	}
	defer f.Close()

	_, seekErr := f.Seek(0, os.SEEK_END)
	if seekErr != nil {
		log.Println("LM69 Error:", fileErr)
		return
	}

	for {
		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			line := scanner.Text()
			length := len(line)
			if length > 0 {

				cli.lineNo += int64(length)

				logsinkPack := createLogSink(cli.cName, cli.cid, cli.cLogpath, cli.oNode)
				logsinkPack.Line = cli.lineNo
				logsinkPack.Content = line

				if logsinkPack != nil && len(sendBuffer) < (sendBufferSize-1) {
					sendBuffer <- logsinkPack
				}

			}
		}

		time.Sleep(time.Second * 10)
	}
}

func createLogSink(cName string, cid string, logpath string, oNode string) *pack.LogSinkPack {
	p := pack.NewLogSinkPack()
	p.Pcode = Pcode
	p.Oid = Oid
	p.Category = CONTAINER_LOG_CATEOGORY
	p.Time = dateutil.Now()

	p.Tags.PutString("container", cName)
	p.Tags.PutString("containerId", cid)
	p.Tags.Put("containerKey", value.NewDecimalValue(int64(whataphash.HashStr(cid))))
	p.Tags.PutString("onodeName", oNode)
	p.Tags.Put("onode", value.NewDecimalValue(int64(whataphash.HashStr(oNode))))

	return p
}

func OnContainerDetected(cid string, cName string, clogpath string, oNode string) {

	if _, ok := containerLookup[cid]; !ok {
		containerLookup[cid] = true

		cli := &ContainerLogInfo{cName: cName, cid: cid, cLogpath: clogpath, oNode: oNode}
		cli.start()

		containers = append(containers, cli)
	}
}

func findAllContainers() {
	conf := config.GetConfig()
	for {
		findAllContainersOnNode(func(c types.ContainerJSON) {
			if stringutil.IsNotEmpty(conf.LogsinkExcludePattern) && strings.Contains(c.Name, conf.LogsinkExcludePattern) {
				return
			}
			OnContainerDetected(c.ID, c.Name, c.LogPath, conf.ONODE)
		})

		time.Sleep(time.Second * 10)
	}

}

func StartPolling() {
	sendBuffer = make(chan pack.Pack, sendBufferSize)

	go findAllContainers()
	go startQueueListener()
}

func startQueueListener() {

	for {
		var packscliTime []pack.Pack
		lastPackReceive := int64(0)
		for {
			now := dateutil.Now()
			select {
			case p := <-sendBuffer:
				packscliTime = append(packscliTime, p)
				if len(packscliTime) == batchSize || now-lastPackReceive > queueTimeout.Milliseconds() {
					sendZippedPacks(packscliTime)

					packscliTime = nil
					lastPackReceive = 0
				}
				if lastPackReceive == 0 {
					lastPackReceive = now
				}
			case <-time.After(queueTimeout):
				if len(packscliTime) > 0 {
					sendZippedPacks(packscliTime)
					// sendPacks(packscliTime)
					packscliTime = nil
					lastPackReceive = 0
				}
			}
		}
	}
}

func sendPacks(packs []pack.Pack) {
	for _, p := range packs {
		send(p)
	}
}

func sendZippedPacks(packs []pack.Pack) {
	// log.Println("sendZippedPacks step -1")
	zp := pack.NewZipPack()

	zp.Pcode = Pcode
	zp.Oid = Oid
	zp.Time = dateutil.Now()
	zp.RecountCount = len(packs)
	zp.Records = getStream(packs)

	// log.Println("sendZippedPacks step -2", zp.RecountCount)
	err := doZip(zp)
	// log.Println("sendZippedPacks step -3", err)
	if err != nil {
		// log.Println("sending log sink", err)
		return
	}
	// log.Println("sendZippedPacks step -4")
	send(zp)
}

func doZip(p *pack.ZipPack) error {
	buf := new(bytes.Buffer)

	gz := gzip.NewWriter(buf)
	_, gzipErr := gz.Write(p.Records)
	if gzipErr != nil {
		return gzipErr
	}
	gz.Close()

	p.Status = ZIP_MOD_DEFULAT_GZIP
	p.Records = buf.Bytes()

	return nil
}

func getStream(packs []pack.Pack) []byte {
	dout := whatapio.NewDataOutputX()

	for _, p := range packs {
		pack.WritePack(dout, p)
	}

	return dout.ToByteArray()
}

func send(p pack.Pack) bool {

	return session.Send(p)
}
