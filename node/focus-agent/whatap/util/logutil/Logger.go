package logutil

import (
	"fmt"
	"log"
	"runtime/debug"

	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/ansi"
)

func build(id, message string) string {
	return fmt.Sprint("[", id, "] ", message)
}

func Writeln(v ...interface{}) {
	log.Println(fmt.Sprint(v...))
}

func Println(id string, v ...interface{}) {
	log.Println(build(id, fmt.Sprint(v...)))
}
func Errorln(id string, v ...interface{}) {
	log.Println(ansi.Red(build(id, fmt.Sprint(v...))))
}

func Printf(id string, format string, v ...interface{}) {
	log.Println(build(id, fmt.Sprintf(format, v...)))
}

func Errorf(id string, format string, v ...interface{}) {
	log.Println(ansi.Red(build(id, fmt.Sprintf(format, v...))))
}
func PrintlnError(id, message string, t error) {
	log.Println(build(id, message), t)
}

func GetCallStack() string {
	defer func() {
		if r := recover(); r != nil {
			log.Println("WA10001 getCallStack Recover", r)
		}
	}()
	return string(debug.Stack())
}

//
//func Info(id string, message string) {
//	logger.info(id, message)
//}
//
//func Infoln(id string, v ...interface{}) {
//	logger.info(id, fmt.Sprint(v...))
//}
//func Infof(id string, format string, v ...interface{}) {
//	logger.info(id, fmt.Sprintf(format, v...))
//}

//
//func (this *Logger) printlnStd(msg string, sysout bool) {
//	defer func() {
//		if r := recover(); r != nil {
//			log.Println("WA10002", "println Recover", r)
//		}
//	}()
//	if sysout {
//		fmt.Println(msg)
//	} else {
//		log.Println(msg)
//	}
//}

//
//func (this *Logger) info(id string, message string) {
//	this.printlnStd(build(id, message), false)
//}
//
//func Sysout(message string) {
//	logger.sysout(message)
//}
//
//func (this *Logger) sysout(message string) {
//	fmt.Println(message)
//}
