package tests

import (
	"github.com/rs/zerolog"
	"github.com/yarox24/EvtxHussar/chart"
	"github.com/yarox24/EvtxHussar/common"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func CheckFileAgainstSubstrings(t *testing.T, Path string, SubstringsRequired []string) {
	file, err := os.Open(Path)
	if err != nil {
		t.Fatalf("Chart file not present %s: %v", Path, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read chart file %s: %v", Path, err)
	}

	contentStr := string(content)

	for _, substring := range SubstringsRequired {
		found := false
		lines := strings.Split(contentStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, substring) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Substring '%s' not found in the chart file %s", substring, Path)
		}
	}
}

func TestChartGeneration(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	defer zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Preparation phase
	_, templates_path := GrabCmdlineMapAndTemplatesPath()
	templates_path, _ = common.Determine_Templates_Path(templates_path)

	_, filename, _, _ := runtime.Caller(0)
	testChartDir := filepath.Join(filepath.Dir(filename), "../tests/files/chart")
	var ChartOutputDir1 = t.TempDir()
	var ChartOutputDir2 = t.TempDir()

	// Persistent dir for debugging
	//var ChartOutputDir = filepath.Join(filepath.Dir(filename), "../tests/delme")

	// VSS Case
	testChartVSSDir := filepath.Join(testChartDir, "vss")
	var EfiList1 = common.Generate_list_of_files_to_process([]string{testChartVSSDir}, true)
	EfiList1 = common.Inspect_evtx_paths(EfiList1)

	chart.GenerateFrequencyChart("html", ChartOutputDir1, templates_path, EfiList1, 5)

	FinalChartFilename1 := filepath.Join(ChartOutputDir1, "win10/chart/frequency_distribution_win10.html")
	CheckFileAgainstSubstrings(t,
		FinalChartFilename1,
		[]string{
			"u(2022, 4, 22, 0,       265)",
			"u(2022, 4, 23, 0,       86)",
			"<title>Chart - win10</title>",
		},
	)

	// Non VSS Case #HTML test
	testChartSingleFile := filepath.Join(testChartDir, "vss/Security4624vss1.evtx")
	var EfiList2 = common.Generate_list_of_files_to_process([]string{testChartSingleFile}, false)
	EfiList2 = common.Inspect_evtx_paths(EfiList2)

	chart.GenerateFrequencyChart("html", ChartOutputDir2, templates_path, EfiList2, 5)

	FinalChartFilename2 := filepath.Join(ChartOutputDir2, "win10/chart/frequency_distribution_win10.html")
	CheckFileAgainstSubstrings(t,
		FinalChartFilename2,
		[]string{
			"u(2022, 4, 22, 0,       150)",
			"u(2022, 4, 23, 0,       86)",
			"<title>Chart - win10</title>",
		},
	)

}
