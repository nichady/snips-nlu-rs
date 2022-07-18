package snips

// #include <stdio.h>
// #include <libsnips_nlu.h>
import "C"
import (
	"errors"
	"unsafe"
)

func GetModelVersion() (string, error) {
	var version *C.char
	defer C.snips_nlu_engine_destroy_string(version)

	err := parseErr(C.snips_nlu_engine_get_model_version(&version))
	if err != nil {
		return "", err
	}

	return C.GoString(version), nil
}

func parseErr(result C.SNIPS_RESULT) error {
	if result != C.SNIPS_RESULT_OK {
		var err *C.char
		defer C.snips_nlu_engine_destroy_string(err)

		C.snips_nlu_engine_get_last_error(&err)
		return errors.New(C.GoString(err))
	}

	return nil
}

func NewIntentEngineFromDir(dir string) (*IntentEngine, error) {
	cdir := C.CString(dir)
	defer C.free(unsafe.Pointer(cdir))

	var client *C.CSnipsNluEngine
	err := parseErr(C.snips_nlu_engine_create_from_dir(cdir, &client))
	if err != nil {
		return nil, err
	}

	ie := IntentEngine{client: client}
	return &ie, nil
}

func NewIntentEngineFromZip(zip []byte) (*IntentEngine, error) {
	var client *C.CSnipsNluEngine
	err := parseErr(C.snips_nlu_engine_create_from_zip((*C.uchar)(&zip[0]), C.uint(len(zip)), &client))
	if err != nil {
		return nil, err
	}

	ie := IntentEngine{client: client}
	return &ie, nil
}

type IntentEngine struct {
	client *C.CSnipsNluEngine
}

func (ie *IntentEngine) GetIntents(input string) ([]IntentClassifierResult, error) {
	cinput := C.CString(input)
	defer C.free(unsafe.Pointer(cinput))

	var cresult *C.CIntentClassifierResultArray
	defer C.snips_nlu_engine_destroy_intent_classifier_results(cresult)

	err := parseErr(C.snips_nlu_engine_run_get_intents(ie.client, cinput, &cresult))
	if err != nil {
		return nil, err
	}

	result := make([]IntentClassifierResult, cresult.size)
	for i, v := range unsafe.Slice(cresult.intent_classifier_results, cresult.size) {
		result[i] = IntentClassifierResult{
			IntentName:      C.GoString(v.intent_name),
			ConfidenceScore: float32(v.confidence_score),
		}
	}

	return result, nil
}

func (ie *IntentEngine) GetIntentsIntoJson(input string) (string, error) {
	cinput := C.CString(input)
	defer C.free(unsafe.Pointer(cinput))

	var json *C.char
	defer C.snips_nlu_engine_destroy_string(json)

	err := parseErr(C.snips_nlu_engine_run_get_intents_into_json(ie.client, cinput, &json))
	if err != nil {
		return "", err
	}

	return C.GoString(json), nil
}

func (ie *IntentEngine) GetSlots(input, intent string) ([]Slot, error) {
	return ie.GetSlotsWithAlternatives(input, intent, 0)
}

func (ie *IntentEngine) GetSlotsIntoJson(input, intent string) (string, error) {
	return ie.GetSlotsWithAlternativesIntoJson(input, intent, 0)
}

func (ie *IntentEngine) GetSlotsWithAlternatives(input, intent string, slotsAlternatives uint) ([]Slot, error) {
	cinput := C.CString(input)
	defer C.free(unsafe.Pointer(cinput))

	cintent := C.CString(intent)
	defer C.free(unsafe.Pointer(cintent))

	var cresult *C.CSlotList
	defer C.snips_nlu_engine_destroy_slots(cresult)

	err := parseErr(C.snips_nlu_engine_run_get_slots_with_alternatives(ie.client, cinput, cintent, C.uint(slotsAlternatives), &cresult))
	if err != nil {
		return nil, err
	}

	result := make([]Slot, cresult.size)
	for i, v := range unsafe.Slice(cresult.slots, cresult.size) {
		result[i] = Slot{
			Value:           parseSlotValue(v.value),
			Alternatives:    parseSlotAlternatives(v.alternatives),
			RawValue:        C.GoString(v.raw_value),
			Entity:          C.GoString(v.entity),
			SlotName:        C.GoString(v.slot_name),
			RangeStart:      int32(v.range_start),
			RangeEnd:        int32(v.range_end),
			ConfidenceScore: float32(v.confidence_score),
		}
	}

	return result, nil
}

func (ie *IntentEngine) GetSlotsWithAlternativesIntoJson(input, intent string, slotsAlternatives uint) (string, error) {
	cinput := C.CString(input)
	defer C.free(unsafe.Pointer(cinput))

	cintent := C.CString(intent)
	defer C.free(unsafe.Pointer(cintent))

	var json *C.char
	defer C.snips_nlu_engine_destroy_string(json)

	err := parseErr(C.snips_nlu_engine_run_get_slots_with_alternatives_into_json(ie.client, cinput, cintent, C.uint(slotsAlternatives), &json))
	if err != nil {
		return "", err
	}

	return C.GoString(json), nil
}

