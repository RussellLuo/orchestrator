package orchestrator

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/PaesslerAG/jsonpath"
	"github.com/RussellLuo/structool"
)

var (
	reVar = regexp.MustCompile(`\${([^}]+)}`)

	defaultCodec = structool.New().TagName("json").DecodeHook(
		structool.DecodeStringToDuration,
	)
)

type Evaluator struct {
	data map[string]any
}

func NewEvaluator() *Evaluator {
	return &Evaluator{
		data: make(map[string]any),
	}
}

func (e *Evaluator) AddInput(taskName string, input map[string]any) {
	e.addIO(taskName, "input", input)
}

func (e *Evaluator) AddOutput(taskName string, output map[string]any) {
	e.addIO(taskName, "output", output)
}

func (e *Evaluator) addIO(taskName string, typ string, value map[string]any) {
	taskIO, ok := e.data[taskName]
	if !ok {
		taskIO = make(map[string]any)
		e.data[taskName] = taskIO
	}
	io := taskIO.(map[string]any)
	io[typ] = value
}

// Evaluate evaluates the expression s.
func (e *Evaluator) Evaluate(s string) (any, error) {
	matches := reVar.FindStringSubmatch(s)
	switch len(matches) {
	case 0: // expression s contains no variable, return it as the result value.
		return s, nil

	case 1: // unreachable case
		return s, nil

	case 2: // expression s contains only one variable.
		if matches[0] == s {
			// The variable is the whole string, return the raw result value.
			return e.evaluateVar(matches[1])
		}

		// The variable is just a substring of expression s, replace the substring
		// with the result value.
		fallthrough

	default:
		// expression s contains more than one variable, replace all the matched
		// substrings with the result value.
		var result any
		var err error
		return reVar.ReplaceAllStringFunc(s, func(s string) string {
			part := s[len("${") : len(s)-len("}")]
			result, err = e.evaluateVar(part)
			if err != nil {
				return s
			}
			return fmt.Sprintf("%v", result)
		}), err
	}
}

func (e *Evaluator) evaluateVar(s string) (any, error) {
	// Convert s to a valid JSON path.
	path := "$." + s
	return jsonpath.Get(path, e.data)
}

// Expr represents an expression.
type Expr[T any] struct {
	Expr  any
	Value T
}

func (e *Expr[T]) DecodeMapStructure(value any) error {
	e.Expr = value
	return nil
}

func (e *Expr[T]) Evaluate(input Input) error {
	out, err := Evaluate(e.Expr, input.Evaluate)
	if err != nil {
		return err
	}
	return defaultCodec.Decode(out, &e.Value)
}

// Evaluate traverses the value v and recursively evaluate every possible
// expression string. It will return a new copy of v in which every expression
// has been evaluated.
func Evaluate(v any, f func(string) (any, error)) (any, error) {
	if v == nil {
		return v, nil
	}

	value := reflect.ValueOf(v)
	typ := value.Type()

	switch value.Kind() {
	case reflect.Map:
		m := reflect.MakeMap(typ)
		for _, key := range value.MapKeys() {
			// Recursively evaluate the map value.
			//
			// NOTE: We assume that only map values contain expression variables.
			out, err := Evaluate(value.MapIndex(key).Interface(), f)
			if err != nil {
				return nil, err
			}
			m.SetMapIndex(key, reflect.ValueOf(out))
		}
		return m.Interface(), nil

	case reflect.Slice, reflect.Array:
		s := reflect.MakeSlice(typ, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			// Recursively evaluate the slice/array element.
			out, err := Evaluate(value.Index(i).Interface(), f)
			if err != nil {
				return nil, err
			}
			s = reflect.Append(s, reflect.ValueOf(out))
		}
		return s.Interface(), nil

	/*
		case reflect.Ptr:
			p := reflect.New(typ.Elem())
			// Recursively evaluate the value the pointer points to.
			out, err := Evaluate(value.Elem().Interface(), f)
			if err != nil {
				return nil, err
			}
			p.Elem().Set(reflect.ValueOf(out))
			return p.Interface(), nil
	*/

	case reflect.String:
		// Evaluate the possible expression string.
		return f(value.Interface().(string))

	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		// It's impossible for these basic types to hold an expression, so just return the value as is.
		return value.Interface(), nil

	default:
		return nil, fmt.Errorf("unsupported type %T", value)
	}
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
	return func(next structool.DecodeHookFunc) structool.DecodeHookFunc {
		return func(from, to reflect.Value) (any, error) {
			if to.Type().String() != "orchestrator.Task" {
				return next(from, to)
			}

			var def *TaskDefinition
			if err := defaultCodec.Decode(from.Interface(), &def); err != nil {
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
