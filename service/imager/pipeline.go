package imager

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// A Pipeline represents all data required for converting an image from its original format to the
// desired result.
type Pipeline struct {
	Width   int64   `default:"0"`
	Height  int64   `default:"0"`
	Density float64 `default:"1"`
	Quality int64   `default:"75"`
	Fit     string  `default:"crop"`
}

func NewPipeline() (*Pipeline, error) {
	p := &Pipeline{}
	pt := reflect.ValueOf(p).Elem().Type()

	// Set default values from field tags.
	for i := 0; i < pt.NumField(); i++ {
		f := pt.Field(i)
		if err := p.SetString(f.Name, f.Tag.Get("default")); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (p *Pipeline) SetString(field, value string) error {
	pv := reflect.ValueOf(p).Elem()
	f := pv.FieldByName(strings.Title(field))
	if f.Kind() == reflect.Invalid {
		return fmt.Errorf("field with name '%s' not found", field)
	}

	switch f.Kind() {
	case reflect.Int64:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to convert value to concrete 'int64' type: %s", err)
		}

		f.SetInt(v)
	case reflect.Float64:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("unable to convert value to concrete 'float64' type: %s", err)
		}

		f.SetFloat(v)
	case reflect.String:
		f.SetString(value)
	default:
		return fmt.Errorf("field '%s' with type '%s' does not match supported types", field, f.Kind().String())
	}

	return nil
}
