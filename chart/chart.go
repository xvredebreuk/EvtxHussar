package chart

import (
	"fmt"
	"github.com/Velocidex/ordereddict"
	"github.com/yarox24/EvtxHussar/common"
	"github.com/yarox24/EvtxHussar/eventmap"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	"www.velocidex.com/golang/evtx"
)

func returnOnlyHostRelatedData(chaglobmem *ChartGlobalMemory, hostname string) []SingleEvtxChartData {
	var filteredData []SingleEvtxChartData

	for _, data := range chaglobmem.Atomic_MultipleEvtxChartData {
		if data.WinningHostname != "" {
			if strings.ToLower(data.WinningHostname) == strings.ToLower(hostname) {
				filteredData = append(filteredData, data)
			}
		}
	}

	return filteredData
}

func nonEmptyHostRelatedData(hostRelatedData []SingleEvtxChartData) bool {
	for _, data := range hostRelatedData {
		if len(data.EventsByYear) > 0 {
			return true
		}
	}

	return false
}

func determineUniqueDirectories(chaglobmem *ChartGlobalMemory) []string {
	unique_directories := make(map[string]bool)

	for _, data := range chaglobmem.Atomic_MultipleEvtxChartData {
		directory := filepath.Dir(data.Path)
		unique_directories[directory] = true
	}

	// Convert the map keys (which are the unique paths) to a slice
	var uniquePaths []string
	for path := range unique_directories {
		uniquePaths = append(uniquePaths, path)
	}

	return uniquePaths
}

func determineSubsetofDirectory(chaglobmem *ChartGlobalMemory, directory string) []SingleEvtxChartData {
	var subset []SingleEvtxChartData
	for _, data := range chaglobmem.Atomic_MultipleEvtxChartData {
		if filepath.Dir(data.Path) == directory {
			subset = append(subset, data)
		}
	}

	return subset
}

func determineAllHostnames(EvtxDirectorySubset []SingleEvtxChartData) []string {
	hostnames_map := make(map[string]bool)

	for _, data := range EvtxDirectorySubset {
		if len(data.Hostname) > 0 {
			hostnames_map[data.Hostname] = true
		}
	}

	var uniqueHostnames []string
	for path := range hostnames_map {
		uniqueHostnames = append(uniqueHostnames, path)
	}

	return uniqueHostnames
}

func determineWinningHostname(EvtxDirectorySubset []SingleEvtxChartData) string {
	var TRUSTED_CHANNELS_IN_ORDER = []string{"Security", "Application", "System", "Microsoft-Windows-PowerShell/Operational", "Microsoft-Windows-TaskScheduler/Operational"}

	// Best hostname from common logs
	for _, ordered_channel := range TRUSTED_CHANNELS_IN_ORDER {
		for _, data := range EvtxDirectorySubset {
			if strings.ToLower(data.Channel) == strings.ToLower(ordered_channel) {
				return data.Hostname
			}
		}
	}

	// Last resort from any non-empty logs
	for _, data := range EvtxDirectorySubset {
		if data.Hostname != "" {
			return data.Hostname
		}
	}

	return "Empty"
}

func setWinningHostname(chaglobmem *ChartGlobalMemory, EvtxDirectorySubset []SingleEvtxChartData, winningHostname string) {
	for _, data := range EvtxDirectorySubset {
		for i, _ := range chaglobmem.Atomic_MultipleEvtxChartData {

			if chaglobmem.Atomic_MultipleEvtxChartData[i].Hostname == "" {
				continue
			}

			if chaglobmem.Atomic_MultipleEvtxChartData[i].Path == data.Path {
				chaglobmem.Atomic_MultipleEvtxChartData[i].WinningHostname = winningHostname
				break
			}
		}

	}
}

