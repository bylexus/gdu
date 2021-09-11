package main

import (
	"flag"
	"fmt"
	"io/fs"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
)

type Flags struct {
	humanReadable bool
	printDetails  bool
}

var flags Flags = Flags{false, true}

type filelike interface {
	getPath() string
	getByteSize() uint64
}

type file struct {
	relPath   string
	sizeBytes uint64
}

func (f file) getPath() string {
	return f.relPath
}

func (f file) getByteSize() uint64 {
	return f.sizeBytes
}

type dir struct {
	relPath        string
	totalSizeBytes uint64
	children       []filelike
}

func (d dir) getPath() string {
	return d.relPath
}

func (d dir) getByteSize() uint64 {
	return d.totalSizeBytes
}

func examineDir(path string) (filelike, error) {
	var files []fs.FileInfo
	var err error

	d := dir{
		relPath:        path,
		totalSizeBytes: 0,
		children:       make([]filelike, 0),
	}
	files, err = ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		child, err := examinePath(filepath.Join(path, file.Name()))
		if err == nil && child != nil {
			d.totalSizeBytes += child.getByteSize()
			d.children = append(d.children, child)
		}
	}
	return d, nil
}

func examineFile(fileInfo fs.FileInfo, path string) (filelike, error) {
	return file{
		relPath:   path,
		sizeBytes: uint64(fileInfo.Size()),
	}, nil
}

func examinePath(path string) (filelike, error) {
	var ret filelike

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		ret, err = examineDir(path)
	} else if fileInfo.Mode().IsRegular() {
		ret, err = examineFile(fileInfo, path)
	}
	if err == nil && ret != nil && flags.printDetails == true {
		printEntry(ret)
	}

	return ret, err
}

func printEntry(entry filelike) {
	if flags.humanReadable {
		fmt.Printf("%s\t%s\n", toHumanReadableSize(entry.getByteSize()), entry.getPath())
	} else {
		fmt.Printf("%d\t%s\n", entry.getByteSize(), entry.getPath())
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
	flags.printDetails = *summary != true
	flags.humanReadable = *humanReadable

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
				if flags.printDetails == false {
					printEntry(ret)
				}
				size = ret.getByteSize()
			}
			res <- size
		}(path, ch)
	}
	for i := 0; i < len(searchPaths); i++ {
		total += <-ch
	}

	printEntry(file{"Total", total})
}
