package engine

import (
	"fmt"
	"github.com/Velocidex/ordereddict"
	"github.com/yarox24/EvtxHussar/common"
	"github.com/yarox24/EvtxHussar/eventmap"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"strings"
)

type Engine struct {
	Layer1         []Layer1
	Layer2         []Layer2
	EventsCache    map[string]map[string]map[string]Layer1EventsEnhanced // EventsCache[l2_name][channel][eid] = l1events_enhanced
	DoubleQuotes   map[string]common.Params
	SIDList        map[string]string
	VariousMappers map[string]common.Params
	Common         Layer1
	Maps_path      string
	OutputFormat   string
}

func NewEngine(output_format string, maps_path string) Engine {
	return Engine{
		Layer1:         make([]Layer1, 0),
		Layer2:         make([]Layer2, 0),
		EventsCache:    make(map[string]map[string]map[string]Layer1EventsEnhanced, 0),
		DoubleQuotes:   make(map[string]common.Params, 0),
		SIDList:        make(map[string]string, 0),
		VariousMappers: make(map[string]common.Params, 0),
		Common:         Layer1{},
		Maps_path:      maps_path,
		OutputFormat:   output_format,
	}
}

func (e *Engine) AllowLayerToBeAppendedBasedOnL2Name(IncludeOnly common.CommaSeparated, ExcludeOnly common.CommaSeparated, l2name string) bool {

	// IncludeOnly - MODE
	if len(IncludeOnly.Entries) > 0 {
		if common.StringSliceContainsCaseInsensitive(IncludeOnly.Entries, l2name) {
			return true
		} else {
			return false
		}
		// ExcludeOnly - MODE
	} else if len(ExcludeOnly.Entries) > 0 {
		if common.StringSliceContainsCaseInsensitive(ExcludeOnly.Entries, l2name) {
			return false
		} else {
			return true
		}
	}

	// No filtering applied
	return true
}

func (e *Engine) LoadLayer1(IncludeOnly common.CommaSeparated, ExcludeOnly common.CommaSeparated) {

	l1_files, err := ioutil.ReadDir(e.Maps_path)

	if err != nil {
		common.LogCriticalErrorWithError("When reading maps directory", err)
	}
	common.LogDebugStructure("L1 files", l1_files, "l1_files")

	// Layer1 - maps/*.yaml
	for _, f := range l1_files {

		// Skip directories
		if f.IsDir() {
			continue
		}

		file_bytes, err := ioutil.ReadFile(e.Maps_path + f.Name())

		if err != nil {
			common.LogCriticalErrorWithError("When reading L1 map: "+f.Name(), err)
		}

		var l1 = new(Layer1)
		err = yaml.Unmarshal(file_bytes, l1)

		if err != nil {
			common.LogCriticalErrorWithError("When unmarshalling L1 map: "+f.Name(), err)
		}

		if l1.Info.Typ == "common" {
			e.Common = *l1
			common.LogDebug("Loaded common map (L1)")
		} else if l1.Info.Typ == "layer1" {

			// Skip loading filtered layers
			if e.AllowLayerToBeAppendedBasedOnL2Name(IncludeOnly, ExcludeOnly, l1.Sendto_layer2) {
				common.LogDebug("Loaded L1 (layer1) map: " + f.Name())
			} else {
				common.LogDebug("Skip loading (based on filter) L1 (layer1) map: " + f.Name())
				continue
			}

			// Logic engine
			for eid, _ := range l1.Events {
				event := l1.Events[eid]

				// Global logic lowercase
				event.Matching_Rules.Global_Logic = strings.ToLower(event.Matching_Rules.Global_Logic)

				// Initalize
				event.Matching_Rules.Container_OrEnhanced = make([][]common.ExtractedLogic, 0)

				// Enhance logic OR
				event.Matching_Rules.EnhanceRulesInPlace()

				l1.Events[eid] = event
			}

			// Enhance
			l1.EventsEnhanced = make(map[string]Layer1EventsEnhanced, 0)

			for eid, l1event := range l1.Events {
				l1.EventsEnhanced[eid] = NewLayer1EventsEnhanced(&l1event)
			}

			e.Layer1 = append(e.Layer1, *l1)

		} else {
			panic("YAML - LoadLayer1() - Unsupported Info.Typ")
		}
	}
}

