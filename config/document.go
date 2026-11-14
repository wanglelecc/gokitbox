package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
)

// Document is a single YAML document read from an explicit path. It has no
// process-global cache; a fresh ReadYAML sees fresh bytes. Decode never opens
// files, resolves environment variables, or constructs application resources.
type Document struct {
	path string
	data []byte
}

func ReadYAML(path string) (*Document, error) {
	if path == "" {
		return nil, errors.New("config: empty path")
	}
	full, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.New("config: cannot resolve path")
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", full, err)
	}
	if err := validateDocument(data); err != nil {
		return nil, fmt.Errorf("config: %s: %w", full, err)
	}
	return &Document{path: full, data: data}, nil
}

func (d *Document) Path() string { return d.path }

// Decode rejects unknown fields and duplicate keys. Error messages contain
// position and category only: parser source excerpts can contain credentials.
func (d *Document) Decode(out interface{}) error { return DecodeYAML(d.data, out) }

func DecodeYAML(data []byte, out interface{}) error {
	if reflect.ValueOf(out).Kind() != reflect.Ptr || reflect.ValueOf(out).IsNil() {
		return errors.New("config: destination must be a non-nil pointer")
	}
	if err := validateDocument(data); err != nil {
		return err
	}
	var raw interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return decodeError(err)
	}
	if err := checkTypes(raw, reflect.TypeOf(out).Elem()); err != nil {
		return err
	}
	if err := yaml.UnmarshalWithOptions(data, out, yaml.Strict()); err != nil {
		return decodeError(err)
	}
	return nil
}

func validateDocument(data []byte) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return decodeError(err)
	}
	if _, ok := value.(map[string]interface{}); !ok {
		return errors.New("config: expected a mapping document")
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return decodeError(err)
		}
		return errors.New("config: multiple YAML documents are not supported")
	}
	return nil
}

func decodeError(err error) error {
	category := "invalid YAML or field type"
	var unknown *yaml.UnknownFieldError
	var duplicate *yaml.DuplicateKeyError
	if errors.As(err, &unknown) {
		category = "unknown field"
	}
	if errors.As(err, &duplicate) {
		category = "duplicate key"
	}
	var detail yaml.Error
	if errors.As(err, &detail) && detail.GetToken() != nil && detail.GetToken().Position != nil {
		pos := detail.GetToken().Position
		return fmt.Errorf("config: %s at line %d, column %d", category, pos.Line, pos.Column)
	}
	return fmt.Errorf("config: %s", category)
}

// Install installs an already loaded configuration for legacy GetConf consumers.
// Call it once at application startup, before constructing any module singleton.
// It does not reconfigure clients that have already been created.
func Install(cfg Config) error {
	if cfg == nil {
		return errors.New("config: cannot install nil configuration")
	}
	gCfgMu.Lock()
	defer gCfgMu.Unlock()
	gCfg = cfg
	return nil
}

// NewSnapshot copies section data for the legacy interface. New code should
// receive typed Options rather than read this compatibility view.
func NewSnapshot(sections map[string]interface{}) (Config, error) {
	data, err := yaml.Marshal(sections)
	if err != nil {
		return nil, errors.New("config: cannot encode snapshot")
	}
	return parseYaml(data)
}

// YAML parsers often coerce numbers/bools into strings. Startup schemas must
// reject those mistakes; durations are the one deliberate string-to-int type.
func checkTypes(value interface{}, typ reflect.Type) error {
	if value == nil {
		return nil
	}
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() == reflect.Interface {
		return nil
	}
	if typ == reflect.TypeOf(time.Duration(0)) {
		if s, ok := value.(string); ok {
			if _, err := time.ParseDuration(s); err == nil {
				return nil
			}
		}
		return errors.New("config: duration requires a value with units")
	}
	mismatch := func() error { return errors.New("config: field type does not match schema") }
	switch typ.Kind() {
	case reflect.String:
		if _, ok := value.(string); !ok {
			return mismatch()
		}
	case reflect.Bool:
		if _, ok := value.(bool); !ok {
			return mismatch()
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch reflect.TypeOf(value).Kind() {
		case reflect.Int, reflect.Int64, reflect.Uint64:
		default:
			return mismatch()
		}
	case reflect.Map:
		m, ok := value.(map[string]interface{})
		if !ok {
			return mismatch()
		}
		for _, v := range m {
			if err := checkTypes(v, typ.Elem()); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		list, ok := value.([]interface{})
		if !ok {
			return mismatch()
		}
		for _, v := range list {
			if err := checkTypes(v, typ.Elem()); err != nil {
				return err
			}
		}
	case reflect.Struct:
		m, ok := value.(map[string]interface{})
		if !ok {
			return mismatch()
		}
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if f.PkgPath != "" {
				continue
			}
			tag := strings.Split(f.Tag.Get("yaml"), ",")
			if tag[0] == "-" {
				continue
			}
			if len(tag) > 1 && tag[1] == "inline" {
				if err := checkTypes(m, f.Type); err != nil {
					return err
				}
				continue
			}
			key := tag[0]
			if key == "" {
				key = strings.ToLower(f.Name)
			}
			if v, exists := m[key]; exists {
				if v == nil {
					return errors.New("config: null field is not supported")
				}
				if err := checkTypes(v, f.Type); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
