package config

import (
	"fmt"
	"reflect"
	"strings"
)

func resolveConfigValueByPath(cfg *Config, parts []string) (reflect.Value, error) {
	current := reflect.ValueOf(cfg).Elem()
	for _, part := range parts {
		next, err := resolveConfigPathSegment(current, part)
		if err != nil {
			return reflect.Value{}, err
		}
		current = next
	}
	return current, nil
}

func resolveConfigParentForSet(cfg *Config, parts []string) (reflect.Value, string, error) {
	current := reflect.ValueOf(cfg).Elem()
	for _, part := range parts[:len(parts)-1] {
		next, err := resolveConfigPathSegment(current, part)
		if err != nil {
			return reflect.Value{}, "", err
		}
		current = next
	}
	return current, parts[len(parts)-1], nil
}

func resolveConfigPathSegment(current reflect.Value, part string) (reflect.Value, error) {
	if current.Kind() == reflect.Map {
		key := reflect.ValueOf(part)
		mapVal := current.MapIndex(key)
		if !mapVal.IsValid() {
			return reflect.Value{}, fmt.Errorf("key not found: %s", part)
		}
		return mapVal, nil
	}
	if current.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("not a struct: %s", part)
	}

	fieldVal, found := findStructFieldByYAMLTag(current, part)
	if !found {
		return reflect.Value{}, fmt.Errorf("field not found: %s", part)
	}
	return fieldVal, nil
}

func findStructFieldByYAMLTag(v reflect.Value, tagName string) (reflect.Value, bool) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if yamlTagName(field.Tag.Get("yaml")) == tagName {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

func yamlTagName(tag string) string {
	return strings.Split(tag, ",")[0]
}

func assignValueByPathPart(parent reflect.Value, part string, value interface{}) error {
	if parent.Kind() == reflect.Map {
		return assignMapValueByPathPart(parent, part, value)
	}
	if parent.Kind() != reflect.Struct {
		return fmt.Errorf("not a struct: %s", part)
	}

	fieldVal, found := findStructFieldByYAMLTag(parent, part)
	if !found {
		return fmt.Errorf("field not found: %s", part)
	}
	if !fieldVal.CanSet() {
		return fmt.Errorf("cannot set field: %s", part)
	}
	return assignConvertibleValue(fieldVal, value)
}

func assignMapValueByPathPart(parent reflect.Value, part string, value interface{}) error {
	if parent.IsNil() {
		if !parent.CanSet() {
			return fmt.Errorf("cannot initialize map for key: %s", part)
		}
		parent.Set(reflect.MakeMap(parent.Type()))
	}

	key := reflect.ValueOf(part)
	if !key.Type().ConvertibleTo(parent.Type().Key()) {
		return fmt.Errorf("map key type mismatch: expected %s, got %s", parent.Type().Key(), key.Type())
	}

	mapValue := reflect.ValueOf(value)
	if !mapValue.Type().ConvertibleTo(parent.Type().Elem()) {
		return fmt.Errorf("type mismatch: expected %s, got %s", parent.Type().Elem(), mapValue.Type())
	}

	parent.SetMapIndex(key.Convert(parent.Type().Key()), mapValue.Convert(parent.Type().Elem()))
	return nil
}

func assignConvertibleValue(dst reflect.Value, value interface{}) error {
	newVal := reflect.ValueOf(value)
	if !newVal.Type().ConvertibleTo(dst.Type()) {
		return fmt.Errorf("type mismatch: expected %s, got %s", dst.Type(), newVal.Type())
	}
	dst.Set(newVal.Convert(dst.Type()))
	return nil
}
