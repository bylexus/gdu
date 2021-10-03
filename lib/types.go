package gdu

type Flags struct {
	HumanReadable bool
	PrintDetails  bool
}

type Filelike interface {
	GetPath() string
	GetByteSize() uint64
}

type File struct {
	RelPath   string
	SizeBytes uint64
}

func (f File) GetPath() string {
	return f.RelPath
}

func (f File) GetByteSize() uint64 {
	return f.SizeBytes
}

type Dir struct {
	RelPath        string
	TotalSizeBytes uint64
	Children       []Filelike
}

func (d Dir) GetPath() string {
	return d.RelPath
}

func (d Dir) GetByteSize() uint64 {
	return d.TotalSizeBytes
}
