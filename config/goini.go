package config

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/unknwon/goconfig"
)

// ini struct
// implement the Config interface
type IniFile struct {
	*goconfig.ConfigFile
}

func (this *IniFile) Set(section, key string, value interface{}) {
	//not support
}

// load function
func loadIniFile(path string) (cfg Config, err error) {
	//load file
	file, err := goconfig.LoadConfigFile(path)
	config := new(IniFile)
	config.ConfigFile = file
	return config, err
}

// GetSectionObject implemented
// obj must a pointer
func (ini *IniFile) GetSectionObject(section string, obj interface{}) error {
	if ret, err := ini.GetSection(section); err != nil {
		log.Printf("cofnig error: %v", err)
		return err
	} else {
		//reflect
		typ := reflect.TypeOf(obj)
		val := reflect.ValueOf(obj)
		if typ != nil && typ.Kind() == reflect.Ptr && !val.IsNil() && val.Elem().Kind() == reflect.Struct {
			val = val.Elem()
		} else {
			return errors.New("cannot map to non-pointer struct")
		}
		return mapTo(ret, val)
	}
}

// convert map to object
func mapTo(info map[string]string, val reflect.Value) error {
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := val.Field(i)
		tpField := typ.Field(i)
		tag := tpField.Tag.Get("ini")
		if tag == "-" || !field.CanSet() {
			continue
		}
		if value, ok := info[tag]; ok {
			if err := setWithProperType(tpField.Type, value, field); err != nil {
				return fmt.Errorf("error mapping field(%s): %v", tag, err)
			}
		}
	}
	return nil
}

func setWithProperType(t reflect.Type, value string, field reflect.Value) error {
	if t == reflect.TypeOf(time.Duration(0)) {
		d, err := time.ParseDuration(value)
		if err != nil {
			return errors.New("invalid duration")
		}
		field.SetInt(int64(d))
		return nil
	}
	if t == reflect.TypeOf(time.Time{}) {
		location, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			return errors.New("cannot load timestamp location")
		}
		v, err := time.ParseInLocation("2006-01-02 15:04:05", value, location)
		if err != nil {
			return errors.New("invalid timestamp")
		}
		field.Set(reflect.ValueOf(v))
		return nil
	}
	switch t.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return errors.New("invalid boolean")
		}
		field.SetBool(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(value, 10, t.Bits())
		if err != nil {
			return errors.New("invalid integer")
		}
		field.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(value, 10, t.Bits())
		if err != nil {
			return errors.New("invalid unsigned integer")
		}
		field.SetUint(v)
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(value, t.Bits())
		if err != nil {
			return errors.New("invalid floating-point number")
		}
		field.SetFloat(v)
	case reflect.Slice:
		values := strings.Fields(value)
		slice := reflect.MakeSlice(t, len(values), len(values))
		for i, v := range values {
			if err := setWithProperType(t.Elem(), v, slice.Index(i)); err != nil {
				return err
			}
		}
		field.Set(slice)
	default:
		return fmt.Errorf("unsupported field type %s", t)
	}
	return nil
}
