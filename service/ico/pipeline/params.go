package pipeline

import (
	// Standard library.
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Params represents a list of pipeline parameters, indexed under their unique
// name. Parameter values may contain a prefix, which is typically removed when
// unpacking to a destination structure.
type Params map[string]string

// Unpack stores the partially parsed parameter list in the destination structure.
func (p *Params) Unpack(dest interface{}) error {
	// Deference pointer value if needed.
	pv := reflect.ValueOf(dest)
	if pv.Kind() == reflect.Ptr {
		pv = pv.Elem()
	}

	if pv.Kind() != reflect.Struct {
		return fmt.Errorf("invalid destination type '%s', expected 'struct'", pv.Kind())
	}

	return p.populateStruct(&pv)
}

// Iterates over list of fields in struct and attempts to set corresponding
// parameter for each one in turn. Un-settable fields are skipped.
func (p *Params) populateStruct(s *reflect.Value) error {
	// Set each struct field in sequence.
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)

		// Skip fields that cannot be written to.
		if f.CanSet() == false {
			continue
		}

		val, err := p.getFieldValue(&f, s.Type().Field(i).Tag)
		if err != nil {
			return err
		}

		if err := p.populateField(&f, val); err != nil {
			return err
		}
	}

	return nil
}

// Sets stringly-typed value into field, converting if necessary.
func (p *Params) populateField(f *reflect.Value, val string) error {
	switch f.Kind() {
	case reflect.Struct:
		return p.populateStruct(f)
	case reflect.String:
		f.SetString(val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val != "" {
			v, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return fmt.Errorf("unable to convert '%s' to integer: %s", val, err)
			}

			f.SetInt(v)
		}
	case reflect.Float32, reflect.Float64:
		if val != "" {
			v, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return fmt.Errorf("unable to convert '%s' to float: %s", val, err)
			}

			f.SetFloat(v)
		}
	default:
		return fmt.Errorf("unable to set unhandled field type '%s'", f.Kind())
	}

	return nil
}

// Parses and returns parameter value corresponding to field, as defined by the
// field tags.
func (p *Params) getFieldValue(f *reflect.Value, t reflect.StructTag) (string, error) {
	// Get field tags and determine parameter to set.
	key := strings.SplitN(t.Get("key"), "=", 2)
	val, ok := (*p)[key[0]]

	// Check for and handle extended key prefix. The field is skipped if the key
	// prefix doesn't match the key value, otherwise the prefix is removed from
	// the value.
	if ok && len(key) == 2 {
		if strings.HasPrefix(val, key[1]) {
			val = strings.TrimLeft(val[len(key[1]):], ":")
		} else {
			return "", nil
		}
	}

	// Skip field without error if field has no corresponding parameter in list
	// and no pre-set default value, otherwise, set default value.
	if def := t.Get("default"); !ok && def == "" {
		return "", nil
	} else if !ok || val == "" {
		val = def
	} else {
		// Split value in fields and get correct value for index.
		var i int
		vf := strings.Split(val, ":")

		// Get index from field tags, if available.
		if idx := t.Get("index"); idx != "" {
			i, _ = strconv.Atoi(idx)
		}

		if len(vf) <= i {
			return "", fmt.Errorf("%s: non-existing index '%d'", key[0], i)
		}

		// Validate value if needed.
		if vr := t.Get("valid"); vr != "" {
			if ok, _ = regexp.MatchString(vr, vf[i]); !ok {
				return "", fmt.Errorf("%s: value '%s' does not match '%s'", key[0], vf[i], vr)
			}
		}

		val = vf[i]
	}

	return val, nil
}

// Parse slices the parameter string provided and returns a Params instance,
// allowing for processing on individual parameters. Returns an error if parsing
// fails for any reason.
func Parse(params string) (*Params, error) {
	// Return error on empty parameter list.
	if params == "" {
		return nil, fmt.Errorf("unable to parse empty parameter list")
	}

	p := make(Params)

	fields := strings.Split(params, ",")
	for _, r := range fields {
		o := strings.Split(r, "=")
		if len(o) < 2 {
			return nil, fmt.Errorf("unable to parse malformed parameter '%s'", r)
		}

		p[o[0]] = o[1]
	}

	return &p, nil
}
