# <img src="https://github.com/yarox24/EvtxHussar/blob/447cd68ab8f3a4e5bd9d0197461d81cc162b8202/icon/icons8-forensics-96.png" alt="Icon" width="40"/> EvtxHussar

Initial triage of Windows Event logs. This is beta quality software.

### Input data
- .evtx - Windows event log files coming from various hosts or single host

### Output data
- Chart generation (Event frequency distribution of all *.evtx files)
- Subset of events based on event ID's defined in maps (e.g. System 104 - The log file was cleared.)
- Events useful for forensics
- One of the following output formats: CSV, JSON, JSONL, Excel
- Default output format Excel
- Files with the same computer name are merged

#### Example output
###### Chart (HTML)
[Live chart demo](http://evtxhussardemofiles.s3-website-eu-west-1.amazonaws.com/frequency_distribution_DESKTOP-EVTXHUSSAR.html)
![](evtxhussar_chart_demo.gif)


###### Subset of columns only (Click for fullscreen preview)
![image](https://user-images.githubusercontent.com/18016218/164982801-4fdc2786-0bfb-439a-8679-1ab35537e4c0.png)

###### Output directory structure
![image](https://user-images.githubusercontent.com/18016218/180607885-ece585ea-7d07-4108-a83b-7005f41a4d82.png)


### Interesting features
- Logon related events dumping
- Reconstruction of PowerShell Scriptblocks
- Powershell -enc <base64 string> is automatically decoded
- Scheduled Tasks XML parsing
- Audit changes
- Boot up/Restart/Shutdown events
= SMB related events
- Merge events from different sources (e.g. Microsoft-Windows-PowerShellOperational_General and Windows PowerShell) to single output file
- Deduplication of events (so you can provide logs from backup, VSS, archive)
- Supported events can be easily added by adding .yaml files to maps/ directory
- Parameters resolution (e.g. %%1936 changed to TokenElevationTypeDefault (1))
- Fields resolution (e.g. servicestarttype = 2 is replaced with "Auto start")
- Fields with different names are normalized to single field (whenever possible) e.g. Filename -> TargetFileName

### Which events are supported?
Please look into [maps/](https://github.com/yarox24/EvtxHussar/tree/main/maps "L1 maps") (which contains Layer 1 maps)

### Quick usage

**Parse events (C:\\evtx_compromised_machine\\\*.evtx) from single host to default Excel format (also generate chart)**
```cmd
EvtxHussar.exe -o C:\evtxhussar_results C:\evtx_compromised_machine
```

**Parse events (C:\\evtx_many_machines\\\*\\\*.evtx) from many machines recursively saving them with JSONL format**
```cmd
EvtxHussar.exe -f jsonl -r -o C:\evtxhussar_results C:\evtx_many_machines
```

**Parse only 2 files (Security.evtx and System.evtx) and save them with CSV format**
```cmd
EvtxHussar.exe -f csv -o C:\evtxhussar_results C:\evtx_compromised_machine\Security.evtx C:\evtx_compromised_machine\System.evtx
```

**Parse events with 100 workers (1 worker = 1 Evtx file handled) Default: 30**
```cmd
EvtxHussar.exe -w 100 -r -o C:\evtxhussar_results C:\evtx_many_machines
```

**Parse with custom maps relevant to incident**
```cmd
EvtxHussar.exe -m C:\incident_specific_maps -r -o C:\evtxhussar_results C:\evtx_many_machines
```

**Parse only with selected Layer2 maps e.g. PowerShellUniversal,PowerShellScriptBlock**
```cmd
EvtxHussar.exe --includeonly PowerShellUniversal,PowerShellScriptBlock -r -o C:\evtxhussar_results C:\evtx_many_machines
```

**Parse with all Layer2 maps but exclude e.g. FirewallUniversal**
```cmd
EvtxHussar.exe --excludeonly FirewallUniversal -r -o C:\evtxhussar_results C:\evtx_many_machines
```

**Generate chart only**
```cmd
EvtxHussar.exe --includeonly ChartOnly -r -o C:\evtxhussar_results C:\evtx_many_machines
```

**Parse events only (disable chart generation)**
```cmd
EvtxHussar.exe -c none -r -o C:\evtxhussar_results C:\evtx_many_machines
```

### Usage (as Velociraptor plugin)

https://docs.velociraptor.app/exchange/artifacts/pages/windows.eventlogs.evtxhussar/

### Blog article
:memo:
https://atos.net/en/lp/securitydive/how-to-accelerate-analysis-of-windows-event-logs

### Help
```cmd
Usage: EvtxHussar [--recursive] [--output_dir OUTPUT_DIR] [--format FORMAT] [--workers WORKERS] [--maps MAPS] [--templates TEMPLATES] [--includeonly INCLUDEONLY] [--excludeonly EXCLUDEONLY] [--chart CHART] [--scriptblockxor] [--debug] [INPUT_EVTX_PATHS [INPUT_EVTX_PATHS ...]]

Positional arguments:
  INPUT_EVTX_PATHS       Path(s) to .evtx files or directories containing these files (can be mixed)

Options:
  --recursive, -r        Recursive traversal for any input directories. [default: false]
  --output_dir OUTPUT_DIR, -o OUTPUT_DIR
                         Reports will be saved in this directory (if doesn't exists it will be created)
  --format FORMAT, -f FORMAT
                         Output data in one of the formats: Csv,JSON,JSONL,Excel [default: Excel]
  --workers WORKERS, -w WORKERS
                         Max concurrent workers (.evtx opened) [default: 30]
  --maps MAPS, -m MAPS   Custom directory with maps/ (Default: program directory)
  --templates TEMPLATES, -t TEMPLATES
                         Directory with Apache Echarts template (Default: program directory)
  --includeonly INCLUDEONLY, -i INCLUDEONLY
                         Include only Layer2 maps present on the list comma separated (Name taken from YAML) [default: {[]}]
  --excludeonly EXCLUDEONLY, -e EXCLUDEONLY
                         Start with all Layer2 maps and exclude only maps present on the comma separated list (Name taken from YAML) [default: {[]}]
  --chart CHART, -c CHART
                         Generate frequency chart of all .evtx files (Not only the ones supported by maps). Valid values: html,none [default: html]
  --scriptblockxor, -x   Apply XOR on reconstructed PS ScriptBlocks with key 'Y' (0x59) to prevent deletion by AV [default: false]
  --debug, -d            Be more verbose [default: false]
  --help, -h             display this help and exit
  --version              display version and exit
```
  
  ### Then the winged hussars arrived, coming down they turned the tide
  [![Winged Hussars](https://user-images.githubusercontent.com/18016218/164983755-ce34e0db-4867-4118-8441-d546c090c8a9.png)](https://www.youtube.com/watch?v=rcYhYO02f98 "Winged Hussars")  
