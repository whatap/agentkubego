//go:build linux || windows
// +build linux windows

package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/whatap/kube/node/src/whatap/lang/value"
)

const (
	DOCLINELIMIT = 50000
)

var (
	nvidiaexe = "nvidia-smi"
	proc      *os.Process
)

func collectNvidia(callback func(*value.MapValue, *value.MapValue)) {
	_, err := exec.LookPath(nvidiaexe)
	if err == nil {
		pollNvidiaPerf(nvidiaexe, callback)
	}
}

func pollNvidiaPerf(exe string, callback func(*value.MapValue, *value.MapValue)) {
	cmd := exec.Command(exe, "-x", "-q", "-a", "-l", "5")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	err = cmd.Start()
	if err != nil {
		return
	}
	proc = cmd.Process
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		cmd.Wait()
	}()

	var b bytes.Buffer

	var linecount int
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		b.WriteString(line)
		if "</nvidia_smi_log>" == line {
			parseDoc(b.Bytes(), callback)
			b.Reset()
		}
		linecount += 1
		if linecount > DOCLINELIMIT {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
			return
		}
	}
}

func parseDoc(bts []byte, callback func(*value.MapValue, *value.MapValue)) {
	res := new(NvidiaSmi)
	err := xml.Unmarshal(bts, res)
	if err != nil {
		return
	}

	for _, gpu := range res.GPUS {

		tags := value.NewMapValue()
		fields := value.NewMapValue()
		fields.PutRaw("Timestamp", res.Timestamp)

		tags.PutString("DriverVersion", res.DriverVersion)
		tags.PutString("AttachedGpus", res.AttachedGpus)

		fields.PutRaw("ID", gpu.ID)
		fields.PutRaw("MemClockClocksGpu", gpu.MemClockClocksGpu)
		fields.PutRaw("L1Cache", gpu.L1Cache)
		fields.PutRaw("ProductName", gpu.ProductName)
		fields.PutRaw("FreeFbMemoryUsageGpu", ParseBytes(gpu.FreeFbMemoryUsageGpu))
		fields.PutRaw("PowerState", gpu.PowerState)
		fields.PutRaw("Free", ParseBytes(gpu.Free))
		fields.PutRaw("RetiredCountDoubleBitRetirementRetiredPagesGpu", gpu.RetiredCountDoubleBitRetirementRetiredPagesGpu)
		fields.PutRaw("ClocksThrottleReasonUnknown", gpu.ClocksThrottleReasonUnknown)
		fields.PutRaw("ClocksThrottleReasonApplicationsClocksSetting", gpu.ClocksThrottleReasonApplicationsClocksSetting)
		for i, proc := range gpu.Processes {

			fields.PutRaw(fmt.Sprint("GpuInstanceId", i), proc.GpuInstanceId)
			fields.PutRaw(fmt.Sprint("ComputeInstanceId", i), proc.ComputeInstanceId)
			fields.PutRaw(fmt.Sprint("Pid", i), proc.Pid)
			fields.PutRaw(fmt.Sprint("ProcessType", i), proc.ProcessType)
			fields.PutRaw(fmt.Sprint("ProcessName", i), proc.ProcessName)
			fields.PutRaw(fmt.Sprint("UsedMemory", i), proc.UsedMemory)
		}

		fields.PutRaw("MemClockApplicationsClocksGpu", gpu.MemClockApplicationsClocksGpu)
		fields.PutRaw("L2CacheSingleBitAggregateEccErrorsGpu", gpu.L2CacheSingleBitAggregateEccErrorsGpu)
		fields.PutRaw("CurrentLinkGen", gpu.CurrentLinkGen)
		fields.PutRaw("TotalSingleBitVolatileEccErrorsGpu", gpu.TotalSingleBitVolatileEccErrorsGpu)
		fields.PutRaw("TextureMemoryDoubleBitVolatileEccErrorsGpu", gpu.TextureMemoryDoubleBitVolatileEccErrorsGpu)
		fields.PutRaw("L1CacheSingleBitAggregateEccErrorsGpu", gpu.L1CacheSingleBitAggregateEccErrorsGpu)
		fields.PutRaw("PendingGom", gpu.PendingGom)
		fields.PutRaw("AutoBoostDefault", gpu.AutoBoostDefault)
		fields.PutRaw("GraphicsClockApplicationsClocksGpu", gpu.GraphicsClockApplicationsClocksGpu)
		fields.PutRaw("PciBusID", gpu.PciBusID)
		fields.PutRaw("PowerManagement", ParseBytes(gpu.PowerManagement))
		fields.PutRaw("DeviceMemoryDoubleBitAggregateEccErrorsGpu", gpu.DeviceMemoryDoubleBitAggregateEccErrorsGpu)
		fields.PutRaw("BoardID", gpu.BoardID)
		fields.PutRaw("DeviceMemoryDoubleBitVolatileEccErrorsGpu", gpu.DeviceMemoryDoubleBitVolatileEccErrorsGpu)
		fields.PutRaw("SupportedGraphicsClock", gpu.SupportedGraphicsClock)
		fields.PutRaw("PersistenceMode", gpu.PersistenceMode)
		fields.PutRaw("MemClock", gpu.MemClock)
		fields.PutRaw("GraphicsClockClocksGpu", gpu.GraphicsClockClocksGpu)
		fields.PutRaw("Used", ParseBytes(gpu.Used))
		fields.PutRaw("ImgVersion", gpu.ImgVersion)
		fields.PutRaw("UsedFbMemoryUsageGpu", ParseBytes(gpu.UsedFbMemoryUsageGpu))
		fields.PutRaw("TotalDoubleBitAggregateEccErrorsGpu", gpu.TotalDoubleBitAggregateEccErrorsGpu)
		fields.PutRaw("MinorNumber", gpu.MinorNumber)
		fields.PutRaw("ProductBrand", gpu.ProductBrand)
		fields.PutRaw("GraphicsClockDefaultApplicationsClocksGpu", gpu.GraphicsClockDefaultApplicationsClocksGpu)
		fields.PutRaw("TotalFbMemoryUsageGpu", ParseBytes(gpu.TotalFbMemoryUsageGpu))
		fields.PutRaw("RegisterFileDoubleBitVolatileEccErrorsGpu", gpu.RegisterFileDoubleBitVolatileEccErrorsGpu)
		fields.PutRaw("MinPowerLimit", parsePower(gpu.MinPowerLimit))
		fields.PutRaw("TxUtil", parsePct(gpu.TxUtil))
		fields.PutRaw("TextureMemory", gpu.TextureMemory)
		fields.PutRaw("RegisterFileDoubleBitAggregateEccErrorsGpu", gpu.RegisterFileDoubleBitAggregateEccErrorsGpu)
		fields.PutRaw("PerformanceState", gpu.PerformanceState)
		fields.PutRaw("CurrentDm", gpu.CurrentDm)
		fields.PutRaw("PciDeviceID", gpu.PciDeviceID)
		fields.PutRaw("AccountedProcesses", gpu.AccountedProcesses)
		fields.PutRaw("PendingRetirement", gpu.PendingRetirement)
		fields.PutRaw("TotalDoubleBitVolatileEccErrorsGpu", gpu.TotalDoubleBitVolatileEccErrorsGpu)
		fields.PutRaw("UUID", gpu.UUID)
		fields.PutRaw("PowerLimit", parsePower(gpu.PowerLimit))
		fields.PutRaw("ClocksThrottleReasonHwSlowdown", gpu.ClocksThrottleReasonHwSlowdown)
		fields.PutRaw("BridgeChipFw", gpu.BridgeChipFw)
		fields.PutRaw("ReplayCounter", gpu.ReplayCounter)
		fields.PutRaw("L2CacheDoubleBitAggregateEccErrorsGpu", gpu.L2CacheDoubleBitAggregateEccErrorsGpu)
		fields.PutRaw("ComputeMode", gpu.ComputeMode)
		fields.PutRaw("FanSpeed", gpu.FanSpeed)
		fields.PutRaw("Total", ParseBytes(gpu.Total))
		fields.PutRaw("SmClock", gpu.SmClock)
		fields.PutRaw("RxUtil", parsePct(gpu.RxUtil))
		fields.PutRaw("GraphicsClock", gpu.GraphicsClock)
		fields.PutRaw("PwrObject", gpu.PwrObject)
		fields.PutRaw("PciBus", gpu.PciBus)
		fields.PutRaw("DecoderUtil", parsePct(gpu.DecoderUtil))
		fields.PutRaw("PciSubSystemID", gpu.PciSubSystemID)
		fields.PutRaw("MaxLinkGen", gpu.MaxLinkGen)
		fields.PutRaw("BridgeChipType", gpu.BridgeChipType)
		fields.PutRaw("SmClockClocksGpu", gpu.SmClockClocksGpu)
		fields.PutRaw("CurrentEcc", gpu.CurrentEcc)
		fields.PutRaw("PowerDraw", parsePower(gpu.PowerDraw))
		fields.PutRaw("CurrentLinkWidth", gpu.CurrentLinkWidth)
		fields.PutRaw("AutoBoost", gpu.AutoBoost)
		fields.PutRaw("GpuUtil", parsePct(gpu.GpuUtil))
		fields.PutRaw("PciDevice", gpu.PciDevice)
		fields.PutRaw("RegisterFile", gpu.RegisterFile)
		fields.PutRaw("L2Cache", gpu.L2Cache)
		fields.PutRaw("L1CacheDoubleBitAggregateEccErrorsGpu", gpu.L1CacheDoubleBitAggregateEccErrorsGpu)
		fields.PutRaw("RetiredCount", parseInt(gpu.RetiredCount))
		fields.PutRaw("PendingDm", gpu.PendingDm)
		fields.PutRaw("AccountingModeBufferSize", gpu.AccountingModeBufferSize)
		fields.PutRaw("GpuTempSlowThreshold", parseTemp(gpu.GpuTempSlowThreshold))
		fields.PutRaw("OemObject", gpu.OemObject)
		fields.PutRaw("TextureMemorySingleBitAggregateEccErrorsGpu", gpu.TextureMemorySingleBitAggregateEccErrorsGpu)
		fields.PutRaw("RegisterFileSingleBitAggregateEccErrorsGpu", gpu.RegisterFileSingleBitAggregateEccErrorsGpu)
		fields.PutRaw("MaxLinkWidth", gpu.MaxLinkWidth)
		fields.PutRaw("TextureMemoryDoubleBitAggregateEccErrorsGpu", gpu.TextureMemoryDoubleBitAggregateEccErrorsGpu)
		fields.PutRaw("ClocksThrottleReasonGpuIdle", gpu.ClocksThrottleReasonGpuIdle)
		fields.PutRaw("MultigpuBoard", gpu.MultigpuBoard)
		fields.PutRaw("GpuTempMaxThreshold", parseTemp(gpu.GpuTempMaxThreshold))
		fields.PutRaw("MaxPowerLimit", parsePower(gpu.MaxPowerLimit))
		fields.PutRaw("L2CacheDoubleBitVolatileEccErrorsGpu", gpu.L2CacheDoubleBitVolatileEccErrorsGpu)
		fields.PutRaw("PciDomain", gpu.PciDomain)
		fields.PutRaw("MemClockDefaultApplicationsClocksGpu", gpu.MemClockDefaultApplicationsClocksGpu)
		fields.PutRaw("VbiosVersion", gpu.VbiosVersion)
		fields.PutRaw("RetiredPageAddresses", gpu.RetiredPageAddresses)
		fields.PutRaw("GpuTemp", parseTemp(gpu.GpuTemp))
		fields.PutRaw("AccountingMode", gpu.AccountingMode)
		fields.PutRaw("L1CacheDoubleBitVolatileEccErrorsGpu", gpu.L1CacheDoubleBitVolatileEccErrorsGpu)
		fields.PutRaw("DeviceMemorySingleBitAggregateEccErrorsGpu", gpu.DeviceMemorySingleBitAggregateEccErrorsGpu)
		fields.PutRaw("DisplayActive", gpu.DisplayActive)
		fields.PutRaw("DefaultPowerLimit", parsePower(gpu.DefaultPowerLimit))
		fields.PutRaw("EncoderUtil", parsePct(gpu.EncoderUtil))
		fields.PutRaw("Serial", gpu.Serial)
		fields.PutRaw("EnforcedPowerLimit", parsePower(gpu.EnforcedPowerLimit))
		fields.PutRaw("RetiredPageAddressesDoubleBitRetirementRetiredPagesGpu", gpu.RetiredPageAddressesDoubleBitRetirementRetiredPagesGpu)
		fields.PutRaw("EccObject", gpu.EccObject)
		fields.PutRaw("Value", gpu.Value)
		fields.PutRaw("DisplayMode", gpu.DisplayMode)
		fields.PutRaw("DeviceMemory", gpu.DeviceMemory)
		fields.PutRaw("PendingEcc", gpu.PendingEcc)
		fields.PutRaw("ClocksThrottleReasonSwPowerCap", parsePower(gpu.ClocksThrottleReasonSwPowerCap))
		fields.PutRaw("TotalSingleBitAggregateEccErrorsGpu", ParseBytes(gpu.TotalSingleBitAggregateEccErrorsGpu))
		fields.PutRaw("CurrentGom", gpu.CurrentGom)
		fields.PutRaw("MemoryUtil", parsePct(gpu.MemoryUtil))

		callback(tags, fields)
	}
}

