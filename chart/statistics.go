package chart

import (
	"fmt"
	"github.com/yarox24/EvtxHussar/common"
	"golang.org/x/exp/slices"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"
)

func GetOutputPath(Output_dir string, latest_computer string, extension string) string {
	return Output_dir + string(os.PathSeparator) + latest_computer + string(os.PathSeparator) + "chart" + string(os.PathSeparator) + "frequency_distribution_" + latest_computer + "." + extension
}

// Apache Echarts
// [949363200000, 17.9353930182935]
// https://currentmillis.com
// Example (UTC): 946684800000 => Sat Jan 01 2000 00:00:00
// [Array(2), Array(2), Array(2)]   Array(2) = [Date, Counter]
// new Date(Date.UTC(2024, 3, 10, 18, 40, 0)) => 1712774400000 => Wed Apr 10 2024 18:40:00

type Record struct {
	Year     int
	Month    int
	Day      int
	Hour     int
	Channels map[string]int
}

func rgba_color(red, green, blue int) string {
	const alpha = 0.4

	return fmt.Sprintf("rgba(%d, %d, %d, %.1f)", red, green, blue, alpha)
}

func getEvtxChartDataForChannel(channel string, data []SingleEvtxChartData) []SingleEvtxChartData {
	var filteredData []SingleEvtxChartData

	for _, data := range data {
		if strings.ToLower(data.Channel) == strings.ToLower(channel) {
			filteredData = append(filteredData, data)
		}
	}

	return filteredData
}

func listUniqueChannels(data []SingleEvtxChartData) []string {
	seenChannels := make(map[string]bool)

	for _, evtx := range data {
		if evtx.Channel != "" {
			if _, ok := seenChannels[evtx.Channel]; !ok {
				seenChannels[evtx.Channel] = true
			}
		}
	}
	var channels []string
	for channel := range seenChannels {
		channels = append(channels, channel)
	}
	sort.Strings(channels)

	return channels
}

