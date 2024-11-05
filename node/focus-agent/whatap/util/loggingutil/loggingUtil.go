package loggingutil

import (
	"bufio"
	"bytes"
	"io"
	"os"
)

func ReadLog(filename string, endpos int64, length int64) (before int64, next int64, logText string, e error) {
	finfo, err := os.Lstat(filename)
	if err != nil {
		e = err
		return
	}
	filesize := finfo.Size()
	fileOpen, err := os.Open(filename)
	if err != nil {
		e = err
		return
	}
	defer fileOpen.Close()

	filepos := int64(0)
	if endpos >= 0 && endpos < filesize {
		filepos = endpos
		_, err = fileOpen.Seek(endpos, io.SeekStart)
		if err != nil {
			e = err
			return
		}
		before = endpos - length
		if before < 0 {
			before = 0
		}
	} else {
		filepos = filesize - length
		if filepos < 0 {
			filepos = 0
		}
		_, err = fileOpen.Seek(filepos, io.SeekStart)
		if err != nil {
			e = err
			return
		}
		before = filepos - length
	}
	next += filepos

	fileScanner := bufio.NewScanner(fileOpen)

	var buffer bytes.Buffer
	isFileEnd := true
	nbytesUntilNow := int64(0)
	for fileScanner.Scan() {
		l := fileScanner.Text()
		nbytesUntilNow += int64(len(l)) + 1
		buffer.WriteString(l)
		buffer.WriteString("\n")
		if nbytesUntilNow >= length {
			isFileEnd = false
			break
		}
	}

	if endpos < 0 || isFileEnd {
		next = -1
	} else {
		next += nbytesUntilNow
	}

	logText = buffer.String()

	return
}