func toBytes(unit string) int64 {
	var ret int64
	switch unit {
	case "TB":
		ret = 1000000000000
	case "tB", "TiB":
		ret = 0x10000000000
	case "GB":
		ret = 1000000000
	case "gB", "GiB":
		ret = 0x40000000
	case "MB":
		ret = 1000000
	case "mB", "MiB":
		ret = 0x100000
	case "kB":
		ret = 1000
	case "KB", "KiB":
		ret = 0x400
	default:
		ret = 1
	}
	return ret
}

func ParseBytes(src string) int64 {
	words := strings.Fields(src)
	if len(words) == 2 {
		val, _ := strconv.ParseInt(words[0], 10, 64)
		val *= toBytes(words[1])

		return val
	} else if len(words) == 2 {
		val, _ := strconv.ParseInt(words[0], 10, 64)

		return val
	}

	return 0
}

func parsePct(src string) (ret float32) {
	ret = 0
	trimmed := strings.ReplaceAll(src, "%", "")
	trimmed = strings.TrimSpace(trimmed)

	v, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return
	}
	ret = float32(v)

	return
}

func parseInt(src string) (ret int32) {
	ret = 0
	trimmed := strings.TrimSpace(src)

	v, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return
	}
	ret = int32(v)

	return
}

func parseTemp(src string) (ret float32) {
	ret = 0
	trimmed := strings.ReplaceAll(src, "C", "")
	trimmed = strings.ReplaceAll(src, "F", "")
	trimmed = strings.TrimSpace(trimmed)

	v, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return
	}
	ret = float32(v)

	return
}

func parsePower(src string) (ret float32) {
	ret = 0
	trimmed := strings.ReplaceAll(src, "W", "")
	trimmed = strings.TrimSpace(trimmed)

	v, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return
	}
	ret = float32(v)

	return
}