func generateDeduplicatedMap(chartRawData []SingleEvtxChartData, hourPrecision bool) map[string]map[string]int {
	dateChannelCounterMap := make(map[string]map[string]int)

	// Key format (Hour vs daily precision of the output)
	var key_format = KEY_FORMAT_DAILY

	if hourPrecision {
		key_format = KEY_FORMAT_HOUR
	}

	all_channels := listUniqueChannels(chartRawData)

	for _, channel := range all_channels {
		SubsetEvtxForChannel := getEvtxChartDataForChannel(channel, chartRawData)

		// Empty Evtx for aggreation/deduplication
		VirtualEvtxForChannel := SingleEvtxChartData{
			Hostname:           "VirtualEvtx",
			WinningHostname:    SubsetEvtxForChannel[0].WinningHostname,
			NanoSecPrecision:   SubsetEvtxForChannel[0].NanoSecPrecision,
			Filename:           "VirtualEvtx",
			Path:               "VirtualEvtx",
			Channel:            channel,
			record_counter:     -1,
			alternative_header: common.EVTXHeaderAlternative{},
			EventsByYear:       make(map[int]YearStruct),
		}

		// Shortcut when SubsetEvtxForChannel == 1
		if len(SubsetEvtxForChannel) == 1 {
			VirtualEvtxForChannel = SubsetEvtxForChannel[0]
		} else {
			// Merge multiple evtx files with the same channel
			for _, singleevtx := range SubsetEvtxForChannel {
				for year, yearStruct := range singleevtx.EventsByYear {
					for month, monthStruct := range yearStruct.Months {
						for day, dayStruct := range monthStruct.Days {

							// Precision Hour VS NanoSEC
							if VirtualEvtxForChannel.NanoSecPrecision {
								// NanoSec case
								for hour, hourStruct := range dayStruct.HoursNanosec {
									for minute, minuteStruct := range hourStruct.Minutes {
										for second, secondStruct := range minuteStruct.Seconds {
											for nanosec, _ := range secondStruct.NanoSec {

												if _, ok := VirtualEvtxForChannel.EventsByYear[year]; !ok {
													VirtualEvtxForChannel.EventsByYear[year] = YearStruct{Months: make(map[int]MonthStruct)}
												}

												if _, ok := VirtualEvtxForChannel.EventsByYear[year].Months[month]; !ok {
													VirtualEvtxForChannel.EventsByYear[year].Months[month] = MonthStruct{Days: make(map[int]DayStruct)}
												}

												if _, ok := VirtualEvtxForChannel.EventsByYear[year].Months[month].Days[day]; !ok {
													VirtualEvtxForChannel.EventsByYear[year].Months[month].Days[day] = DayStruct{
														HoursNanosec: make(map[int]HourStruct),
														HoursHours:   nil,
													}
												}

												if _, ok := VirtualEvtxForChannel.EventsByYear[year].Months[month].Days[day].HoursNanosec[hour]; !ok {
													VirtualEvtxForChannel.EventsByYear[year].Months[month].Days[day].HoursNanosec[hour] = HourStruct{
														Minutes: make(map[int]MinuteStruct),
													}
												}

												if _, ok := VirtualEvtxForChannel.EventsByYear[year].Months[month].Days[day].HoursNanosec[hour].Minutes[minute]; !ok {
													VirtualEvtxForChannel.EventsByYear[year].Months[month].Days[day].HoursNanosec[hour].Minutes[minute] = MinuteStruct{
														Seconds: make(map[int]SecondStruct),
													}
												}

												if _, ok := VirtualEvtxForChannel.EventsByYear[year].Months[month].Days[day].HoursNanosec[hour].Minutes[minute].Seconds[second]; !ok {
													VirtualEvtxForChannel.EventsByYear[year].Months[month].Days[day].HoursNanosec[hour].Minutes[minute].Seconds[second] = SecondStruct{
														NanoSec: make(map[int]int),
													}
												}

												finalStruct := VirtualEvtxForChannel.EventsByYear[year].Months[month].Days[day].HoursNanosec[hour].Minutes[minute].Seconds[second]

												if _, ok := finalStruct.NanoSec[nanosec]; ok {
													finalStruct.NanoSec[nanosec] += 1
												} else {
													finalStruct.NanoSec[nanosec] = 1
												}
											}
										}
									}
								}
							} else {
								// Hour precision
								for hour, _ := range dayStruct.HoursHours {
									finalStruct := VirtualEvtxForChannel.EventsByYear[year].Months[month].Days[day]
									if _, ok := finalStruct.HoursHours[hour]; ok {
										finalStruct.HoursHours[hour] += 1
									} else {
										finalStruct.HoursHours[hour] = 1
									}
								}
							}

						} // End of days loop
					}
				}
			}

		} // Else: End of merging loop

		// Fill PreMap
		for year, yearStruct := range VirtualEvtxForChannel.EventsByYear {
			for month, monthStruct := range yearStruct.Months {
				for day, dayStruct := range monthStruct.Days {
					DailyCounter := 0
					var dateKey = ""

					// Precision Hour VS NanoSEC as an input
					if VirtualEvtxForChannel.NanoSecPrecision {
						// Nanosec case
						for hour, hourStruct := range dayStruct.HoursNanosec {
							HourlyCounter := 0

							for _, minuteStruct := range hourStruct.Minutes {
								for _, secondStruct := range minuteStruct.Seconds {
									// Doesn't count number of events inside NanoSec to get rid of duplicates
									HourlyCounter += len(secondStruct.NanoSec)
								}
							}
							// HourlyCounter valid from this point
							if hourPrecision {
								dateKey = fmt.Sprintf(key_format, year, month, day, hour)

								if _, exists := dateChannelCounterMap[dateKey]; !exists {
									dateChannelCounterMap[dateKey] = make(map[string]int)
								}
								dateChannelCounterMap[dateKey][VirtualEvtxForChannel.Channel] = HourlyCounter
							}

							DailyCounter += HourlyCounter
						}

					} else {
						// Hour case
						for hour, HourlyCounter := range dayStruct.HoursHours {

							// HourlyCounter valid from this point
							if hourPrecision {
								dateKey = fmt.Sprintf(key_format, year, month, day, hour)

								if _, exists := dateChannelCounterMap[dateKey]; !exists {
									dateChannelCounterMap[dateKey] = make(map[string]int)
								}
								dateChannelCounterMap[dateKey][VirtualEvtxForChannel.Channel] = HourlyCounter
							}

							DailyCounter += HourlyCounter
						}
					}

					// DailyCounter valid from this point
					if hourPrecision == false {
						dateKey = fmt.Sprintf(key_format, year, month, day)

						if _, exists := dateChannelCounterMap[dateKey]; !exists {
							dateChannelCounterMap[dateKey] = make(map[string]int)
						}
						dateChannelCounterMap[dateKey][VirtualEvtxForChannel.Channel] = DailyCounter
					}
				}
			}
		}

	} // Channel loop end

	return dateChannelCounterMap

}

func generateJavaScriptRecords(PreMap map[string]map[string]int, hourPrecision bool) []Record {
	var records []Record

	// Key format (Hour vs daily precision)
	var key_format = KEY_FORMAT_DAILY

	if hourPrecision {
		key_format = KEY_FORMAT_HOUR
	}

	for dateKey, channelCounterMap := range PreMap {
		var year, month, day, hour int

		if hourPrecision {
			fmt.Sscanf(dateKey, key_format, &year, &month, &day, &hour)
		} else {
			fmt.Sscanf(dateKey, key_format, &year, &month, &day)
			hour = 0
		}

		records = append(records, Record{
			Year:     year,
			Month:    month,
			Day:      day,
			Hour:     hour,
			Channels: channelCounterMap,
		})
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].Year != records[j].Year {
			return records[i].Year < records[j].Year
		}
		if records[i].Month != records[j].Month {
			return records[i].Month < records[j].Month
		}
		if records[i].Day != records[j].Day {
			return records[i].Day < records[j].Day
		}
		return records[i].Hour < records[j].Hour
	})

	return records
}

