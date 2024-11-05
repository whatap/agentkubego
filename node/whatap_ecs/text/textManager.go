package text

import (
	"sync"
	"time"

	"github.com/whatap/go-api/common/lang/pack"
	"github.com/whatap/go-api/common/util/dateutil"
	"github.com/whatap/go-api/common/util/hash"
	"github.com/whatap/go-api/common/util/hmap"
	"whatap.io/aws/ecs/config"
	"whatap.io/aws/ecs/session"
)

type TextKey struct {
	div  byte
	hash int32
}

func (dt *TextKey) Hash() uint {
	return uint(dt.hash ^ int32(dt.div<<32))
}
func (dt *TextKey) Equals(o hmap.LinkedKey) bool {
	other := o.(*TextKey)
	return dt.div == other.div && dt.hash == other.hash
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
	tk := &TextKey{div: div, hash: h}

	if textCache.Contains(tk) {
		return
	}
	textCache.Put(tk)

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

func (dt *DataText) process() {
	lock.Lock()
	defer lock.Unlock()
	sz := len(dt.buffer)
	if sz == 0 {
		return
	}
	rec := make([]pack.TextRec, sz)
	for i := 0; i < sz; i++ {
		rec[i] = <-dt.buffer
	}
	conf := config.GetConfig()
	p := new(pack.TextPack)
	p.Pcode = conf.PCODE
	p.Oid = conf.OID
	p.Time = dateutil.Now()
	p.AddTexts(rec)

	session.SendEncrypted(p)
}
