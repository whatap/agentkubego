package text

import (
	"log"
	"sync"
	"time"

	"github.com/whatap/golib/lang/pack"
	"github.com/whatap/golib/util/hash"
	"github.com/whatap/golib/util/hmap"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/dateutil"
	"whatap.io/k8s/sidecar/config"
	"whatap.io/k8s/sidecar/session"
)

type TextKey struct {
	div  byte
	hash int32
}

func (this *TextKey) Hash() uint {
	return uint(this.hash ^ int32(this.div<<32))
}
func (this *TextKey) Equals(o hmap.LinkedKey) bool {
	other := o.(*TextKey)
	return this.div == other.div && this.hash == other.hash
}

type DataText struct {
	buffer chan pack.TextRec
}

var dataText *DataText
var lock = sync.Mutex{}

func InitializeTextSender() *DataText {
	lock.Lock()
	defer lock.Unlock()

	if dataText != nil {
		return dataText
	}
	dataText = new(DataText)
	dataText.buffer = make(chan pack.TextRec, 5120)
	go func() {
		for {
			dataText.process()
			processOnewayPacks()
			time.Sleep(1000 * time.Millisecond)
		}
	}()
	return dataText
}

var textCache *hmap.LinkedSet = hmap.NewLinkedSet().SetMax(100000)

func SendText(div byte, text string) {

	var this *DataText
	if dataText != nil {
		this = dataText
	} else {
		this = InitializeTextSender()
	}

	h := hash.HashStr(text)

	this.buffer <- pack.TextRec{Div: div, Hash: h, Text: text}
}
func SendHashText(div byte, h int32, text string) {
	var this *DataText
	if dataText != nil {
		this = dataText
	} else {
		this = InitializeTextSender()
	}
	this.buffer <- pack.TextRec{Div: div, Hash: h, Text: text}
}

func SendHashTextOneway(pcode int64, licenseHash int64, div byte, h int32, text string) {
	log.Println("SendHashTextOneway step -1 pcode:", pcode, " licenseHash:", licenseHash, div, h, text)
	c := getChannel(pcode)
	log.Println("SendHashTextOneway step -2 ", c)
	if nil == c {
		c = make(chan pack.TextRec, 5120)
		addChannelListener(pcode, c, func() {
			log.Println("SendHashTextOneway step -2.1 pcode:", pcode)
			sz := len(c)
			if sz == 0 {
				return
			}
			log.Println("SendHashTextOneway step -2.2 len:", sz)
			rec := make([]pack.TextRec, sz)
			for i := 0; i < sz; i++ {
				rec[i] = <-c
			}
			log.Println("SendHashTextOneway step -2.3")
			p := new(pack.TextPack)
			p.Pcode = pcode
			p.Oid = 0
			p.Time = dateutil.Now()
			p.AddTexts(rec)
			log.Println("SendHashTextOneway step -2.4")

			session.SendOneway(pcode, licenseHash, p)
			log.Println("SendHashTextOneway step -2.5")
		})
	}
	c <- pack.TextRec{Div: div, Hash: h, Text: text}
	log.Println("SendHashTextOneway step -3")
}

func processOnewayPacks() {
	for _, cc := range listeners {
		cc.listener()
	}
}

type TextChannelComplex struct {
	pcode    int64
	buffer   chan pack.TextRec
	listener func()
}

var (
	listeners []TextChannelComplex
)

func addChannelListener(pcode int64, buffer chan pack.TextRec, listener func()) {
	for _, cc := range listeners {
		if cc.pcode == pcode {
			return
		}
	}
	listeners = append(listeners, TextChannelComplex{pcode: pcode, buffer: buffer, listener: listener})
}

func getChannel(pcode int64) chan pack.TextRec {
	for _, cc := range listeners {
		if cc.pcode == pcode {
			return cc.buffer
		}
	}
	return nil
}

func (this *DataText) process() {
	lock.Lock()
	defer lock.Unlock()
	sz := len(this.buffer)
	if sz == 0 {
		return
	}
	rec := make([]pack.TextRec, sz)
	for i := 0; i < sz; i++ {
		rec[i] = <-this.buffer
	}
	conf := config.GetConfig()
	p := new(pack.TextPack)
	p.Pcode = conf.PCODE
	p.Oid = conf.OID
	p.Time = dateutil.Now()
	p.AddTexts(rec)

	session.SendEncrypted(p)
}