func generateChartHTML(output_dir string, hostname string, alternative_hostnames []string, templates_path string, chartRawData []SingleEvtxChartData) {

	// Prepare output path
	output_path_html := GetOutputPath(output_dir, hostname, "html")
	common.EnsureDirectoryStructureIsCreated(output_path_html)

	// Function "add"
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	}

	// Load template
	tmpl, err := template.New("apache_echarts.tmpl").Funcs(funcMap).ParseFiles(templates_path + "apache_echarts.tmpl")
	if err != nil {
		log.Fatalf("Error parsing template: %s", err)
	}

	var hour_precision bool = false

	// Prepare data
	PreMap := generateDeduplicatedMap(chartRawData, hour_precision)

	// Calculate statistics
	records := generateJavaScriptRecords(PreMap, hour_precision)
	records = mergeAndLowerCaseChannelJavaScriptRecords(records)
	existing_labels_sorted := generateExistingLegendLabels(records)
	dynamic_lines := recordToDynamicLine(records, existing_labels_sorted)

	// Generate subtext
	var subtext = ""
	if len(alternative_hostnames) > 0 {
		subtext = "Alternative hostnames: " + strings.Join(alternative_hostnames, ", ")
	}
	subtext += fmt.Sprintf(" [Based on %d non-empty files] Timezone: UTC", len(chartRawData))

	data := map[string]interface{}{
		"PageTitle":        fmt.Sprintf("Chart - %s", hostname),
		"PageTitleSubtext": subtext,
		"DynamicLines":     dynamic_lines,
		"Hostname":         hostname,
		"Legends":          existing_labels_sorted,
		"ChannelColors":    CHANNEL_COLORS,
	}

	outputFile, err := os.Create(output_path_html)
	if err != nil {
		log.Fatalf("Error creating output file: %s", err)
	}

	defer outputFile.Close()

	err = tmpl.Funcs(funcMap).Execute(outputFile, data)
	if err != nil {
		log.Fatalf("Error executing template: %s", err)
	}

}

func generateExistingLegendLabels(records []Record) []string {
	var legendLabelsRandom = make([]string, 0)
	seenLegends := make(map[string]bool)

	for _, record := range records {
		for channel, _ := range record.Channels {
			if !seenLegends[channel] {
				seenLegends[channel] = true
				legendLabelsRandom = append(legendLabelsRandom, channel)

				if len(legendLabelsRandom) == len(CHANNEL_COLORS) {
					break
				}
			}
		}
	}

	//sort.Sort(sort.StringSlice(legendLabelsSorted))
	// Fixed Order:
	// Top of bar is Other
	order := []string{"Security", "Microsoft-Windows-Sysmon/Operational", "System", "Application", "Microsoft-Windows-PowerShell/Operational", "Windows PowerShell", "Other"}

	var legendLabelsOrdered []string
	legendLabelsOrdered = make([]string, 0)

	for _, label := range order {
		// Append to legendLabelsOrdered only if label exists in legendLabelsRandom
		if slices.Contains(legendLabelsRandom, label) {
			legendLabelsOrdered = append(legendLabelsOrdered, label)
		}
	}

	return legendLabelsOrdered
}

func removeLastComma(sb *strings.Builder) {
	str := sb.String()
	if len(str) > 0 && str[len(str)-1] == ',' {
		str = str[:len(str)-1]
	}
	sb.Reset()
	sb.WriteString(str)
}

func recordChannelToEventsNumber(record Record, channel string) int {
	if value, ok := record.Channels[channel]; ok {
		return value
	}

	return 0
}

func recordToDynamicLine(records []Record, existing_labels_sorted []string) []string {
	var dynamicLines []string

	for _, record := range records {
		var readyLine strings.Builder

		readyLine.WriteString(fmt.Sprintf("u(%d, %d, %d, %d,       ", record.Year, record.Month, record.Day, record.Hour))

		// Important channels
		for _, label := range existing_labels_sorted {
			readyLine.WriteString(fmt.Sprintf("%d,", recordChannelToEventsNumber(record, label)))
		}
		removeLastComma(&readyLine)

		readyLine.WriteString(")")

		dynamicLines = append(dynamicLines, readyLine.String())
	}

	return dynamicLines
}

func mergeAndLowerCaseChannelJavaScriptRecords(records []Record) []Record {
	for i := range records {
		otherCount := 0
		newChannels := make(map[string]int)

		for channel, count := range records[i].Channels {
			if _, exists := CHANNEL_COLORS[channel]; !exists {
				otherCount += count
			} else {
				newChannels[channel] = count
			}
		}

		if otherCount > 0 {
			newChannels["Other"] = otherCount
		}

		records[i].Channels = newChannels
	}

	return records
}
