package orchestrator

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/RussellLuo/structool"
	"github.com/antonmedv/expr"
)

var (
	// JSONPath expression (https://github.com/PaesslerAG/jsonpath):
	//
	//   ${...}
	//
	// Expr expression (https://github.com/antonmedv/expr):
	//
	//   #{...}
	reExpr = regexp.MustCompile(`(?:\$|#){[^}]+}`)

	// reVar is like reExpr but actually extracts the leading character and the variable.
	reVar = regexp.MustCompile(`^(\$|#){([^}]+)}$`)

	DefaultCodec = structool.New().TagName("json").DecodeHook(
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

func (e *Evaluator) Add(taskName string, value map[string]any) {
	e.data[taskName] = value
}

func (e *Evaluator) Get(taskName string) map[string]any {
	value, _ := e.data[taskName].(map[string]any)
	return value
}

// Evaluate evaluates the expression s.
func (e *Evaluator) Evaluate(s string) (any, error) {
	matches := reExpr.FindAllStringSubmatch(s, -1)
	switch len(matches) {
	case 0: // expression s contains no variable, return it as the result value.
		return s, nil

	case 1: // expression s contains only one variable.
		m := matches[0][0]
		if m == s {
			// The variable is the whole string.
			result, err := e.evaluate(m)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate '%s': %v", s, err)
			}
			// Return the raw result value.
			return result, nil
		}

		// The variable is just a substring of expression s, replace the substring
		// with the result value.
		fallthrough

	default:
		// expression s contains more than one variable, replace all the matched
		// substrings with the result value.
		var errors []string
		result := reExpr.ReplaceAllStringFunc(s, func(s string) string {
			result, err := e.evaluate(s)
			if err != nil {
				errors = append(errors, fmt.Sprintf("failed to evaluate '%s': %v", s, err))
				return s
			}
			return fmt.Sprintf("%v", result)
		})
		if len(errors) > 0 {
			return nil, fmt.Errorf(strings.Join(errors, "; "))
		}
		return result, nil
	}
}

// evaluate evaluates a single expression variable.
func (e *Evaluator) evaluate(s string) (any, error) {
	matches := reVar.FindStringSubmatch(s)
	if len(matches) != 3 {
		return nil, fmt.Errorf("bad expression: %s", s)
	}

	dialect, variable := matches[1], matches[2]
	switch dialect {
	case "$": // JSONPath
		return e.evaluateJSONPathVar(variable)
	case "#": // Expr
		return e.evaluateExprVar(variable)
	default:
		return nil, fmt.Errorf("bad expression: %s", s)
	}
}

func (e *Evaluator) evaluateJSONPathVar(s string) (any, error) {
	// Convert s to a valid JSON path.
	path := "$." + s
	if s == "*" {
		// A single asterisk means to get the root object.
		path = "$"
	}
	return jsonpath.Get(path, e.data)
}

func (e *Evaluator) evaluateExprVar(s string) (any, error) {
	env := map[string]any{
		"getenv": os.Getenv,
	}
	for k, v := range e.data {
		env[k] = v
	}

	program, err := expr.Compile(s, expr.Env(env))
	if err != nil {
		return nil, err
	}

	output, err := expr.Run(program, env)
	if err != nil {
		return nil, err
	}
	//fmt.Printf("raw expr: %s, evaluated to: %v\n", s, output)
	return output, nil
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
	return DefaultCodec.Decode(out, &e.Value)
}

// Evaluate will return a copy of v in which all expressions have been
// replaced by the return value of function f.
//
// To achieve this, it traverses the value v and recursively evaluate
// every possible expression (of type string).
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
		return nil, fmt.Errorf("unsupported type %T", value.Interface())
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
			if err := DefaultCodec.Decode(from.Interface(), &def); err != nil {
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