func (e *Engine) LoadLayer2(Output_dir string, IncludeOnly common.CommaSeparated, ExcludeOnly common.CommaSeparated) {

	l2_layer_dir := e.Maps_path + "layer2" + string(os.PathSeparator)
	l2_files, err := ioutil.ReadDir(l2_layer_dir)

	if err != nil {
		common.LogCriticalErrorWithError("When reading maps/layer2 directory", err)
	}
	common.LogDebugStructure("L2 files", l2_files, "l2_files")

	// Layer2 - maps/layer2/*.yaml
	for _, f := range l2_files {

		// Skip directories
		if f.IsDir() {
			continue
		}

		file_bytes, err := ioutil.ReadFile(l2_layer_dir + f.Name())

		if err != nil {
			common.LogCriticalErrorWithError("When reading L2 map: "+f.Name(), err)
		}

		var l2 = new(Layer2)
		err = yaml.Unmarshal(file_bytes, l2)

		if err != nil {
			common.LogCriticalErrorWithError("When unmarshalling L2 map: "+f.Name(), err)
		}

		if l2.Info.Typ == "layer2" {

			// Skip loading filtered layers
			if e.AllowLayerToBeAppendedBasedOnL2Name(IncludeOnly, ExcludeOnly, l2.Info.Name) {
				common.LogDebug("Loaded L2 (layer2) map: " + f.Name())
			} else {
				common.LogDebug("Skip loading (based on filter) L2 (layer2) map: " + f.Name())
				continue
			}

			// Append OutputDir
			l2.Output.GlobalOutputDirectory = Output_dir + string(os.PathSeparator)

			// Case insensitive remmaping dict
			l2.Fields_remap_dict = ordereddict.NewDict()
			l2.Fields_remap_dict.SetCaseInsensitive()

			// Field extra transformations - Options append & Rename
			for i, trans := range l2.Field_extra_transformations {
				ef := common.FunctionExtractor(trans.Special_transform)
				l2.Field_extra_transformations[i].Special_transform = ef.Name
				l2.Field_extra_transformations[i].Options = ef.Options
			}

			// Enhance ordered fields
			l2.Ordered_fields_enhanced = make(map[string]common.SingleField, 0)

			for i := 0; i < len(l2.Ordered_fields); i++ {
				sf := e.SingleFieldExtractor(l2.Ordered_fields[i])
				l2.Ordered_fields_enhanced[strings.ToLower(sf.NiceName)] = sf
				l2.Ordered_fields[i] = sf.NiceName
			}

			for k, v := range l2.Fields_remap {
				l2.Fields_remap_dict.Set(k, v)
			}

			e.Layer2 = append(e.Layer2, *l2)
		} else {
			panic("YAML - LoadLayer2() - Unsupported Info.Typ")
		}
	}
}

func (e *Engine) IncreaseUsageCounterForLayer2(l2name string) {
	for i := 0; i < len(e.Layer2); i++ {
		if e.Layer2[i].Info.Name == l2name {
			e.Layer2[i].UsageCounter += 1
			return
		}
	}

	panic("Wrong name for sendto_layer2")
}

func (e *Engine) GetAllLayer1WhichSupportsChannel(channel string) []*Layer1 {
	var temp = make([]*Layer1, 0)

	for i := 0; i < len(e.Layer1); i++ {
		if strings.ToLower(e.Layer1[i].Info.Channel) == strings.ToLower(channel) {
			temp = append(temp, &e.Layer1[i])
		}
	}
	return temp
}

func (e *Engine) IsEfiSupported(efi *common.EvtxFileInfo) {

	// Is valid
	if !efi.IsValid() {
		return
	}

	// Is non-empty | This might not be necessary
	if efi.IsEmpty() {
		return
	}

	// Is channel supported?
	ch := strings.ToLower(efi.GetChannel())

	for _, llayer1 := range e.Layer1 {
		if ch == strings.ToLower(llayer1.Info.Channel) {
			efi.EnableForProcessing()
			e.IncreaseUsageCounterForLayer2(llayer1.Sendto_layer2)
		}
	}

	return
}

func (e *Engine) FindL2LayerByName(name string) *Layer2 {

	for i := 0; i < len(e.Layer2); i++ {
		var l2 = e.Layer2[i]
		if strings.ToLower(l2.Info.Name) == strings.ToLower(name) {
			return &l2
		}

	}

	return nil
}

func (e *Engine) PrepareCommonFieldsEmptyOrderedDict() *ordereddict.Dict {
	o := ordereddict.NewDict()
	o.SetCaseInsensitive()

	for _, v := range e.Common.Ordered_fields {
		o.Set(v, nil)
	}

	return o
}

