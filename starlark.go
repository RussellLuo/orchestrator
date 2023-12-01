package orchestrator

import (
	"errors"
	"fmt"
	"os"

	//"go.starlark.net/lib/json"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type MyDict struct {
	*starlark.Dict
}

func NewMyDict(size int) *MyDict {
	return &MyDict{Dict: starlark.NewDict(size)}
}

// Attr make MyDict keys can be read by a dot expression (y = x.f).
func (md *MyDict) Attr(name string) (starlark.Value, error) {
	// Fields located in the hashtable, if any, will hide the built-in dict methods.
	if v, found, _ := md.Dict.Get(starlark.String(name)); found {
		return v, nil
	}

	// NOTE: Here we do not check the error since Dict.Attr always returns nil error.
	if v, _ := md.Dict.Attr(name); v != nil {
		return v, nil
	}

	// Return None for non-existent fields.
	return starlark.None, nil
}

// SetField make MyDict keys can be written by a dot expression (x.f = y).
func (md *MyDict) SetField(name string, val starlark.Value) error {
	return md.Dict.SetKey(starlark.String(name), val)
}

// starlarkIterator implements starlark.Iterator and serves as a Starlark
// representation of Orchestrator's Iterator.
type starlarkIterator struct {
	iter *Iterator
}

func newStarlarkIterator(iter *Iterator) *starlarkIterator {
	return &starlarkIterator{iter: iter}
}

func (si *starlarkIterator) String() string        { return "orchestrator.Iterator" }
func (si *starlarkIterator) Type() string          { return si.String() }
func (si *starlarkIterator) Freeze()               {} // immutable
func (si *starlarkIterator) Truth() starlark.Bool  { return starlark.True }
func (si *starlarkIterator) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", si.Type()) }

func (si *starlarkIterator) Next(p *starlark.Value) bool {
	v, ok := <-si.iter.Next()
	if !ok {
		// The iterator is exhausted.
		return false
	}

	m := map[string]any{
		"name":   v.Name,
		"output": v.Output,
		"err":    v.Err.Error(),
	}
	sv, err := interfaceAsStarlarkValue(m)
	if err != nil {
		panic(err)
		//return false
	}

	*p = sv
	return true
}

func (si *starlarkIterator) Done() {}

func (si *starlarkIterator) Iterator() *Iterator {
	return si.iter
}

func StarlarkEvalExpr(s string, env map[string]any) (any, error) {
	expr, err := syntax.ParseExpr("", s, 0)
	if err != nil {
		return nil, err
	}

	envDict := make(starlark.StringDict, len(env))
	for k, v := range env {
		sv, err := interfaceAsStarlarkValue(v)
		if err != nil {
			return nil, err
		}
		envDict[k] = sv
	}
	// Add a pre-declared function `isiterator`.
	envDict["getenv"] = starlark.NewBuiltin("getenv", getEnv)
	envDict["isiterator"] = starlark.NewBuiltin("isiterator", isIterator)
	envDict["jsonencode"] = starlark.NewBuiltin("jsonencode", encode)
	envDict["jsondecode"] = starlark.NewBuiltin("jsondecode", decode)

	value, err := starlark.EvalExprOptions(&syntax.FileOptions{}, &starlark.Thread{}, expr, envDict)
	if err != nil {
		return nil, err
	}

	return starlarkValueAsInterface(value)
}

func StarlarkCallFunc(s string, env map[string]any) (any, error) {
	thread := &starlark.Thread{}
	globals, err := starlark.ExecFile(thread, "", s, nil)
	if err != nil {
		return nil, err
	}

	// Retrieve a module global.
	f, ok := globals["_"]
	if !ok {
		return nil, fmt.Errorf(`found no func named "_"`)
	}

	envValue, err := interfaceAsStarlarkValue(env)
	if err != nil {
		return nil, err
	}

	// Call Starlark function from Go.
	value, err := starlark.Call(thread, f, starlark.Tuple{envValue}, nil)
	if err != nil {
		return nil, err
	}
	return starlarkValueAsInterface(value)
}

