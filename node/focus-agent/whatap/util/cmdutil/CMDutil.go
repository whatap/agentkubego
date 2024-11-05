package cmdutil

import (
	"bytes"
	"container/list"

	//"log"

	"os"
	"os/exec"

	//"syscall"
	//"runtime/debug"

	"strings"

	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/logutil"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/stringutil"
)

// Pipeline strings together the given exec.Cmd commands in a similar fashion
// to the Unix pipeline.  Each command's standard output is connected to the
// standard input of the next command, and the output of the final command in
// the pipeline is returned, along with the collected standard error of all
// commands and the first error found (if any).
//
// To provide input to the pipeline, assign an io.Reader to the first's Stdin.
func Pipeline(cmds ...*exec.Cmd) (pipeLineOutput, collectedStandardError []byte, pipeLineError error) {
	defer func() {
		for _, cmd := range cmds {
			//syscall.Kill(cmd.Process.Pid, syscall.SIGKILL)
			cmd.Process.Kill()
		}
	}()
	// Require at least one command
	if len(cmds) < 1 {
		return nil, nil, nil
	}

	// Collect the output from the command(s)
	var output bytes.Buffer
	var stderr bytes.Buffer

	last := len(cmds) - 1
	for i, cmd := range cmds[:last] {
		var err error
		// Connect each command's stdin to the previous command's stdout
		if cmds[i+1].Stdin, err = cmd.StdoutPipe(); err != nil {
			//logutil.Infoln("cmd.StdoutPipe() cmd=", cmd.Path, " error", err)
			return nil, nil, err
		}
		// Connect each command's stderr to a buffer
		cmd.Stderr = &stderr
	}

	// Connect the output and error for the last command
	cmds[last].Stdout, cmds[last].Stderr = &output, &stderr

	// Start and Wait each command
	// 2018.8.21 먼저 Start를 다 시키고 나서, Wait 실행 중 중간 cmd 에서 에러가 나면 나머지 cmd 는 좀비 프로세스로 변함
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			logutil.Println("WA30200", "PipeLine Start Error, Path=", cmd.Path, ", err=", err)
			return output.Bytes(), stderr.Bytes(), err
		}
	}

	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			logutil.Println("WA30201", "PipeLine Wait Error, Path=", cmd.Path, ", err=", err)
			return output.Bytes(), stderr.Bytes(), err
		}
	}

	// Return the pipeline output and the collected standard error
	return output.Bytes(), stderr.Bytes(), nil
}

func GetPHPInfo() map[string]string {
	defer func() {
		// recover
		if r := recover(); r != nil {
			//
			//log.Println("recover:", r, string(debug.Stack()))
		}
	}()
	m := make(map[string]string)
	phpinfo := cmdPHPInfo()
	phpinfo = strings.Replace(phpinfo, "\r", "", -1)

	// PHP Version
	phpVersion := SubstringBetween(phpinfo, "PHP Version", "\n\nConfiguration\n\n")
	s1 := strings.Split(phpVersion, "\n")

	for _, tmp := range s1 {
		k, v := ToPair(tmp, "=>")
		if k != "" {
			m[k] = v
		}
	}

	return m
}