func (e *Engine) PrepareLayer2FieldsEmptyOrderedDict(l2_name string) *ordereddict.Dict {
	o := ordereddict.NewDict()
	o.SetCaseInsensitive()

	active_l2 := e.FindL2LayerByName(l2_name)

	for _, k := range active_l2.Ordered_fields {
		o.Set(k, nil)
	}

	return o
}

func (e *Engine) PrepareCommonAndLayer2FieldsEmptyOrderedDict(l2_name string) *ordereddict.Dict {

	// Common
	ord_map := e.PrepareCommonFieldsEmptyOrderedDict()

	// Append Active Layer2
	ord_map.MergeFrom(e.PrepareLayer2FieldsEmptyOrderedDict(l2_name))

	return ord_map
}

func (e *Engine) GetCSVHeadersOrdered(l2_name string) []string {
	ord_map := e.PrepareCommonAndLayer2FieldsEmptyOrderedDict(l2_name)
	return common.OrderedDictToKeysOrderedStringList(ord_map)
}

func (e *Engine) ParseCommonFieldsOrderedDict(ev_map *eventmap.EventMap, l2_name string) *ordereddict.Dict {
	ord_map := e.PrepareCommonFieldsEmptyOrderedDict()

	//EventTime
	if _, eventtimeok := ord_map.Get("EventTime"); eventtimeok {
		ord_map.Update("EventTime", eventmap.GetSystemTime(ev_map, e.Common.Options["HighPrecisionEventTime"]))
	}

	//EID
	if _, eidok := ord_map.Get("EID"); eidok {
		ord_map.Update("EID", eventmap.GetEID(ev_map))
	}

	//EID [Description]
	if _, eidok := ord_map.Get("Description"); eidok {
		ord_map.Update("Description", e.GetEIDDescription(ev_map, l2_name))
	}

	//Computer
	if _, current_computer_ok := ord_map.Get("Computer"); current_computer_ok {
		ord_map.Update("Computer", eventmap.GetCurrentComputer(ev_map))
	}

	//Channel
	if _, channel_ok := ord_map.Get("Channel"); channel_ok {
		ord_map.Update("Channel", eventmap.GetChannel(ev_map))
	}

	//Provider
	if _, provider_ok := ord_map.Get("Provider"); provider_ok {
		ord_map.Update("Provider", eventmap.GetProvider(ev_map))
	}

	//EventRecord ID
	if _, erid_ok := ord_map.Get("EventRecord ID"); erid_ok {
		ord_map.Update("EventRecord ID", eventmap.GetEventRecordID(ev_map))
	}

	// Keywords
	if _, erid_ok := ord_map.Get("Keywords"); erid_ok {
		ord_map.Update("Keywords", eventmap.GetKeywords(ev_map))
	}

	//Correlation ActivityID
	if _, erid_ok := ord_map.Get("Correlation ActivityID"); erid_ok {
		ord_map.Update("Correlation ActivityID", eventmap.GetCorrelationActivityID(ev_map))
	}

	//System Process ID
	if _, sysprocid_ok := ord_map.Get("System Process ID"); sysprocid_ok {
		ord_map.Update("System Process ID", eventmap.GetSystemProcessID(ev_map))
	}

	//Security User ID
	if _, secuid_ok := ord_map.Get("Security User ID"); secuid_ok {
		ord_map.Update("Security User ID", eventmap.GetSecurityUserID(ev_map))
	}

	return ord_map
}

func (e *Engine) ParseL2FieldsOrderedDict(l2_name string, ev_map *eventmap.EventMap) *ordereddict.Dict {

	channel := eventmap.GetChannel(ev_map)
	eid := eventmap.GetEID(ev_map)
	l2_current := e.FindL2LayerByName(l2_name)

	// Empty dict with fields
	ord_map := e.PrepareLayer2FieldsEmptyOrderedDict(l2_name)

	// Attrib extraction - RAW data types
	attrib_map := eventmap.ExtractAttribs(ev_map, e.EventsCache[l2_name][channel][eid].Attrib_extraction, false)

	// Check
	len_before := ord_map.Len()

	// Convert to string type
	eventmap.MapAttribToOrderedMap(attrib_map, ord_map, l2_current.Fields_remap_dict, l2_current.Ordered_fields_enhanced)

	if ord_map.Len() != len_before {
		common.LogError("Wrong numbers of arguments mapped - MapAttribToOrderedMap")
	}

	// Resolve - Mappers & Double Quotes (Optional)
	eventmap.ResolveMappersAndDoubleQuotesInPlace(ord_map, l2_current.Ordered_fields_enhanced, e.VariousMappers, e.GetDoubleQuotesForChannel(channel), e.SIDList)

	// Special transformations
	if len(l2_current.Field_extra_transformations) > 0 {
		eventmap.ApplySpecialTransformations(ord_map, l2_current.Field_extra_transformations)
	}

	return ord_map
}

