package builtin

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/RussellLuo/orchestrator"
	"github.com/mitchellh/mapstructure"
)

// RegisterIn registers all the built-in tasks in the given orchestrator.
func RegisterIn(o *orchestrator.Orchestrator) {
	decoder := &inputDecoder{o: o}

	o.MustRegister(TypeSerial, func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
		s := &Serial{def: def}
		if err := decoder.Decode(def.InputTemplate, &s.Input); err != nil {
			return nil, err
		}
		return s, nil
	})

	o.MustRegister(TypeParallel, func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
		p := &Parallel{def: def}
		if err := decoder.Decode(def.InputTemplate, &p.Input); err != nil {
			return nil, err
		}
		return p, nil
	})

	o.MustRegister(TypeFunc, func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
		p := &Func{def: def}
		if err := decoder.Decode(def.InputTemplate, &p.Input); err != nil {
			return nil, err
		}
		return p, nil
	})

	o.MustRegister(TypeHTTP, func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
		h := &HTTP{
			def:    def,
			client: &http.Client{Timeout: def.Timeout},
		}
		if err := decoder.Decode(def.InputTemplate, &h.Input); err != nil {
			return nil, err
		}

		h.Encoding(h.Input.Encoding)

		return h, nil
	})

	o.MustRegister(TypeDecision, func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
		p := &Decision{def: def}
		if err := decoder.Decode(def.InputTemplate, &p.Input); err != nil {
			return nil, err
		}
		return p, nil
	})

	o.MustRegister(TypeTerminate, func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
		p := &Terminate{def: def}
		if err := decoder.Decode(def.InputTemplate, &p.Input); err != nil {
			return nil, err
		}
		return p, nil
	})
}

type inputDecoder struct {
	o *orchestrator.Orchestrator
}

func (d *inputDecoder) Decode(in interface{}, out interface{}) error {
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

func (d *inputDecoder) decodeHookFunc(from, to reflect.Value) (interface{}, error) {
	value := from.Interface()

	switch v := value.(type) {
	case *orchestrator.TaskDefinition:
		task, err := d.o.Construct(v)
		if err != nil {
			return nil, err
		}
		return task, nil

	case []*orchestrator.TaskDefinition:
		var tasks []orchestrator.Task

		names := make(map[string]bool) // Detect duplicate task name.
		for _, def := range v {
			if _, ok := names[def.Name]; ok {
				return nil, fmt.Errorf("duplicate task name %q", def.Name)
			}
			names[def.Name] = true

			task, err := d.o.Construct(def)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, task)
		}

		return tasks, nil

	case map[interface{}]*orchestrator.TaskDefinition:
		tasks := make(map[interface{}]orchestrator.Task)

		names := make(map[string]bool) // Detect duplicate task name.
		for key, def := range v {
			if _, ok := names[def.Name]; ok {
				return nil, fmt.Errorf("duplicate task name %q", def.Name)
			}
			names[def.Name] = true

			task, err := d.o.Construct(def)
			if err != nil {
				return nil, err
			}
			tasks[key] = task
		}

		return tasks, nil
	}

	return value, nil
}
