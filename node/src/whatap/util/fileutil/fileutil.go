package fileutil

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
)

func IsExists(abspath string) (ret bool) {
	if _, err := os.Stat(abspath); err == nil {
		ret = true
	} else {
		ret = false
	}

	return
}

// ReadFile get file content
func ReadFile(filepath string, maxlength int64) ([]byte, int64, error) {
	f, e := os.Open(filepath)
	if e != nil {
		return nil, 0, e
	}
	defer f.Close()
	var output bytes.Buffer
	buf := make([]byte, 4096)
	nbyteuntilnow := int64(0)
	for nbyteleft := maxlength; nbyteleft > 0; {
		nbytethistime, e := f.Read(buf)
		if nbytethistime == 0 || e != nil {
			break
		}
		nbyteleft -= int64(nbytethistime)
		nbyteuntilnow += int64(nbytethistime)
		output.Write(buf[:nbytethistime])
	}

	if nbyteuntilnow > 0 {
		return output.Bytes(), nbyteuntilnow, nil
	}

	return nil, 0, e
}

func Readlines(filename string, h1 func(string)) (reterr error) {
	f, err := os.Open(filename)
	if err != nil {
		reterr = err
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		h1(line)
	}

	return
}

func WriteFile(filename string, buf []byte, perm os.FileMode) (writeFileErr error) {
	if len(filename) < 0 || buf == nil || len(buf) < 1 {
		writeFileErr = fmt.Errorf("writeFile invalid param", filename, buf)
		return
	}
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, perm)
	if err != nil {
		writeFileErr = err
		return
	}
	defer f.Close()

	bufsize := len(buf)
	nbytesleft := bufsize
	for nbytesleft > 0 {
		nbytesthistime, err := f.Write(buf[bufsize-nbytesleft:])
		if err != nil {
			writeFileErr = err
			return
		}
		nbytesleft -= nbytesthistime
	}
	writeFileErr = f.Truncate(int64(bufsize))

	return
}
