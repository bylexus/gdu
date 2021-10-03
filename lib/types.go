package gdu

import "io/fs"

type Filelike interface {
	GetPath() string
	GetByteSize() uint64
}

type File struct {
	RelPath   string
	SizeBytes uint64
	FileInfo  fs.FileInfo
}

func NewFile(path string, info fs.FileInfo) File {
	return File{
		RelPath:   path,
		SizeBytes: 0,
		FileInfo:  info,
	}
}

func (f File) GetPath() string {
	return f.RelPath
}

func (f File) GetByteSize() uint64 {
	return f.SizeBytes
}

type Dir struct {
	RelPath        string
	totalSizeBytes uint64
	Children       []Filelike
	FileInfo       fs.FileInfo
}

func NewDir(path string, info fs.FileInfo) Dir {
	return Dir{
		RelPath:        path,
		totalSizeBytes: 0,
		Children:       make([]Filelike, 0),
		FileInfo:       info,
	}
}

func (d Dir) GetPath() string {
	return d.RelPath
}

func (d Dir) GetByteSize() uint64 {
	var sum uint64 = 0
	for _, c := range d.Children {
		sum += c.GetByteSize()
	}
	return sum
}