func setEmptyHosts(efi_list []common.EvtxFileInfo, c *ChartGlobalMemory) {

	for _, efi := range efi_list {
		var secd = SingleEvtxChartData{
			Hostname:           efi.GetLatestComputer(),
			WinningHostname:    "",
			NanoSecPrecision:   false,
			Filename:           efi.GetFilenameWithoutExtension(),
			Path:               efi.GetPath(),
			Channel:            efi.GetChannel(),
			alternative_header: *efi.GetAlternativeHeader(),
			EventsByYear:       make(map[int]YearStruct), // Initialize the map
		}
		c.Atomic_MultipleEvtxChartData = append(c.Atomic_MultipleEvtxChartData, secd)
	}

}

func determineChartHosts(chaglobmem *ChartGlobalMemory) map[string][]string {
	hostMap := make(map[string]map[string]bool)

	// Determine hostname based on few files in the same directory
	// Assume all files in the same directory got the determine hostname (regardless if this is true, this is input requirement)
	uniqueDirectories := determineUniqueDirectories(chaglobmem)

	for _, directory := range uniqueDirectories {
		EvtxDirectorySubset := determineSubsetofDirectory(chaglobmem, directory)

		allHostnames := determineAllHostnames(EvtxDirectorySubset)
		winningHostname := determineWinningHostname(EvtxDirectorySubset)

		// Add winner to the host map
		if _, exists := hostMap[winningHostname]; !exists {
			hostMap[winningHostname] = make(map[string]bool)
		}

		// Add alternative names to the host map (Excluding winner)
		for _, host := range allHostnames {
			if host != winningHostname {
				hostMap[winningHostname][host] = true
			}
		}

		// Set winner as WinningHostname for current directory
		setWinningHostname(chaglobmem, EvtxDirectorySubset, winningHostname)
	}
	// Convert the map of maps to a map of slices containing only unique values
	uniqueHostMap := make(map[string][]string)

	for primaryHost, hostSet := range hostMap {
		uniqueHosts := make([]string, 0, len(hostSet))
		for host := range hostSet {
			uniqueHosts = append(uniqueHosts, host)
		}
		uniqueHostMap[primaryHost] = uniqueHosts
	}

	return uniqueHostMap
}

func GenerateFrequencyChart(chart_type string, output_dir string, templates_path string, efi []common.EvtxFileInfo, WorkersLimit int) {

	// None - disable chart generation
	if chart_type == "none" {
		common.LogInfo("Chart won't be generated as you disabled generation.")
		return
	}

	// Generate chart
	var chartglobmem = NewChartGlobalMemory()

	common.LogInfo(fmt.Sprintf("Chart will be using format %s", chart_type))
	common.LogInfo("Start chart generation")

	// Set empty hosts
	setEmptyHosts(efi, &chartglobmem)

	// From this moment statistics are split by winning hostnames
	uniqueHostMap := determineChartHosts(&chartglobmem)

	// Nanosec precision only for duplicates
	for winningHostname, _ := range uniqueHostMap {
		chartRawData := returnOnlyHostRelatedData(&chartglobmem, winningHostname)
		all_channels := listUniqueChannels(chartRawData)

		for _, channel := range all_channels {
			SubsetEvtxForChannel := getEvtxChartDataForChannel(channel, chartRawData)

			if len(SubsetEvtxForChannel) > 1 {
				// Multiple files for same host and same channel. Increase precision
				for _, evtxCopy := range SubsetEvtxForChannel {
					for i, atomic_evtx := range chartglobmem.Atomic_MultipleEvtxChartData {
						if atomic_evtx.Path == evtxCopy.Path {
							chartglobmem.Atomic_MultipleEvtxChartData[i].NanoSecPrecision = true
							break
						}
					}
				}

			}

		}
	}

	chartglobmem.startWorkers(WorkersLimit)

	// Wait until jobs are done
	chartglobmem.Wg_chart_all.Wait()

	//For every hostname generate separate statistics
	for winningHostname, alternative_hostnames := range uniqueHostMap {
		chartRawData := returnOnlyHostRelatedData(&chartglobmem, winningHostname)

		// Check if there is something to generate for this hostname
		if !nonEmptyHostRelatedData(chartRawData) {
			common.LogError(fmt.Sprintf("Empty data for chart generation. Some charts won't be generated"))
			continue
		}

		switch chart_type {
		//case "json":
		//	generateChartJSON(output_dir, hostname)
		case "html":
			generateChartHTML(output_dir, winningHostname, alternative_hostnames, templates_path, chartRawData)
		}
	}

	common.LogDebugMap(uniqueHostMap, "Unique hosts and their mapping for chart")

	common.LogInfo("End chart generation")
	return
}

