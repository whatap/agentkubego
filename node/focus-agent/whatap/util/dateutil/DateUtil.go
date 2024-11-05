package dateutil

import (
	//"log"
	"bytes"
	"fmt"
	"strings"
	"time"
)

var helper = getDateTimeHelper("")

func DateTime(time int64) string {
	return helper.datetime(time)
}
func TimeStamp(time int64) string {
	return helper.timestamp(time)
}
func TimeStampNow() string {
	return helper.timestamp(Now())
}
func WeekDay(time int64) string {
	return helper.weekday(time)
}
func GetDateUnitNow() int64 {
	return helper.getDateUnit(Now())
}

func GetDateUnit(time int64) int64 {
	return helper.getDateUnit(time)
}
func Ymdhms(time int64) string {
	return helper.ymdhms(time)
}
func YYYYMMDD(time int64) string {
	return helper.yyyymmdd(time)
}
func HHMMSS(time int64) string {
	return helper.hhmmss(time)
}
func HHMM(time int64) string {
	return helper.hhmm(time)
}
func YmdNow() string {
	return helper.yyyymmdd(Now())
}

var delta int64 = 0

func SystemNow() int64 {
	return (time.Now().UnixNano() / 1000000)
}
func Systemyymmdd() string {
	h := getDateTimeHelper(time.Local.String())
	yymmdd := h.yyyymmdd(SystemNow())
	return yymmdd
}
func Now() int64 {
	t := SystemNow()
	return t + delta
}
func SetDelta(t int64) {
	delta = t
}

func SetServerTime(serverTime int64, syncfactor float64) int64 {
	now := SystemNow()
	delta = serverTime - now
	if delta != 0 {
		delta = int64(float64(delta) * syncfactor)
	}
	return delta
}
func GetDelta() int64 {
	return delta
}

//
//	func timestamp(time int64 ) string{
//		return helper.timestamp(time);
//	}
//
//	func yyyymmdd(time int64 ) string{
//		return helper.yyyymmddStr(time);
//	}

func GetFiveMinUnit(time int64) int64 {
	return helper.getFiveMinUnit(time)
}

func GetMinUnit(time int64) int64 {
	return helper.getMinUnit(time)
}

func GetYmdTime(yyyyMMdd string) int64 {
	return helper.getYmdTime(yyyyMMdd)
}

// y- 2020, m - 03, d - 31, H - 23, M - 59, S - 59
func DateFormat(t time.Time, format string) string {
	ret := format
	if strings.Index(ret, "y") > -1 {
		tm := int(t.Year())
		ret = strings.Replace(ret, "y", LPadInt(tm, 2), -1)
	}
	if strings.Index(ret, "m") > -1 {
		tm := int(t.Month())
		ret = strings.Replace(ret, "m", LPadInt(tm, 2), -1)
	}
	if strings.Index(ret, "d") > -1 {
		tm := int(t.Day())
		ret = strings.Replace(ret, "d", LPadInt(tm, 2), -1)
	}
	if strings.Index(ret, "H") > -1 {
		tm := int(t.Hour())
		ret = strings.Replace(ret, "H", LPadInt(tm, 2), -1)
	}
	if strings.Index(ret, "M") > -1 {
		tm := int(t.Minute())
		ret = strings.Replace(ret, "M", LPadInt(tm, 2), -1)
	}
	if strings.Index(ret, "S") > -1 {
		tm := int(t.Second())
		ret = strings.Replace(ret, "S", LPadInt(tm, 2), -1)
	}

	return ret
}

func padding(n int, ch string) string {
	buf := bytes.Buffer{}
	for i := 0; i < n; i++ {
		buf.WriteString(ch)
	}
	return buf.String()
}

func LPadInt(v, size int) string {
	var ret string
	ret = fmt.Sprintf("%d", v)
	if len(ret) > size {
		return ret
	}
	return padding(size-len(ret), "0") + ret
}

func SysNow() int64 {
	t := time.Now()
	return t.Unix() * 1000
}