func GetPHPModuleInfo() map[string]string {
	defer func() {
		// recover
		if r := recover(); r != nil {
			//
			//log.Println("recover:", r, string(debug.Stack()))
		}
	}()
	m := make(map[string]string)
	keysList := list.New()

	//	pos := -1
	//	pos1 := -1
	//	mpos := -1
	//	mpos1 := -1

	//php -m
	moduleinfo := cmdPHPModuleInfo()
	moduleinfo = strings.Replace(moduleinfo, "\r", "", -1)

	phpmodules := SubstringBetween(moduleinfo, "[PHP Modules]", "[Zend Modules]")
	s1 := strings.Split(phpmodules, "\n")
	// key 등록
	for _, tmp := range s1 {
		if strings.TrimSpace(tmp) != "" {
			m[tmp] = ""
			keysList.PushBack(tmp)
			//log.Println("PHP Module key= ", tmp)
		}
	}

	zendmodules := SubstringBetween(moduleinfo, "[Zend Modules]", "")

	s2 := strings.Split(zendmodules, "\n")
	// key 등록
	for _, tmp := range s2 {
		if strings.TrimSpace(tmp) != "" {
			m[tmp] = ""
			keysList.PushBack(tmp)
			//log.Println("Zend Module key= ", tmp)
		}
	}

	mLen := keysList.Len()
	keys := make([]string, mLen)
	idx := 0
	for e := keysList.Front(); e != nil; e = e.Next() {
		//log.Println("keys=", e)
		keys[idx] = string(e.Value.(string))
		idx++
	}

	//phpI := exec.Command(php, "-i")
	phpinfo := cmdPHPInfo()
	phpinfo = strings.Replace(phpinfo, "\r", "", -1)

	// Configuration
	str := SubstringBetween(phpinfo, "\n\nConfiguration\n\n", "\n\nAdditional Modules\n\n")
	//log.Println("Configuration=", str)
	for i := 0; i < mLen; i++ {
		detail := ""
		if i+1 < mLen {
			detail = SubstringBetween(str, "\n\n"+keys[i], "\n\n"+keys[i+1])
		} else {
			detail = SubstringBetween(str, "\n\n"+keys[i], "")
		}

		s3 := stringutil.Tokenizer(detail, "\n")
		m[keys[i]] = strings.Join(s3, ", ")
	}

	return m
}

func cmdPHPInfo() string {
	defer func() {
		if r := recover(); r != nil {
		}
	}()
	php := os.Getenv("WHATAP_PHP_BIN")
	if strings.TrimSpace(php) != "" {
		cmd := exec.Command(php, "-i")
		out, err := cmd.Output()

		if err != nil {
			//error
			//log.Println("command err", err)
			return ""
		}

		return string(out)
	}
	return ""
}

func cmdPHPModuleInfo() string {
	defer func() {
		if r := recover(); r != nil {
		}
	}()
	php := os.Getenv("WHATAP_PHP_BIN")
	if strings.TrimSpace(php) != "" {
		cmd := exec.Command(php, "-m")
		out, err := cmd.Output()

		if err != nil {
			//error
			//log.Println("command err", err)
			return ""
		}
		return string(out)
	}
	return ""
}

func SubstringBetween(s string, from string, to string) string {
	defer func() {
		// recover
		if r := recover(); r != nil {
			//
			//log.Println("recover:", r, string(debug.Stack()))
		}
	}()

	pos := 0
	pos1 := 0
	result := ""

	pos = strings.Index(strings.ToLower(s), strings.ToLower(from))
	if to != "" {
		pos1 = strings.Index(strings.ToLower(s), strings.ToLower(to))
	} else {
		pos1 = -1
	}

	if pos != -1 {
		pos += len(from)
		if pos1 != -1 {
			result = s[pos:pos1]
		} else {
			result = s[pos:]
		}
	} else {
		return ""
	}
	return strings.TrimSpace(result)
}

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

// Get docker full id from /proc/self/cgroup
func GetDockerFullId() string {

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	// check exists /proc/self/cgroup
	if _, err := os.Stat("/proc/self/cgroup"); os.IsNotExist(err) {
		// path/to/whatever does not exist
		return ""
	}
	//cat /proc/self/cgroup | head -n 1 | cut -d '/' -f3
	c1 := exec.Command("cat", "/proc/self/cgroup")
	c2 := exec.Command("head", "-n", "1")
	c3 := exec.Command("cut", "-d", "/", "-f3")

	// Run the pipeline
	out, _, err := Pipeline(c1, c2, c3)
	if err != nil {
		logutil.Println("WA30203", "GetDockerFullId Error : errors : ", err)
		return ""
	}
	return strings.TrimSuffix(string(out), "\n")
}

func CMDMain() {
	c1 := exec.Command("ps", "aux")
	c2 := exec.Command("grep", "httpd")
	c3 := exec.Command("awk", "{print $3}")
	c4 := exec.Command("awk", "{total = total + $1} END {print total}")

	// Run the pipeline
	//output, stderr, err := Pipeline(c1, c2, c3, c4)
	output, _, err := Pipeline(c1, c2, c3, c4)
	if err != nil {
		//logutil.Printf("Error : %s", err)
	}

	// Print the stdout, if any
	if len(output) > 0 {
		//logutil.Printf("output %s", output)

	}
}