func (ie *IntentEngine) Close() error {
	return parseErr(C.snips_nlu_engine_destroy_client(ie.client))
}

type IntentClassifierResult struct {
	IntentName      string
	ConfidenceScore float32
}

type Slot struct {
	Value           any
	Alternatives    []any
	RawValue        string
	Entity          string
	SlotName        string
	RangeStart      int32
	RangeEnd        int32
	ConfidenceScore float32
}

type CustomValue string
type NumberValue float64
type OrdinalValue int64
type InstantTimeValue struct {
	Value     string
	Grain     Grain
	Precision Precision
}
type TimeIntervalValue struct {
	From, To string
}
type AmountOfMoneyValue struct {
	Unit      string
	Value     float32
	Precision Precision
}
type TemperatureValue struct {
	Unit  string
	Value float32
}
type DurationValue struct {
	Years, Quarters, Months, Weeks, Days, Hours, Minutes, Seconds int64
	Precision                                                     Precision
}
type PercentageValue float64
type MusicAlbumValue string
type MusicArtistValue string
type MusicTrackValue string
type CityValue string
type CountryValue string
type RegionValue string

func parseSlotValue(value *C.CSlotValue) any {
	v := value.value
	switch value.value_type {
	case C.SNIPS_SLOT_VALUE_TYPE_CUSTOM:
		return CustomValue(C.GoString((*C.char)(v)))
	case C.SNIPS_SLOT_VALUE_TYPE_NUMBER:
		return *(*NumberValue)(v)
	case C.SNIPS_SLOT_VALUE_TYPE_ORDINAL:
		return *(*OrdinalValue)(v)
	case C.SNIPS_SLOT_VALUE_TYPE_INSTANTTIME:
		v := (*C.CInstantTimeValue)(v)
		return InstantTimeValue{
			Value:     C.GoString(v.value),
			Grain:     Grain(v.grain),
			Precision: Precision(v.precision),
		}
	case C.SNIPS_SLOT_VALUE_TYPE_TIMEINTERVAL:
		v := (*C.CTimeIntervalValue)(v)
		return TimeIntervalValue{
			From: C.GoString(v.from),
			To:   C.GoString(v.to),
		}
	case C.SNIPS_SLOT_VALUE_TYPE_AMOUNTOFMONEY:
		v := (*C.CAmountOfMoneyValue)(v)
		return AmountOfMoneyValue{
			Unit:      C.GoString(v.unit),
			Value:     float32(v.value),
			Precision: Precision(v.precision),
		}
	case C.SNIPS_SLOT_VALUE_TYPE_TEMPERATURE:
		v := (*C.CTemperatureValue)(v)
		return TemperatureValue{
			Unit:  C.GoString(v.unit),
			Value: float32(v.value),
		}
	case C.SNIPS_SLOT_VALUE_TYPE_DURATION:
		v := (*C.CDurationValue)(v)
		return DurationValue{
			Years:     int64(v.years),
			Quarters:  int64(v.quarters),
			Months:    int64(v.months),
			Weeks:     int64(v.weeks),
			Days:      int64(v.days),
			Hours:     int64(v.hours),
			Minutes:   int64(v.minutes),
			Seconds:   int64(v.seconds),
			Precision: Precision(v.precision),
		}
	case C.SNIPS_SLOT_VALUE_TYPE_PERCENTAGE:
		return *(*PercentageValue)(v)
	case C.SNIPS_SLOT_VALUE_TYPE_MUSICALBUM:
		return MusicAlbumValue(C.GoString((*C.char)(v)))
	case C.SNIPS_SLOT_VALUE_TYPE_MUSICARTIST:
		return MusicArtistValue(C.GoString((*C.char)(v)))
	case C.SNIPS_SLOT_VALUE_TYPE_MUSICTRACK:
		return MusicTrackValue(C.GoString((*C.char)(v)))
	case C.SNIPS_SLOT_VALUE_TYPE_CITY:
		return CityValue(C.GoString((*C.char)(v)))
	case C.SNIPS_SLOT_VALUE_TYPE_COUNTRY:
		return CountryValue(C.GoString((*C.char)(v)))
	case C.SNIPS_SLOT_VALUE_TYPE_REGION:
		return RegionValue(C.GoString((*C.char)(v)))
	}

	return nil
}

func parseSlotAlternatives(alternatives *C.CSlotValueArray) []any {
	a := make([]any, alternatives.size)
	for i, v := range unsafe.Slice(alternatives.slot_values, alternatives.size) {
		a[i] = parseSlotValue(&v)
	}
	return a
}

type Grain int

const (
	GrainYear Grain = iota
	Quarter
	Month
	Week
	Day
	Hour
	Minute
	Second
)

type Precision int

const (
	PrecisionApproximate Precision = iota
	PrecisionExact
)
