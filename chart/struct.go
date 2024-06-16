package chart

import (
	"github.com/yarox24/EvtxHussar/common"
	"path/filepath"
	"strings"
	"sync"
)

const KEY_FORMAT_HOUR = "%d-%02d-%02d %02d"
const KEY_FORMAT_DAILY = "%d-%02d-%02d"

type SecondStruct struct {
	NanoSec map[int]int
}

type MinuteStruct struct {
	Seconds map[int]SecondStruct
}

type HourStruct struct {
	Minutes map[int]MinuteStruct
}

type DayStruct struct {
	HoursNanosec map[int]HourStruct // For nanosecond precision only
	HoursHours   map[int]int        // For hour precision only
}

type MonthStruct struct {
	Days map[int]DayStruct
}

type YearStruct struct {
	Months map[int]MonthStruct
}

type SingleEvtxChartData struct {
	Hostname           string
	WinningHostname    string
	NanoSecPrecision   bool
	Filename           string
	Path               string
	Channel            string
	record_counter     int64
	alternative_header common.EVTXHeaderAlternative
	EventsByYear       map[int]YearStruct
}

type ChartGlobalMemory struct {
	Wg_chart_all                 *sync.WaitGroup
	Atomic_Counter_Workers       uint64
	Atomic_MultipleEvtxChartData []SingleEvtxChartData
	Mutex_MultipleEvtxChartData  sync.Mutex
}

func NewChartGlobalMemory() ChartGlobalMemory {
	return ChartGlobalMemory{
		Wg_chart_all:                 new(sync.WaitGroup),
		Atomic_Counter_Workers:       0,
		Atomic_MultipleEvtxChartData: make([]SingleEvtxChartData, 0, 300),
		Mutex_MultipleEvtxChartData:  sync.Mutex{},
	}
}

type ChannelColor struct {
	Channel string
	Color   string
}

// Functions for SingleEvtxChartData

func (secd *SingleEvtxChartData) GetPath() string {
	return secd.Path
}

func (secd *SingleEvtxChartData) GetFilenameWithoutExtension() string {
	return strings.TrimSuffix(filepath.Base(secd.Path), filepath.Ext(secd.Path))
}

func (secd *SingleEvtxChartData) GetChannel() string {
	return secd.Channel
}

func (secd *SingleEvtxChartData) GetWinningHostname() string {
	return secd.Hostname
}

func (secd *SingleEvtxChartData) IsFullAndWithProperChannel() bool {
	return secd.Hostname != "" && secd.Channel != ""
}

func (secd *SingleEvtxChartData) GetAlternativeHeader() *common.EVTXHeaderAlternative {
	return &secd.alternative_header
}

func (secd *SingleEvtxChartData) GetNumberOfRecords() int64 {
	return secd.record_counter
}

func (secd *SingleEvtxChartData) SetNumberOfRecords(record_counter int64) {
	secd.record_counter = record_counter
}
