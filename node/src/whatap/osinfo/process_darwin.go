//go:build darwin
// +build darwin

package osinfo

import (
	"encoding/binary"
	"fmt"
	"github.com/whatap/kube/node/src/whatap/util/panicutil"
	"github.com/whatap/kube/node/src/whatap/util/stringutil"
	"golang.org/x/sys/unix"
	"strings"
	"time"
	"unsafe"
)

/*
#include <sys/cdefs.h>
#include <sys/sysctl.h>
#include <libproc.h>
#include <unistd.h>

	pid_t get_pid(struct kinfo_proc* p){
		return p->kp_proc.p_pid;
	}
	pid_t get_ppid(struct kinfo_proc* p){
		return p->kp_eproc.e_ppid;
	}

	size_t get_kinfo_size(){
		return sizeof(struct kinfo_proc);
	}

	size_t get_proc_taskinfo_size(){
		return sizeof(struct proc_taskinfo);
	}

	int get_uid(struct kinfo_proc* p){
		return p->kp_eproc.e_pcred.p_ruid;
	}

	long get_start_time(struct kinfo_proc* p){
		return p->kp_proc.p_starttime.tv_sec*1000+p->kp_proc.p_starttime.tv_usec/1000;
	}

	char get_status(struct kinfo_proc* p){
		return p->kp_proc.p_stat;
	}

	uint64_t get_cpu_user(struct proc_taskinfo* pti){
		return pti->pti_total_user;
	}

	uint64_t get_cpu_sys(struct proc_taskinfo* pti){
		return pti->pti_total_system;
	}

	uint64_t get_resident_size(struct proc_taskinfo* pti){
		return pti->pti_resident_size;
	}

	uint64_t get_virtual_size(struct proc_taskinfo* pti){
		return pti->pti_virtual_size;
	}


	int32_t get_page_faults(struct proc_taskinfo* pti){
		return pti->pti_faults;
	}


*/
import "C"

const (
	CTLKern          = 1
	KernProc         = 14
	KernProcPID      = 1
	KernArgMax       = 8
	KernProcPathname = 62
	KernProcArgs     = 38
	KernProcAll      = 0
	KERN_PROCARGS2   = 49

	ProcPidTaskInfo = 4

	CtlHw     = 6
	HwMemSize = 24
)

type PosixProcessParser struct {
	cgo_processlist []byte

	ProcessList []*ProcessInfo
}

func NewPosixProcessParser() *PosixProcessParser {

	return &PosixProcessParser{}
}

func (this *PosixProcessParser) Populate() (err error) {
	mib := []int32{CTLKern, KernProc, KernProcAll}
	buf, length, err := CallSyscall(mib)
	if err != nil {
		panicutil.Debug(err)
		return err
	}
	sizeOfKinfoProc := int(C.get_kinfo_size())
	count := int(length / uint64(sizeOfKinfoProc))
	totalMemory, err := GetTotalMemorySize()
	if err != nil {
		totalMemory = 0
		fmt.Println("get total memory error: ", err)
	}

	for i := 0; i < count; i++ {
		b := buf[i*int(sizeOfKinfoProc) : (i+1)*int(sizeOfKinfoProc)]
		k := ParseKinfoProc(b)
		k.Cmd1 = getProcName(k.Pid)
		if len(k.Cmd1) < 1 {
			continue
		}

		mib = []int32{CTLKern, KERN_PROCARGS2, int32(k.Pid)}

		buf, _, err := CallSyscallEx(mib)
		if err == nil {
			tokens := stringutil.NullTermToStrings(buf[4:])
			k.Cmd2 = strings.Join(tokens, " ")
		}

		buf, errpidinfo := ProcPidInfo(k.Pid, ProcPidTaskInfo, 0)
		if errpidinfo == nil {
			ParsePidTaskInfo(buf, &k)
			if totalMemory > 0 {
				k.MemoryPercent = float32(k.MemoryBytes*100) / float32(totalMemory)
			}

		}

		this.ProcessList = append(this.ProcessList, &k)
	}

	return
}

func getProcName(pid int) string {
	length := 1024
	buf := make([]byte, length)

	C.proc_name(C.int(pid), unsafe.Pointer(&buf[0]), C.uint(length))

	return stringutil.NullTermToStrings(buf)[0]

}

