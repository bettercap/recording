package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/bettercap/recording"
)

var (
	fileName   = "/home/evilsocket/bettercap-session.record"
	cpuProfile = ""
	memProfile = ""
)

func init() {
	flag.StringVar(&fileName, "file", fileName, "Record file name to load.")
	flag.StringVar(&cpuProfile, "cpu-profile", cpuProfile, "If filled, it'll save a CPU profile on this file.")
	flag.StringVar(&memProfile, "mem-profile", memProfile, "If filled, it'll save a memory profile on this file.")
}

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func main() {
	flag.Parse()

	if cpuProfile != "" {
		fmt.Printf("saving cpu profile to %s\n", cpuProfile)
		f, err := os.Create(cpuProfile)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// load a recording from a file and print progress while loading it
	arch, err := recording.Load(fileName, func(perc float64, done int, total int) {
		fmt.Printf("loaded %d/%d frames ( %.2f ) ...\n", done, total, perc)
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("loaded session started at %s and stopped at %s (%s):\n\n", arch.Session.StartedAt(), arch.Session.StoppedAt(), arch.Session.Duration())

	// keep iterating every session frame until they're over
	// the same is doable with the `arch.Events` field
	for arch.Session.Over() == false {
		raw := arch.Session.Next()
		// parse the raw JSON into a map or whatever to access its attributes
		fmt.Printf("loaded %d bytes of frame\n", len(raw))
	}

	PrintMemUsage()
	runtime.GC()
	PrintMemUsage()

	if memProfile != "" {
		f, err := os.Create(memProfile)
		if err != nil {
			panic(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
	}
}
