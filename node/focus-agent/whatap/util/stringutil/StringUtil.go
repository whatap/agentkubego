package stringutil

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

func Tokenizer(src, delim string) []string {
	if src == "" || delim == "" {
		return []string{src}
	}
	chars := []rune(delim)
	f := func(c rune) bool {
		for i := 0; i < len(chars); i++ {
			if chars[i] == c {
				return true
			}
		}
		return false
	}
	fields := strings.FieldsFunc(src, f)
	return fields
}
func FirstWord(target, delim string) string {
	if target == "" || delim == "" {
		return target
	}

	out := Tokenizer(target, delim)

	if len(out) >= 1 {
		return strings.TrimSpace(out[0])
	} else {
		return ""
	}
}
func LastWord(target, delim string) string {
	if target == "" || delim == "" {
		return target
	}
	out := Tokenizer(target, delim)

	if len(out) >= 1 {
		return strings.TrimSpace(out[len(out)-1])
	} else {
		return ""
	}
}
func Cp949toUtf8(src []byte) string {
	var b bytes.Buffer
	wInUTF8 := transform.NewWriter(&b, korean.EUCKR.NewDecoder())
	wInUTF8.Write(src)
	wInUTF8.Close()
	return b.String()
}

// 2017.5.24 Java string.hashCode 함수 대체
func HashCode(s string) int {
	h := 0
	for i := 0; i < len(s); i++ {
		h = 31*h + int(s[i])
	}
	return h
}

func TrimEmpty(s string) string {
	if s == "" {
		return s
	} else {
		return strings.TrimSpace(s)
	}
}

func Truncate(str string, length int) string {
	if str == "" || len(str) <= length {
		return str
	} else {
		return str[0:length]
	}
}

// TODO Tckenizer, Split 테스트 필요
func Split(s, sep string) []string {
	return strings.Split(s, sep)
}

// substring
func Substring(s string, from string, to string) string {
	defer func() {
		// recover
		if r := recover(); r != nil {
			//log.Println("recover:", r, string(debug.Stack()))
		}
	}()
	pos := 0
	pos1 := 0

	pos = strings.Index(strings.ToLower(s), strings.ToLower(from))
	if pos < 0 {
		return ""
	}
	pos += len(from)

	if to != "" {
		pos1 = strings.Index(strings.ToLower(s[pos:]), strings.ToLower(to))
		if pos1 < 0 {
			pos1 = len(s)
		} else {
			pos1 = pos + pos1
		}
	} else {
		pos1 = len(s)
	}

	return strings.TrimSpace(s[pos:pos1])
}

// substring
func SubstringN(s string, from string, to string, n int) []string {
	defer func() {
		// recover
		if r := recover(); r != nil {
			//fmt.Println("recover:", r, string(debug.Stack()))
		}
	}()
	result := make([]string, 0)

	lastPos := 0
	pos := 0
	pos1 := 0
	idx := 0
	for pos = strings.Index(strings.ToLower(s), strings.ToLower(from)); pos >= 0; pos = strings.Index(strings.ToLower(s[lastPos:]), strings.ToLower(from)) {

		fmt.Println("pos=", pos)
		if pos < 0 {
			break
		}
		pos += len(from)

		if to != "" {
			pos1 = strings.Index(strings.ToLower(s[lastPos+pos:]), strings.ToLower(to))
			if pos1 < 0 {
				pos1 = len(s)
			} else {
				pos1 = pos + pos1
			}
		} else {
			pos1 = len(s)
		}

		str := strings.TrimSpace(s[lastPos+pos : lastPos+pos1])
		result = append(result, str)

		lastPos = lastPos + pos1 + len(to)
		idx++

		if n != -1 && idx >= n {
			break
		}

		if lastPos >= len(s) {
			break
		}
	}
	return result
}

// substring Prefix "abcd efg hijk" from[]{"a","e","h"} => result[]{"abcd", "efg", "hijk"}
func SubstringWords(s string, from []string) map[string]string {
	defer func() {
		// recover
		if r := recover(); r != nil {
			//log.Println("recover:", r, string(debug.Stack()))
		}
	}()
	result := make(map[string]string)
	//result := make([]string, 0)

	lastPos := 0
	pos := -1
	pos1 := -1

	for i, it := range from {
		pos = strings.Index(strings.ToLower(s[lastPos:]), strings.ToLower(it))
		if i == 0 {
			pos1 = pos
		} else {
			if pos1 > -1 && pos > pos1 {
				str := strings.TrimSpace(s[pos1:pos])
				//result = append(result, str)
				result[it] = str

				pos1 = pos
			}
		}
	}

	return result
}

// get k, v
func ToPair(s string, sep string) (k, v string) {
	pos := 0
	pos = strings.Index(strings.ToLower(s), strings.ToLower(sep))
	if pos != -1 {
		k = s[0:pos]
		v = s[pos+len(sep):]
	} else {
		k = ""
		v = ""
	}

	return strings.TrimSpace(k), strings.TrimSpace(v)
}

func LPad(str string, n int) string {
	if str == "" {
		return padding(n, " ")
	}
	slen := len(str)
	if slen >= n {
		return str
	}
	return padding(n-slen, " ") + str
}
func RPad(str string, n int) string {
	if str == "" {
		return padding(n, " ")
	}
	slen := len(str)
	if slen >= n {
		return str
	}
	return str + padding(n-slen, " ")
}
func padding(n int, ch string) string {
	buf := bytes.Buffer{}
	for i := 0; i < n; i++ {
		buf.WriteString(ch)
	}
	return buf.String()
}

func IsNotEmpty(s string) bool {
	if s == "" {
		return false
	} else {
		return true
	}
}

func LPadInt(v, size int) string {
	var ret string
	ret = fmt.Sprintf("%d", v)
	if len(ret) > size {
		return ret
	}
	return padding(size-len(ret), "0") + ret
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

var linuxPattern = regexp.MustCompile(`\\[0-9]{3}`)

func EscapeSpace(a string) string {
	for _, m := range linuxPattern.FindAllString(a, -1) {
		b := m[1:]
		i, err := strconv.ParseInt(b, 8, 32)
		if err == nil {
			c := fmt.Sprintf("%c", i)
			a = strings.Replace(a, m, c, -1)
		}
	}

	return a
}

func NullTermToStrings(b []byte) (s []string) {
	for {
		i := bytes.IndexByte(b, byte(0))
		if i == -1 {
			break
		}
		s = append(s, string(b[0:i]))
		b = b[i+1:]
		if b[0] == byte(0) {
			break
		}
	}
	return
}

func Contains(tokens []string, src string) (ret bool) {
	ret = false
	for _, t := range tokens {
		if t == src {
			ret = true

			return
		}
	}

	return
}

func TrimAllSpace(str string) string {
	rd := bufio.NewReader(strings.NewReader(str))
	var buf bytes.Buffer
	for {
		if r, _, err := rd.ReadRune(); err == nil {
			if !unicode.IsSpace(r) {
				buf.WriteRune(r)
			}
		} else {
			break
		}
	}
	return buf.String()
}

func ToInt64(token string) (ret int64) {
	converted, e := strconv.ParseInt(token, 10, 64)
	if e == nil {
		ret = converted
	}

	return
}

func Split2(src, delim string) (string, string) {
	tokens := Tokenizer(src, delim)

	if len(tokens) > 1 {

		return tokens[0], tokens[1]
	}
	return "", ""
}