func concurrencyUnlockNewWorkers(ch_limit_worker chan struct{}, nr_of_files int) {

	// Keep new workers at limit
	for i := 0; i < nr_of_files; i++ {
		ch_limit_worker <- struct{}{}
	}

	close(ch_limit_worker)
}

func printRemainsWorkers(Atomic_Counter_Workers *uint64) {

	for {
		// Every report after one minute
		time.Sleep(time.Minute * time.Duration(1))
		remaining_workers := atomic.LoadUint64(Atomic_Counter_Workers)

		if remaining_workers == 0 {
			common.LogWarn("All .evtx workers finished. Going to next phase ...")
			break
		} else {
			common.LogWarn(fmt.Sprintf("%d .evtx chart workers are still running ...", remaining_workers))
		}

	}
}

func (chaglobmem *ChartGlobalMemory) startWorkers(WorkersLimit int) {

	// Expect
	AllEvtxNumber := len(chaglobmem.Atomic_MultipleEvtxChartData)

	chaglobmem.Wg_chart_all.Add(AllEvtxNumber)
	atomic.AddUint64(&chaglobmem.Atomic_Counter_Workers, uint64(AllEvtxNumber))

	// Limit concurrent workers
	ch_limit_worker := make(chan struct{}, WorkersLimit)

	go concurrencyUnlockNewWorkers(ch_limit_worker, AllEvtxNumber)
	go printRemainsWorkers(&chaglobmem.Atomic_Counter_Workers)

	// For every file
	for i := 0; i < AllEvtxNumber; i++ {
		go runWorker(chaglobmem, &chaglobmem.Atomic_MultipleEvtxChartData[i], ch_limit_worker)
	}
}

func runWorker(chaglobmem *ChartGlobalMemory, secd *SingleEvtxChartData, ch_limit_worker chan struct{}) {

	// Run only when not exceeding limit
	<-ch_limit_worker

	if secd.GetWinningHostname() == "" && secd.GetChannel() == "" {
		common.LogInfo(fmt.Sprintf("Run Chart Worker: %s  (Empty)", secd.GetFilenameWithoutExtension()))
	} else {
		common.LogInfo(fmt.Sprintf("Run Chart Worker: %s | %s", secd.GetWinningHostname(), secd.GetChannel()))
	}

	defer chaglobmem.Wg_chart_all.Done()
	defer atomic.AddUint64(&chaglobmem.Atomic_Counter_Workers, ^uint64(0))

	var record_counter int64 = 0

	// Skip if empty or not valid
	if secd.IsFullAndWithProperChannel() {
		// Open evtx file
		fd, err := os.OpenFile(secd.GetPath(), os.O_RDONLY, os.FileMode(0666))

		if err == nil {
			defer fd.Close()
		} else {
			common.LogCriticalErrorWithError("Error occured when opening evtx: "+secd.GetPath(), err)
			return
		}

		chunks, err_chunks := evtx.GetChunks(fd)

		// Flags is dirty
		const IS_DIRTY = 0x1

		if secd.GetAlternativeHeader().FileFlags == IS_DIRTY {
			common.LogDebug(fmt.Sprintf("Dirty file detected: %s", secd.GetPath()))
			// => Parsing all found chunks
		} else {
			// => Cut off chunks over header number
			header_chunks_counts := int(secd.GetAlternativeHeader().ChunkCount)
			found_chunks_count := len(chunks)

			if header_chunks_counts < found_chunks_count {
				chunks = chunks[0:header_chunks_counts]
			}
		}

		if err_chunks != nil {
			common.LogErrorWithError("Evtx chunks error: "+secd.GetPath(), err_chunks)
			return
		}

		for _, chunk := range chunks {
			records, err_chunk := chunk.Parse(0)

			if err_chunk != nil {
				common.LogError("Chunk parsing error: " + secd.GetPath())
				continue
			}

			for _, i := range records {
				ev, ok := i.Event.(*ordereddict.Dict)

				if ok {
					ev_map, ok_map := ordereddict.GetMap(ev, "Event")

					// Now count
					record_counter += 1

					if !ok_map {
						common.LogError("Event parsing error: " + secd.GetPath())
						continue
					}
					rawsystemtime := eventmap.GetRawSystemTime(ev_map)
					secd.incrementEventsCounter(rawsystemtime)
				}
			}
		}

		secd.SetNumberOfRecords(record_counter)
	}

	// Append results to ChartGlobalMemory in atomic way
	chaglobmem.Mutex_MultipleEvtxChartData.Lock()
	defer chaglobmem.Mutex_MultipleEvtxChartData.Unlock()

	common.LogDebug(fmt.Sprintf("Finished Chart worker: %s | %s | Records: %d", secd.GetFilenameWithoutExtension(), secd.GetChannel(), record_counter))
}