var maxarg = int32(0)

func getMaxArg() (int32, error) {
	var err error = nil
	if maxarg < 1 {
		mib := []int32{CTLKern, KernArgMax}
		miblen := uint64(len(mib))
		// get required buffer size
		length := int32(4)
		_, _, err = unix.Syscall6(
			unix.SYS___SYSCTL,
			uintptr(unsafe.Pointer(&mib[0])),
			uintptr(miblen),
			uintptr(unsafe.Pointer(&maxarg)),
			uintptr(unsafe.Pointer(&length)),
			0,
			0)
	}
	return maxarg, err
}

// CallSyscallEx CallSyscallEx
func CallSyscallEx(mib []int32) ([]byte, int32, error) {
	length, _ := getMaxArg()

	buf := make([]byte, length)

	return callSyscallEx(mib, buf, length)
}

// CallSyscallEx CallSyscallEx
func callSyscallEx(mib []int32, buf []byte, buflength int32) ([]byte, int32, error) {
	miblen := uint64(len(mib))

	length := buflength

	_, _, err := unix.Syscall6(
		unix.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(miblen),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&length)),
		0,
		0)
	if err != 0 {
		return buf, length, err
	}

	return buf, length, nil
}

// CallSyscall CallSyscall
func CallSyscall(mib []int32) ([]byte, uint64, error) {
	miblen := uint64(len(mib))

	// get required buffer size
	length := uint64(0)
	_, _, err := unix.Syscall6(
		unix.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(miblen),
		0,
		uintptr(unsafe.Pointer(&length)),
		0,
		0)
	if err != 0 {
		fmt.Println("syscall pref failed:", err)
		var b []byte
		return b, length, err
	}
	if length == 0 {
		fmt.Println("syscall content 0 ", err)
		var b []byte
		return b, length, err
	}
	// get proc info itself
	buf := make([]byte, length)
	_, _, err = unix.Syscall6(
		unix.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(miblen),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&length)),
		0,
		0)
	if err != 0 {
		return buf, length, err
	}

	return buf, length, nil
}

var process_status = []string{"idle", "running", "sleeping", "stopped", "zombie"}

// ParseKinfoProc ParseKinfoProc
func ParseKinfoProc(buf []byte) ProcessInfo {
	proc := (*C.struct_kinfo_proc)(unsafe.Pointer(&buf[0]))

	p := ProcessInfo{Timestamp: int(time.Now().Unix())}

	p.Pid = int(C.get_pid(proc))
	p.PPid = int(C.get_ppid(proc))

	p.User, _ = getUserNameById(int(C.get_uid(proc)))
	procState := int(C.get_status(proc))
	if procState >= 0 && procState < len(process_status) {
		p.State = process_status[procState]
	}
	p.CreateTime = int(C.get_start_time(proc))

	return p
}

func ProcPidInfo(pid int, flavor int32, arg uint64) ([]byte, error) {
	length := uint64(C.get_proc_taskinfo_size())
	buf := make([]byte, int(length))
	ret := uint64(C.proc_pidinfo(C.int(pid), C.int(flavor), C.ulonglong(arg),
		unsafe.Pointer(&buf[0]), C.int(length)))
	if ret <= 0 {
		return nil, fmt.Errorf("proc_pidinfo failed ret:%d", ret)
	}
	return buf, nil
}

func ParsePidTaskInfo(buf []byte, p *ProcessInfo) {
	taskinfo := (*C.struct_proc_taskinfo)(unsafe.Pointer(&buf[0]))
	p.Cpu = float64(uint64(C.get_cpu_user(taskinfo))+uint64(C.get_cpu_sys(taskinfo))) / float64(1000000000.0)
	p.MemoryBytes = int64(C.get_resident_size(taskinfo))
}

func GetTotalMemorySize() (uint64, error) {
	mib := []int32{CtlHw, HwMemSize}
	memorysize := uint64(0)
	length := int32(unsafe.Sizeof(memorysize))
	buf := make([]byte, length)
	buf, _, err := callSyscallEx(mib, buf, length)
	if err == nil {
		memorysize = binary.LittleEndian.Uint64(buf)

	}

	return memorysize, err
}
