package main

import (
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"

	gdu "alexi.ch/gdu/lib"
)

var flags gdu.Flags = gdu.Flags{HumanReadable: false, PrintDetails: true}

func examineDir(path string) (gdu.Filelike, error) {
	var files []fs.FileInfo
	var err error

	d := gdu.Dir{
		RelPath:        path,
		TotalSizeBytes: 0,
		Children:       make([]gdu.Filelike, 0),
	}
	files, err = ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		child, err := examinePath(filepath.Join(path, file.Name()))
		if err == nil && child != nil {
			d.TotalSizeBytes += child.GetByteSize()
			d.Children = append(d.Children, child)
		}
	}
	return d, nil
}

func examineFile(fileInfo fs.FileInfo, path string) (gdu.Filelike, error) {
	return gdu.File{
		RelPath:   path,
		SizeBytes: uint64(fileInfo.Size()),
	}, nil
}

func examinePath(path string) (gdu.Filelike, error) {
	var ret gdu.Filelike

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		ret, err = examineDir(path)
	} else if fileInfo.Mode().IsRegular() {
		ret, err = examineFile(fileInfo, path)
	}
	if err == nil && ret != nil && flags.PrintDetails == true {
		printEntry(ret)
	}

	return ret, err
}

func printEntry(entry gdu.Filelike) {
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
	summary := flag.Bool("s", false, "print only summary per given file")
	humanReadable := flag.Bool("h", false, "Print human readable sizes")

	flag.Parse()
	searchPaths := flag.Args()
	flags.PrintDetails = *summary != true
	flags.HumanReadable = *humanReadable

	total := uint64(0)
	ch := make(chan uint64, len(searchPaths))

	for _, path := range searchPaths {
		// process each given path in a separate go routine in parallel:
		go func(p string, res chan<- uint64) {
			var size uint64 = 0
			ret, err := examinePath(p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			} else if ret != nil {
				// print dir summary only if printDetails is false, otherwise it will already be printed above
				if flags.PrintDetails == false {
					printEntry(ret)
				}
				size = ret.GetByteSize()
			}
			res <- size
		}(path, ch)
	}
	for i := 0; i < len(searchPaths); i++ {
		total += <-ch
	}

	printEntry(gdu.File{RelPath: "Total", SizeBytes: total})
}
