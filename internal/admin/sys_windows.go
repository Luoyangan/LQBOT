//go:build windows
// +build windows

package admin

import (
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetDiskFreeSpaceExW = kernel32.NewProc("GetDiskFreeSpaceExW")
	procGetProcessTimes      = kernel32.NewProc("GetProcessTimes")
	procGetSystemTimes       = kernel32.NewProc("GetSystemTimes")
)

type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

type filetime struct {
	dwLowDateTime  uint32
	dwHighDateTime uint32
}

func (ft filetime) nanosec() int64 {
	return (int64(ft.dwHighDateTime)<<32 + int64(ft.dwLowDateTime)) * 100
}

// cpuCache stores the previous system CPU reading for delta calculation.
var (
	cpuCacheMu  sync.Mutex
	cpuCache    struct {
		idle  int64
		total int64
		time  time.Time
	}
	firstCPUCache sync.Once
)

// getSystemCPUUsage returns the recent system-wide CPU usage (%) by measuring
// the delta between consecutive calls (similar to Task Manager).
func getSystemCPUUsage() float64 {
	var idleTime, kernelTime, userTime filetime
	ret, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idleTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)
	if ret == 0 {
		return 0
	}

	now := time.Now()
	idle := idleTime.nanosec()
	total := kernelTime.nanosec() + userTime.nanosec()

	cpuCacheMu.Lock()
	defer cpuCacheMu.Unlock()

	// First call: just cache and return average since boot
	firstCPUCache.Do(func() {
		cpuCache.idle = idle
		cpuCache.total = total
		cpuCache.time = now
	})

	// Compute delta-based usage
	dIdle := idle - cpuCache.idle
	dTotal := total - cpuCache.total
	elapsed := now.Sub(cpuCache.time).Nanoseconds()

	// Update cache
	cpuCache.idle = idle
	cpuCache.total = total
	cpuCache.time = now

	// Fallback to average since boot if delta is too small (< 500ms)
	if elapsed < 500_000_000 || dTotal <= 0 {
		if total <= 0 {
			return 0
		}
		return float64(total-idle) * 100 / float64(total)
	}

	pct := float64(dTotal-dIdle) * 100 / float64(dTotal)
	if pct > 100 {
		pct = 100
	}
	if pct < 0 {
		pct = 0
	}
	return pct
}

// getProcessCPUUsage returns the average CPU usage (%) since process start.
func getProcessCPUUsage() float64 {
	h, _ := syscall.GetCurrentProcess()
	var creationTime, exitTime, kernelTime, userTime filetime
	ret, _, _ := procGetProcessTimes.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(&creationTime)),
		uintptr(unsafe.Pointer(&exitTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)
	if ret == 0 {
		return 0
	}
	cpuNanos := kernelTime.nanosec() + userTime.nanosec()
	wallNanos := exitTime.nanosec() - creationTime.nanosec()
	if wallNanos <= 0 {
		// Process still running; use time.Now
		wallNanos = time.Now().UnixNano() - creationTime.nanosec()
	}
	if wallNanos <= 0 {
		return 0
	}
	pct := float64(cpuNanos) * 100 / float64(wallNanos)
	if pct > 100 {
		pct = 100
	}
	return pct
}

// getSystemMemoryGB returns (usedGB, totalGB) of physical RAM.
func getSystemMemoryGB() (usedGB, totalGB float64) {
	var ms memoryStatusEx
	ms.dwLength = uint32(unsafe.Sizeof(ms))
	ret, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms)))
	if ret == 0 {
		return 0, 0
	}
	totalGB = float64(ms.ullTotalPhys) / 1024 / 1024 / 1024
	usedGB = totalGB - float64(ms.ullAvailPhys)/1024/1024/1024
	return usedGB, totalGB
}

// getDiskFreeGB returns (freeGB, totalGB) for the given path's mount point.
func getDiskFreeGB(path string) (freeGB, totalGB float64) {
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0
	}
	var freeBytes, totalBytes, totalFree int64
	ret, _, _ := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(unsafe.Pointer(&freeBytes)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFree)),
	)
	if ret == 0 {
		return 0, 0
	}
	freeGB = float64(freeBytes) / 1024 / 1024 / 1024
	totalGB = float64(totalBytes) / 1024 / 1024 / 1024
	return freeGB, totalGB
}