func getEnv(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "s", &s); err != nil {
		return nil, err
	}

	v := os.Getenv(s)
	return starlark.String(v), nil
}

func isIterator(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var v starlark.Value
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "v", &v); err != nil {
		return nil, err
	}

	_, ok := v.(*starlarkIterator)
	return starlark.Bool(ok), nil
}

var ErrStarlarkConversion = errors.New("failed to convert Starlark data type")

func starlarkValueAsInterface(value starlark.Value) (any, error) {
	switch v := value.(type) {
	case starlark.NoneType:
		return nil, nil

	case starlark.Bool:
		return bool(v), nil

	case starlark.Int:
		res, _ := v.Int64()
		return int(res), nil

	case starlark.Float:
		return float64(v), nil

	case starlark.String:
		return string(v), nil

	case *starlark.List:
		it := v.Iterate()
		defer it.Done()

		var listItem starlark.Value
		var result []any

		for it.Next(&listItem) {
			listItemInterfaced, err := starlarkValueAsInterface(listItem)
			if err != nil {
				return nil, err
			}

			result = append(result, listItemInterfaced)
		}

		return result, nil

	case *starlark.Dict:
		return starlarkValueAsMap(v)

	case *MyDict:
		return starlarkValueAsMap(v.Dict)

	case *starlarkIterator:
		return v.Iterator(), nil

	default:
		return nil, fmt.Errorf("%w: unsupported type %T", ErrStarlarkConversion, value)
	}
}

func interfaceAsStarlarkValue(value any) (starlark.Value, error) {
	// TODO: Use reflection instead of type assertion to cover all edge cases.

	switch v := value.(type) {
	case nil:
		return starlark.None, nil

	case bool:
		return starlark.Bool(v), nil

	case int:
		return starlark.MakeInt(v), nil

	case int64:
		return starlark.MakeInt64(v), nil

	case uint:
		return starlark.MakeUint(v), nil

	case uint64:
		return starlark.MakeUint64(v), nil

	case float32:
		return starlark.Float(v), nil

	case float64:
		return starlark.Float(v), nil

	case string:
		return starlark.String(v), nil

	case []any:
		return sliceAsStarlarkValue(v)

	case map[string]any:
		return mapAsStarlarkValue(v)

	case *Iterator:
		return newStarlarkIterator(v), nil

	default:
		var m map[string]any
		if err := DefaultCodec.Decode(v, &m); err == nil {
			return mapAsStarlarkValue(m)
		}

		var s []any
		if err := DefaultCodec.Decode(v, &s); err == nil {
			return sliceAsStarlarkValue(s)
		}

		return nil, fmt.Errorf("%w: unsupported type %T", ErrStarlarkConversion, value)
	}
}

func starlarkValueAsMap(dict *starlark.Dict) (any, error) {
	result := map[string]any{}

	for _, item := range dict.Items() {
		key := item[0]
		value := item[1]

		keyStr, ok := key.(starlark.String)
		if !ok {
			return nil, fmt.Errorf("%w: all dict keys should be strings", ErrStarlarkConversion)
		}

		valueInterfaced, err := starlarkValueAsInterface(value)
		if err != nil {
			return nil, err
		}

		result[string(keyStr)] = valueInterfaced
	}

	return result, nil
}

func mapAsStarlarkValue(m map[string]any) (*MyDict, error) {
	dict := NewMyDict(len(m))
	for key, value := range m {
		mapValueStarlarked, err := interfaceAsStarlarkValue(value)
		if err != nil {
			return nil, err
		}
		if err := dict.SetKey(starlark.String(key), mapValueStarlarked); err != nil {
			return nil, err
		}
	}
	return dict, nil
}

func sliceAsStarlarkValue(s []any) (*starlark.List, error) {
	list := starlark.NewList([]starlark.Value{})
	for _, item := range s {
		listValueStarlarked, err := interfaceAsStarlarkValue(item)
		if err != nil {
			return nil, err
		}

		if err := list.Append(listValueStarlarked); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrStarlarkConversion, err)
		}
	}
	return list, nil
}
