package stringutil

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

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

func Split2(src , delim string) (string,string) {
	tokens := Tokenizer(src, delim )

	if len(tokens) > 1{

		return tokens[0], tokens[1]
	}
	return "",""
}


func ToInt64(token string) (ret int64) {
	converted, e := strconv.ParseInt( token, 10, 64)
	if e == nil{
		ret = converted
	}

	return 
}