func (data *SingleEvtxChartData) incrementEventsCounter(t time.Time) {
	year, month, day := t.Date()
	hour := t.Hour()
	minute := t.Minute()
	seconds := t.Second()
	monthInt := int(month)
	nanosec := t.Nanosecond()

	// Year
	if _, ok := data.EventsByYear[year]; !ok {
		data.EventsByYear[year] = YearStruct{Months: make(map[int]MonthStruct)}
	}

	// Month
	if _, ok := data.EventsByYear[year].Months[monthInt]; !ok {
		data.EventsByYear[year].Months[monthInt] = MonthStruct{Days: make(map[int]DayStruct)}
	}

	// Days [with split]
	if _, ok := data.EventsByYear[year].Months[monthInt].Days[day]; !ok {

		// NanoSec needed due to duplicates
		if data.NanoSecPrecision {
			data.EventsByYear[year].Months[monthInt].Days[day] = DayStruct{
				HoursNanosec: make(map[int]HourStruct),
				HoursHours:   nil,
			}
		} else {
			data.EventsByYear[year].Months[monthInt].Days[day] = DayStruct{
				HoursNanosec: nil,
				HoursHours:   make(map[int]int),
			}
		}

	}

	// Hours + Minutes + Second + Nano Sec
	if data.NanoSecPrecision {
		if _, ok := data.EventsByYear[year].Months[monthInt].Days[day].HoursNanosec[hour]; !ok {
			data.EventsByYear[year].Months[monthInt].Days[day].HoursNanosec[hour] = HourStruct{
				Minutes: make(map[int]MinuteStruct),
			}
		}

		if _, ok := data.EventsByYear[year].Months[monthInt].Days[day].HoursNanosec[hour].Minutes[minute]; !ok {
			data.EventsByYear[year].Months[monthInt].Days[day].HoursNanosec[hour].Minutes[minute] = MinuteStruct{
				Seconds: make(map[int]SecondStruct),
			}
		}

		if _, ok := data.EventsByYear[year].Months[monthInt].Days[day].HoursNanosec[hour].Minutes[minute].Seconds[seconds]; !ok {
			data.EventsByYear[year].Months[monthInt].Days[day].HoursNanosec[hour].Minutes[minute].Seconds[seconds] = SecondStruct{
				NanoSec: make(map[int]int),
			}
		}

		finalStruct := data.EventsByYear[year].Months[monthInt].Days[day].HoursNanosec[hour].Minutes[minute].Seconds[seconds]
		if _, ok := finalStruct.NanoSec[nanosec]; ok {
			finalStruct.NanoSec[nanosec] += 1
		} else {
			finalStruct.NanoSec[nanosec] = 1
		}

	} else {
		// Just hours
		finalStruct := data.EventsByYear[year].Months[monthInt].Days[day]
		if _, ok := finalStruct.HoursHours[hour]; ok {
			finalStruct.HoursHours[hour] += 1
		} else {
			finalStruct.HoursHours[hour] = 1
		}
	}

}
