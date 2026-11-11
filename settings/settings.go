package settings

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/databasifylabs/engine-sdk/command"
	"github.com/databasifylabs/engine-sdk/dtos"
)

const (
	KindInteger = "integer"
	KindDecimal = "decimal"
	KindSize    = "size"
	KindBytes   = "bytes"
	KindChoice  = "choice"
	KindFlags   = "flags"
	KindText    = "text"

	SourceUser    = "user"
	SourceTuned   = "tuned"
	SourceDefault = "default"
)

var sizeValue = regexp.MustCompile(`^([0-9]{1,15})(kB|MB|GB|TB)$`)

var sizeFactors = map[string]float64{"kB": 1, "MB": 1024, "GB": 1024 * 1024, "TB": 1024 * 1024 * 1024}

var bytesValue = regexp.MustCompile(`^([0-9]{1,20})([KMGkmg]?)$`)

var bytesFactors = map[string]float64{"": 1, "k": 1024, "m": 1024 * 1024, "g": 1024 * 1024 * 1024}

func BytesOf(value string) (float64, bool) {
	match := bytesValue.FindStringSubmatch(value)
	if match == nil {
		return 0, false
	}
	number, _ := strconv.ParseFloat(match[1], 64)

	return number * bytesFactors[strings.ToLower(match[2])], true
}

type Catalog struct {
	Fields []dtos.ConfigField
	Tune   func(resources dtos.Member) map[string]string
}

type Field struct{ dtos.ConfigField }

func bounded(field dtos.ConfigField, min float64, max float64) dtos.ConfigField {
	field.Min, field.Max = &min, &max

	return field
}

func Integer(key, label, description string, min, max float64, fallback string) Field {
	return Field{bounded(dtos.ConfigField{Key: key, Label: label, Description: description, Kind: KindInteger, Default: fallback, Options: []string{}}, min, max)}
}

func Decimal(key, label, description string, min, max float64, fallback string) Field {
	return Field{bounded(dtos.ConfigField{Key: key, Label: label, Description: description, Kind: KindDecimal, Default: fallback, Options: []string{}}, min, max)}
}

func Size(key, label, description string, minKb, maxKb float64, fallback string) Field {
	unit := "kB"
	field := bounded(dtos.ConfigField{Key: key, Label: label, Description: description, Kind: KindSize, Default: fallback, Options: []string{}, Unit: &unit}, minKb, maxKb)

	return Field{field}
}

func Bytes(key, label, description string, min, max float64, fallback string) Field {
	unit := "B"

	return Field{bounded(dtos.ConfigField{Key: key, Label: label, Description: description, Kind: KindBytes, Default: fallback, Options: []string{}, Unit: &unit}, min, max)}
}

func Choice(key, label, description string, options []string, fallback string) Field {
	return Field{dtos.ConfigField{Key: key, Label: label, Description: description, Kind: KindChoice, Default: fallback, Options: options}}
}

func Flags(key, label, description string, options []string, fallback string) Field {
	return Field{dtos.ConfigField{Key: key, Label: label, Description: description, Kind: KindFlags, Default: fallback, Options: options}}
}

func Text(key, label, description, pattern, fallback string) Field {
	return Field{dtos.ConfigField{Key: key, Label: label, Description: description, Kind: KindText, Default: fallback, Options: []string{}, Pattern: &pattern}}
}

func (f Field) Restarting() Field {
	f.Restart = true

	return f
}

func (f Field) TunedFromServer() Field {
	f.Tuned = true

	return f
}

func (f Field) In(unit string) Field {
	f.Unit = &unit

	return f
}

func Fields(fields ...Field) []dtos.ConfigField {
	plain := make([]dtos.ConfigField, 0, len(fields))
	for _, field := range fields {
		plain = append(plain, field.ConfigField)
	}

	return plain
}

func allows(field dtos.ConfigField, value string) bool {
	switch field.Kind {
	case KindInteger:
		number, err := strconv.ParseInt(value, 10, 64)

		return err == nil && within(field, float64(number))
	case KindDecimal:
		number, err := strconv.ParseFloat(value, 64)

		return err == nil && within(field, number)
	case KindSize:
		match := sizeValue.FindStringSubmatch(value)
		if match == nil {
			return false
		}
		number, _ := strconv.ParseFloat(match[1], 64)

		return within(field, number*sizeFactors[match[2]])
	case KindBytes:
		number, ok := BytesOf(value)

		return ok && within(field, number)
	case KindChoice:
		return slices.Contains(field.Options, value)
	case KindFlags:
		return value == "" || !slices.ContainsFunc(strings.Split(value, ","), func(flag string) bool { return !slices.Contains(field.Options, flag) })
	}

	return field.Pattern != nil && regexp.MustCompile("^(?:"+*field.Pattern+")$").MatchString(value)
}

func within(field dtos.ConfigField, number float64) bool {
	return (field.Min == nil || number >= *field.Min) && (field.Max == nil || number <= *field.Max)
}

func (c Catalog) field(key string) (dtos.ConfigField, bool) {
	index := slices.IndexFunc(c.Fields, func(field dtos.ConfigField) bool { return field.Key == key })
	if index < 0 {
		return dtos.ConfigField{}, false
	}

	return c.Fields[index], true
}

func (c Catalog) Check(settings map[string]string) error {
	for _, key := range slices.Sorted(maps.Keys(settings)) {
		field, known := c.field(key)
		if !known {
			return command.Invalid(fmt.Sprintf("setting %q is not offered; config fields lists the settings", key))
		}
		if !allows(field, settings[key]) {
			return command.Invalid(fmt.Sprintf("setting %q does not accept %q", key, settings[key]))
		}
	}

	return nil
}

func Effective(tuned map[string]string, user map[string]string) map[string]string {
	effective := maps.Clone(tuned)
	if effective == nil {
		effective = map[string]string{}
	}
	maps.Copy(effective, user)

	return effective
}

func (c Catalog) Tuned(path string, resources dtos.Member) (map[string]string, error) {
	saved := map[string]string{}
	if data, err := os.ReadFile(path); err == nil {
		return saved, json.Unmarshal(data, &saved)
	}

	return c.Retune(path, resources)
}

func (c Catalog) Retune(path string, resources dtos.Member) (map[string]string, error) {
	tuned := c.Tune(resources)
	encoded, err := json.Marshal(tuned)
	if err != nil {
		return nil, err
	}

	return tuned, os.WriteFile(path, encoded, 0o600)
}

func (c Catalog) FieldsWith(tuned map[string]string) []dtos.ConfigField {
	fields := slices.Clone(c.Fields)
	for index, field := range fields {
		if value, found := tuned[field.Key]; found {
			fields[index].Default = value
		}
	}

	return fields
}

func (c Catalog) SourceOf(key string, tuned map[string]string, user map[string]string) string {
	if _, set := user[key]; set {
		return SourceUser
	}
	if _, set := tuned[key]; set {
		return SourceTuned
	}

	return SourceDefault
}

func (c Catalog) RestartChanged(previous map[string]string, tuned map[string]string, user map[string]string) []string {
	changed := []string{}
	for _, field := range c.Fields {
		_, overridden := user[field.Key]
		if field.Restart && !overridden && previous[field.Key] != tuned[field.Key] {
			changed = append(changed, field.Key)
		}
	}

	return changed
}
