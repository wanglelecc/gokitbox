package config

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cast"
)

// YamlFile is the legacy section/key compatibility view. Access is synchronized;
// new application code should use Document.Decode and explicit module Options.
type YamlFile struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

func loadYamlFile(path string) (Config, error) {
	doc, err := ReadYAML(path)
	if err != nil {
		return nil, err
	}
	return parseYaml(doc.data)
}

func parseYaml(content []byte) (*YamlFile, error) {
	var data map[string]interface{}
	if err := DecodeYAML(content, &data); err != nil {
		return nil, err
	}
	return &YamlFile{data: data}, nil
}

func (f *YamlFile) Set(section, key string, value interface{}) {
	// Copy input so callers cannot mutate the view through slices/maps.
	b, err := yaml.Marshal(map[string]interface{}{key: value})
	if err != nil {
		return
	}
	var copied map[string]interface{}
	if err := yaml.Unmarshal(b, &copied); err != nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	sectionData, ok := f.data[section].(map[string]interface{})
	if !ok {
		sectionData = make(map[string]interface{})
		f.data[section] = sectionData
	}
	sectionData[key] = copied[key]
}

func (f *YamlFile) MustValue(section, key string, defaults ...string) string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if values, ok := f.data[section].(map[string]interface{}); ok {
		if val, exists := values[key]; exists && val != nil {
			return cast.ToString(val)
		}
	}
	if len(defaults) > 0 {
		return defaults[0]
	}
	return ""
}

func (f *YamlFile) MustValueArray(section, key, delim string) []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	values, ok := f.data[section].(map[string]interface{})
	if !ok {
		return nil
	}
	switch value := values[key].(type) {
	case []interface{}:
		result := make([]string, len(value))
		for i, v := range value {
			result[i] = cast.ToString(v)
		}
		return result
	case string:
		if value == "" {
			return nil
		}
		if delim == " " {
			return strings.Fields(value)
		}
		return strings.Split(value, delim)
	}
	return nil
}

func (f *YamlFile) GetKeyList(section string) []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	values, ok := f.data[section].(map[string]interface{})
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (f *YamlFile) GetSectionList() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	keys := make([]string, 0, len(f.data))
	for k := range f.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (f *YamlFile) GetSection(section string) (map[string]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	values, ok := f.data[section].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config: section %q is missing or not a mapping", section)
	}
	result := make(map[string]string, len(values))
	for k, v := range values {
		result[k] = cast.ToString(v)
	}
	return result, nil
}

func (f *YamlFile) GetSectionObject(section string, out interface{}) error {
	if reflect.ValueOf(out).Kind() != reflect.Ptr || reflect.ValueOf(out).IsNil() {
		return errors.New("config: destination must be a non-nil pointer")
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	var value interface{} = f.data
	if section != "" {
		var exists bool
		value, exists = f.data[section]
		if !exists || value == nil {
			return fmt.Errorf("config: section %q is missing", section)
		}
	}
	b, err := yaml.Marshal(value)
	if err != nil {
		return errors.New("config: cannot encode section")
	}
	// Legacy consumers may intentionally project only part of a section. Strict
	// application schemas use Document.Decode; keep projection compatibility here.
	if err := yaml.Unmarshal(b, out); err != nil {
		return decodeError(err)
	}
	return nil
}
