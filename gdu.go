package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	gdu "alexi.ch/gdu/lib"
)

func examineDir(jobQueue chan gdu.Filelike, wg *sync.WaitGroup, dir *gdu.Dir) error {
	files, err := ioutil.ReadDir(dir.RelPath)
	if err != nil {
		return err
	}
	for _, file := range files {
		filelike, err := createFileLike(filepath.Join(dir.RelPath, file.Name()))
		if err == nil && filelike != nil {
			dir.Children = append(dir.Children, filelike)
			// enqueue for later, parallel examination:
			enqueueJob(jobQueue, wg, filelike)
		}
	}
	return nil
}

func examineFile(file *gdu.File) {
	file.SizeBytes = uint64(file.FileInfo.Size())
}

func createFileLike(path string) (gdu.Filelike, error) {
	var ret gdu.Filelike

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		dir := gdu.NewDir(path, fileInfo)
		ret = &dir
	} else if fileInfo.Mode().IsRegular() {
		file := gdu.NewFile(path, fileInfo)
		ret = &file
	}

	return ret, err
}

func examineFilelike(jobQueue chan gdu.Filelike, wg *sync.WaitGroup, file gdu.Filelike) {
	switch file.(type) {
	case *gdu.Dir:
		examineDir(jobQueue, wg, file.(*gdu.Dir))
	case *gdu.File:
		examineFile(file.(*gdu.File))
	}
}

func enqueueJob(jobQueue chan gdu.Filelike, wg *sync.WaitGroup, job gdu.Filelike) {
	wg.Add(1)
	select {
	case jobQueue <- job: // ok, someone else took it
	default:
		// do it myself, no one else has time:
		examineFilelike(jobQueue, wg, job)
		wg.Done()
	}
}

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
	jobs := make(chan gdu.Filelike)
	wg := new(sync.WaitGroup)

	for i := 0; i < flags.NrOfWorkers; i++ {
		go func() {
			for job := range jobs {
				// do job
				examineFilelike(jobs, wg, job)
				wg.Done()
			}
		}()
	}

	for _, path := range searchPaths {
		filelike, err := createFileLike(path)
		if err == nil {
			topLevelFiles = append(topLevelFiles, filelike)
			enqueueJob(jobs, wg, filelike)
		}
	}
	wg.Wait()
	close(jobs)

	var total uint64 = 0
	for _, f := range topLevelFiles {
		total += f.GetByteSize()
		printEntry(f, flags)
		// fmt.Printf("%v %v, nr of childs: \n", toHumanReadableSize(f.GetByteSize()), f.GetPath())
	}
	if len(topLevelFiles) > 1 {
		printEntry(&gdu.File{SizeBytes: total, RelPath: "Total"}, flags)
	}
}