func (e *Engine) SingleFieldExtractor(function string) common.SingleField {
	var sf = common.SingleField{
		NiceName: "",
		Options:  make(map[string]string, 0),
	}

	temp1 := strings.SplitN(function, ":", 2)

	// Set name
	sf.NiceName = temp1[0]

	// Optional options
	if len(temp1) > 1 {
		remaining := temp1[1]
		// Options separated by ,
		temp2 := strings.Split(remaining, ",")

		for _, option := range temp2 {
			opt_split := strings.Split(option, "=")

			if len(opt_split) != 2 {
				common.LogError(fmt.Sprintf("[SingleFieldExtractor critical error] %s", "wrong nr of fields after = split"))
				continue
			}
			sf.Options[opt_split[0]] = opt_split[1]
		}
	}

	return sf
}

func (e *Engine) GetEIDDescription(ev_map *ordereddict.Dict, l2_name string) string {
	eid := eventmap.GetEID(ev_map)
	channel := eventmap.GetChannel(ev_map)

	return e.EventsCache[l2_name][channel][eid].Short_description
}

func (e *Engine) LoadParams() {

	params_dir := e.Maps_path + "params" + string(os.PathSeparator)
	params_files, err := ioutil.ReadDir(params_dir)

	if err != nil {
		common.LogCriticalErrorWithError("When reading params directory", err)
	}

	common.LogDebugStructure("Params files", params_files, "params_files")

	// Layer2 - maps/params/*.yaml
	for _, f := range params_files {

		// Skip directories
		if f.IsDir() {
			continue
		}

		file_bytes, err := ioutil.ReadFile(params_dir + f.Name())

		if err != nil {
			common.LogCriticalErrorWithError("When reading params: "+f.Name(), err)
		}

		var p = new(common.Params)
		err = yaml.Unmarshal(file_bytes, p)

		if err != nil {
			common.LogCriticalErrorWithError("When unmarshalling params: "+f.Name(), err)
		}

		if p.Info.Typ == "doublequotes" {
			channel := p.Info.Channel
			e.DoubleQuotes[channel] = *p
			common.LogDebug("Loaded params (doublequotes) map: " + f.Name())
		} else if p.Info.Typ == "mapper_number_to_string" || p.Info.Typ == "mapper_string_to_string" || p.Info.Typ == "mapper_bitwise_to_string" {
			name := p.Info.Name
			e.VariousMappers[name] = *p
			common.LogDebug("Loaded params (mapper) map: " + f.Name())
		} else if p.Info.Typ == "sidlist" {
			//println(p.Params)
			e.SIDList = p.Params
			common.LogDebug("Loaded SIDList (mapper) map: " + f.Name())
		} else {
			panic("YAML - LoadLayer2() - Unsupported Info.Typ")
		}
	}

}

func (e *Engine) GetDoubleQuotesForChannel(channel string) map[string]string {
	if p, exists := e.DoubleQuotes[channel]; exists {
		return p.Params
	}

	return nil
}

func (e *Engine) PrepareEventCache() {
	for _, l2 := range e.Layer2 {

		// l2_name
		l2_name := l2.Info.Name
		if _, l2_name_exists := e.EventsCache[l2_name]; !l2_name_exists {
			e.EventsCache[l2_name] = make(map[string]map[string]Layer1EventsEnhanced, 0)
		}

		// channel
		l1list := e.GetAllLayer1WhichSupportsLayer2(l2_name)

		for _, l1 := range l1list {
			channel := l1.Info.Channel

			if _, channel_exists := e.EventsCache[l2_name][channel]; !channel_exists {
				e.EventsCache[l2_name][channel] = make(map[string]Layer1EventsEnhanced, 0)
			}

			// eid and v
			for eid, l1e_enhanced := range l1.EventsEnhanced {
				if _, eid_exists := e.EventsCache[l2_name][channel][eid]; !eid_exists {
					e.EventsCache[l2_name][channel][eid] = l1e_enhanced
				}

			}

		}

	}

}

func (e *Engine) GetAllLayer1WhichSupportsLayer2(l2_name string) []Layer1 {
	out := make([]Layer1, 0)

	for _, l1 := range e.Layer1 {
		if strings.ToLower(l1.Sendto_layer2) == strings.ToLower(l2_name) {
			out = append(out, l1)
		}
	}

	return out
}
