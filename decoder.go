package orchestrator

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/PaesslerAG/jsonpath"
	"github.com/RussellLuo/structool"
)

type Decoder struct {
	data  map[string]any
	codec *structool.Codec
}

func NewDecoder() *Decoder {
	d := &Decoder{
		data: make(map[string]any),
	}
	d.codec = structool.New().TagName("json").DecodeHook(
		structool.DecodeStringToDuration,
		d.renderJSONPath,
	)
	return d
}

func (d *Decoder) AddInput(taskName string, input map[string]any) {
	d.addIO(taskName, "input", input)
}

func (d *Decoder) AddOutput(taskName string, output map[string]any) {
	d.addIO(taskName, "output", output)
}

func (d *Decoder) addIO(taskName string, typ string, value map[string]any) {
	taskIO, ok := d.data[taskName]
	if !ok {
		taskIO = make(map[string]any)
		d.data[taskName] = taskIO
	}
	io := taskIO.(map[string]any)
	io[typ] = value
}

func (d *Decoder) Decode(in any, out any) error {
	return d.codec.Decode(in, out)
}

func (d *Decoder) renderJSONPath(next structool.DecodeHookFunc) structool.DecodeHookFunc {
	reVar := regexp.MustCompile(`\${([^}]+)}`)

	return func(from, to reflect.Value) (any, error) {
		if from.Kind() != reflect.String {
			return next(from, to)
		}

		template := from.Interface().(string)
		matches := reVar.FindStringSubmatch(template)

		switch len(matches) {
		case 0: // template contains no variable, return it as the result value.
			return next(from, to)

		case 1: // unreachable case
			return next(from, to)

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
			// template contains more than one variable, replace all the matched
			// substrings with the result value.
			var result any
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
}

func (d *Decoder) evaluate(s string) (any, error) {
	// Convert s to a valid JSON path.
	path := "$." + s
	return jsonpath.Get(path, d.data)
}

func NewConstructDecoder(r Registry) *structool.Codec {
	codec := structool.New().TagName("json")
	codec.DecodeHook(
		structool.DecodeStringToDuration,
		decodeDefinitionToTask(r, codec),
	)
	return codec
}

func decodeDefinitionToTask(r Registry, codec *structool.Codec) func(next structool.DecodeHookFunc) structool.DecodeHookFunc {
	c := structool.New().TagName("json").DecodeHook(
		structool.DecodeStringToDuration,
	)

	return func(next structool.DecodeHookFunc) structool.DecodeHookFunc {
		return func(from, to reflect.Value) (any, error) {
			if to.Type().String() != "orchestrator.Task" {
				return next(from, to)
			}

			var def *TaskDefinition
			if err := c.Decode(from.Interface(), &def); err != nil {
				return nil, err
			}

			task, err := r.Construct(codec, def)
			if err != nil {
				return nil, err
			}
			return task, nil
		}
	}
}
