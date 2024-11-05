package osinfo

import (
	"regexp"
)

type CmdPattern struct {
	cmd1        string
	cmd2        string
	user        string
	userPattern *regexp.Regexp
	cmd1Pattern *regexp.Regexp
	cmd2Pattern *regexp.Regexp
}

func (self *CmdPattern) parse() {
	userRegexp := regexp.MustCompile(self.user)
	cmd1Regexp := regexp.MustCompile(self.cmd1)
	cmd2Regexp := regexp.MustCompile(self.cmd2)

	self.userPattern = userRegexp
	self.cmd1Pattern = cmd1Regexp
	self.cmd2Pattern = cmd2Regexp
}

func (self *CmdPattern) matchExe(user string, exe string, cmdline string, h1 func(string)) bool {
	//fmt.Println(user, self.user, "=>", self.userPattern.MatchString(user))
	if self.userPattern.MatchString(user) {
		matches := self.cmd1Pattern.FindStringSubmatch(exe)

		if len(matches) > 1 {
			//fmt.Println("match ", exe, self.cmd1, "=>", matches)
			h1(matches[1])
			return true
		}
	}

	return false
}

func (self *CmdPattern) matchCmdline(user string, exe string, cmdline string) bool {

	return self.userPattern.MatchString(user) &&
		self.cmd1Pattern.MatchString(exe) &&
		self.cmd2Pattern.MatchString(cmdline)
}
