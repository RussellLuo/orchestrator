package orchestrator

import (
	"fmt"
	"reflect"
	"regexp"
	"time"

	"github.com/PaesslerAG/jsonpath"
	"github.com/mitchellh/mapstructure"
)

var (
	reVar = regexp.MustCompile(`\${([^}]+)}`)
)

type Decoder struct {
	data        map[string]interface{}
	noRendering bool // Whether to disable rendering.
}

func NewDecoder() *Decoder {
	return &Decoder{data: make(map[string]interface{})}
}

func (d *Decoder) NoRendering() *Decoder {
	d.noRendering = true
	return d
}

func (d *Decoder) AddInput(taskName string, input map[string]interface{}) {
	d.addIO(taskName, "input", input)
}

func (d *Decoder) AddOutput(taskName string, output map[string]interface{}) {
	d.addIO(taskName, "output", output)
}

func (d *Decoder) addIO(taskName string, typ string, value map[string]interface{}) {
	taskIO, ok := d.data[taskName]
	if !ok {
		taskIO = make(map[string]interface{})
		d.data[taskName] = taskIO
	}
	io := taskIO.(map[string]interface{})
	io[typ] = value
}

func (d *Decoder) Decode(in interface{}, out interface{}) error {
	config := &mapstructure.DecoderConfig{
		DecodeHook: d.decodeHookFunc,
		TagName:    "orchestrator",
		Result:     out,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	return decoder.Decode(in)
}

func (d *Decoder) decodeHookFunc(from, to reflect.Value) (interface{}, error) {
	value := from.Interface()
	if from.Kind() != reflect.String {
		return value, nil
	}

	// string -> time.Duration
	if to.Type() == reflect.TypeOf(time.Duration(0)) {
		duration, err := time.ParseDuration(value.(string))
		if err != nil {
			return nil, err
		}
		return duration, nil
	}

	// Do not render the possible template string.
	if d.noRendering {
		return value, nil
	}

	// string -> evaluated value per JSONPath
	return d.render(value, to)
}

func (d *Decoder) render(value interface{}, to reflect.Value) (interface{}, error) {
	template := value.(string)
	matches := reVar.FindStringSubmatch(template)

	switch len(matches) {
	case 0: // template contains no variable, return it as the result value.
		return value, nil

	case 1: // unreachable case
		return value, nil

	case 2: // template contains only one variable.
		if matches[0] == template {
			// The variable is the whole string.

			result, err := d.evaluate(matches[1])
			if err != nil {
				return nil, err
			}

			if to.Kind() == reflect.String {
				// The target value is of type string, convert the result
				// value to be a string and return it.
				return fmt.Sprintf("%v", result), nil
			}

			// Return the raw result value.
			return result, nil
		}

		// The variable is just a substring of template, replace the substring
		// with the result value.
		fallthrough

	default:
		// template contains more than one variables, replace all the matched
		// substrings with the result value.
		var result interface{}
		var err error
		return reVar.ReplaceAllStringFunc(template, func(s string) string {
			part := s[len("${") : len(s)-len("}")]
			result, err = d.evaluate(part)
			if err != nil {
				return s
			}
			return fmt.Sprintf("%v", result)
		}), err
	}
}

func (d *Decoder) evaluate(s string) (interface{}, error) {
	// Convert s to a valid JSON path.
	path := "$." + s

	result, err := jsonpath.Get(path, d.data)
	if err != nil {
		return nil, err
	}

	return result, nil
}
