package support

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var patterns sync.Map

func DecodeStrict[T any](r io.Reader) (T, error) {
	var value T
	data, err := io.ReadAll(r)
	if err != nil {
		return value, err
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return value, err
	}
	if _, isObject := raw.(map[string]any); !isObject {
		return value, errors.New("the request must be a JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, err
	}

	return value, check(reflect.TypeOf(value), raw, "")
}

func check(t reflect.Type, raw any, path string) error {
	object, _ := raw.(map[string]any)
	if err := rejectUnknownKeys(t, object, path); err != nil {
		return err
	}
	for i := range t.NumField() {
		field := t.Field(i)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		value, present := object[name]
		if field.Type.Kind() == reflect.Pointer && value == nil {
			continue
		}
		if !present || value == nil {
			return fmt.Errorf("%s%s: required", path, name)
		}
		if err := checkValue(field.Type, field.Tag, value, path+name); err != nil {
			return err
		}
	}

	return nil
}

func rejectUnknownKeys(t reflect.Type, object map[string]any, path string) error {
	known := map[string]bool{}
	for i := range t.NumField() {
		known[strings.Split(t.Field(i).Tag.Get("json"), ",")[0]] = true
	}
	for key := range object {
		if !known[key] {
			return fmt.Errorf("%s%s: unknown field", path, key)
		}
	}

	return nil
}

func checkValue(t reflect.Type, tag reflect.StructTag, value any, path string) error {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		if t.String() == "time.Time" {
			return nil
		}

		return check(t, value, path+".")
	case reflect.Slice:
		for i, item := range value.([]any) {
			if err := checkValue(t.Elem(), tag, item, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	case reflect.String:
		return checkText(t, tag, value.(string), path)
	case reflect.Int64:
		if minimum, ok := limit(tag, "min"); ok && int(value.(float64)) < minimum {
			return fmt.Errorf("%s: must be at least %d", path, minimum)
		}
	}

	return nil
}

func checkText(t reflect.Type, tag reflect.StructTag, text string, path string) error {
	if enum, isEnum := reflect.ValueOf(text).Convert(t).Interface().(interface{ Valid() bool }); isEnum && !enum.Valid() {
		return fmt.Errorf("%s: invalid value %q", path, text)
	}
	if maximum, ok := limit(tag, "max"); ok && len([]rune(text)) > maximum {
		return fmt.Errorf("%s: longer than %d", path, maximum)
	}
	if minimum, ok := limit(tag, "min"); ok && len([]rune(text)) < minimum {
		return fmt.Errorf("%s: shorter than %d", path, minimum)
	}
	if pattern := tag.Get("pattern"); pattern != "" && !compiled(pattern).MatchString(text) {
		return fmt.Errorf("%s: does not match %s", path, pattern)
	}

	return nil
}

func limit(tag reflect.StructTag, key string) (int, bool) {
	value, err := strconv.Atoi(tag.Get(key))

	return value, err == nil
}

func compiled(pattern string) *regexp.Regexp {
	if cached, found := patterns.Load(pattern); found {
		return cached.(*regexp.Regexp)
	}
	expression := regexp.MustCompile(pattern)
	patterns.Store(pattern, expression)

	return expression
}

func Encode(value any) ([]byte, error) {
	v := reflect.New(reflect.TypeOf(value)).Elem()
	v.Set(reflect.ValueOf(value))
	emptyNilLists(v)

	return json.Marshal(v.Interface())
}

func emptyNilLists(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			emptyNilLists(v.Elem())
		}
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Field(i).CanSet() {
				emptyNilLists(v.Field(i))
			}
		}
	case reflect.Slice:
		if v.IsNil() {
			v.Set(reflect.MakeSlice(v.Type(), 0, 0))
		}
		for i := range v.Len() {
			emptyNilLists(v.Index(i))
		}
	case reflect.Map:
		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
	}
}
