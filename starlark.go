package orchestrator

import (
	"errors"
	"fmt"

	"go.starlark.net/starlark"
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
	return md.Dict.Attr(name)
}

// SetField make MyDict keys can be written by a dot expression (x.f = y).
func (md *MyDict) SetField(name string, val starlark.Value) error {
	return md.Dict.SetKey(starlark.String(name), val)
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
