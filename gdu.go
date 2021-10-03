package main

import (
	"flag"
	"fmt"
	"math"
	"runtime"
	"sync"

	gdu "alexi.ch/gdu/lib"
)

func printEntry(entry gdu.Filelike, flags gdu.Flags) {
	if flags.PrintDetails == gdu.OUTPUT_FULL {
		switch entry.(type) {
		case *gdu.Dir:
			for _, child := range entry.(*gdu.Dir).Children {
				printEntry(child, flags)
			}
		}
	}
	if flags.HumanReadable {
		fmt.Printf("%s\t%s\n", toHumanReadableSize(entry.GetByteSize()), entry.GetPath())
	} else {
		fmt.Printf("%d\t%s\n", entry.GetByteSize(), entry.GetPath())
	}
}

var kbyteBase float64 = 1000.0
var logBase float64 = math.Log10(kbyteBase)

func toHumanReadableSize(byteSize uint64) string {
	var res string
	if byteSize == 0 {
		return "0"
	}

	thousands := math.Floor(math.Log10(float64(byteSize)) / logBase)
	displaySize := float64(byteSize) / (math.Pow(kbyteBase, thousands))
	switch {
	case thousands < 1.0:
		res = fmt.Sprintf("%.f", displaySize)
	case thousands < 2.0:
		res = fmt.Sprintf("%.3f kB", displaySize)
	case thousands < 3.0:
		res = fmt.Sprintf("%.3f MB", displaySize)
	case thousands < 4.0:
		res = fmt.Sprintf("%.3f GB", displaySize)
	case thousands < 5.0:
		res = fmt.Sprintf("%.3f TB", displaySize)
	case thousands < 6.0:
		res = fmt.Sprintf("%.3f PB", displaySize)
	default:
		res = fmt.Sprintf("%.3f", displaySize)
	}

	return res
}

func main() {
	flags := gdu.Flags{HumanReadable: false, PrintDetails: gdu.OUTPUT_FULL, NrOfWorkers: 1}

	workers := gdu.MaxInt(runtime.NumCPU(), 1)
	summary := flag.Bool("s", false, "print only summary per given file")
	humanReadable := flag.Bool("h", false, "Print human readable sizes")

	flag.Parse()
	searchPaths := flag.Args()
	if *summary == true {
		flags.PrintDetails = gdu.OUTPUT_SUMMARY
	}
	flags.HumanReadable = *humanReadable
	flags.NrOfWorkers = workers

	topLevelFiles := make([]gdu.Filelike, 0)
	queue := gdu.JobQueue{
		WaitGroup: new(sync.WaitGroup),
		JobQueue:  make(chan gdu.Filelike),
	}

	// start workers:
	for i := 0; i < flags.NrOfWorkers; i++ {
		w := gdu.NewWorker(&queue)
		go w.ProcessJobs()
	}

	// create top-level work items:
	for _, path := range searchPaths {
		filelike, err := gdu.CreateFileLike(path)
		if err == nil {
			topLevelFiles = append(topLevelFiles, filelike)
			queue.EnqueueJob(filelike)
		}
	}
	// wait for all workers to signal they're done:
	queue.Join()

	var total uint64 = 0
	for _, f := range topLevelFiles {
		total += f.GetByteSize()
		printEntry(f, flags)
	}
	if len(topLevelFiles) > 1 {
		printEntry(&gdu.File{SizeBytes: total, RelPath: "Total"}, flags)
	}
}
