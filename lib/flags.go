package gdu

type SummaryType int

type Flags struct {
	HumanReadable bool
	PrintDetails  SummaryType
	NrOfWorkers   int
}

const (
	OUTPUT_SUMMARY SummaryType = iota
	OUTPUT_FULL    SummaryType = iota
)
