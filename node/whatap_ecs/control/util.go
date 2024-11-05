package control

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/whatap/go-api/common/lang/pack"
	whatap_docker "github.com/whatap/kube/node/src/whatap/util/docker"
	"whatap.io/aws/ecs/session"
)

const (
	stdWriterPrefixLen = 8
	stdWriterSizeIndex = 4
)

func sendHide(p pack.Pack) bool {
	// b := pack.ToBytesPack(p)
	// secu := secure.GetSecurityMaster()
	// b = secu.Cypher.Hide(b)
	// return sendSecure(net.NET_SECURE_HIDE, b)
	// return sendSecure(0, b)
	return session.SendHide(p)
	// return session.Send(p)
}

func sendEncrypted(p pack.Pack) bool {
	// b := pack.ToBytesPack(p)
	// return sendSecure(0, b)
	// secu := secure.GetSecurityMaster()
	// b = secu.Cypher.Encrypt(b)
	// return sendSecure(net.NET_SECURE_CYPHER, b)
	return session.SendEncrypted(p)
}

func getLastLog(containerid string, taillines int64, h1 func([]byte)) (err error) {
	cli, err := whatap_docker.GetDockerClient()

	options := types.ContainerLogsOptions{ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     false,
		Tail:       fmt.Sprint(taillines),
		Details:    false}

	resp, errlog := cli.ContainerLogs(context.Background(), containerid, options)
	if errlog != nil {
		err = errlog
		return
	}
	buflen := 1024
	buf := make([]byte, buflen)
	bufoffset := 0
	for {
		// fmt.Println("getContainerLogs step -1 bufoffset:", bufoffset)
		availablebuf := buf[bufoffset:buflen]
		nbytethistime, e := resp.Read(availablebuf)
		if e == nil || bufoffset > 0 {

			nbyteuntilnow := 0
			bufthistime := buf[:bufoffset+nbytethistime]
			// fmt.Println("getContainerLogs step -2 nbytethistime:", nbytethistime, len(bufthistime))
			for len(bufthistime) > stdWriterPrefixLen {

				frameSize := int(binary.BigEndian.Uint32(bufthistime[stdWriterSizeIndex : stdWriterSizeIndex+4]))
				if len(bufthistime) < frameSize+stdWriterPrefixLen {
					break
				}
				linebuf := bufthistime[stdWriterPrefixLen : frameSize+stdWriterPrefixLen]
				// fmt.Println("getContainerLogs step -3 frameSize:", frameSize, " len(linebuf):", len(linebuf), " nbyteuntilnow:", nbyteuntilnow)
				if len(linebuf) < frameSize {
					break
				}
				h1(linebuf)
				bufthistime = bufthistime[frameSize+stdWriterPrefixLen:]
				nbyteuntilnow += frameSize + stdWriterPrefixLen
			}
			if len(bufthistime) > 0 {
				copy(buf, bufthistime)
				bufoffset = len(bufthistime)
			} else {
				bufoffset = 0
			}
		} else {
			// fmt.Println("getContainerLogs step -4 read complete ",e )
			break
		}
	}

	return
}
