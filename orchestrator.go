package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/RussellLuo/structool"
)

type InputTemplate map[string]interface{}

type Output map[string]interface{}

func (o Output) SetTerminated() {
	o["terminated"] = true
}

func (o Output) IsTerminated() bool {
	terminated, ok := o["terminated"].(bool)
	return ok && terminated
}

type TaskDefinition struct {
	Name          string        `yaml:"name" orchestrator:"name"`
	Type          string        `yaml:"type" orchestrator:"type"`
	Timeout       time.Duration `yaml:"timeout" orchestrator:"timeout"`
	InputTemplate InputTemplate `yaml:"input" orchestrator:"input"`
}

type Task interface {
	Definition() *TaskDefinition
	InputString() string
	Execute(context.Context, *Decoder) (Output, error)
}

type TaskFactory struct {
	Type        string
	Constructor func(*structool.Codec, *TaskDefinition) (Task, error)
}

type Registry map[string]*TaskFactory

func (r Registry) Register(factory *TaskFactory) error {
	if _, ok := r[factory.Type]; ok {
		return fmt.Errorf("factory for task type %q is already registered", factory.Type)
	}

	r[factory.Type] = factory
	return nil
}

// MustRegister is like Register but panics if there is an error.
func (r Registry) MustRegister(factory *TaskFactory) {
	if err := r.Register(factory); err != nil {
		panic(err)
	}
}

func (r Registry) Construct(decoder *structool.Codec, def *TaskDefinition) (Task, error) {
	factory, ok := r[def.Type]
	if !ok {
		return nil, fmt.Errorf("factory for task type %q is not found", def.Type)
	}
	return factory.Constructor(decoder, def)
}

func MustRegister(factory *TaskFactory) {
	GlobalRegistry.MustRegister(factory)
}

func Construct(decoder *structool.Codec, def *TaskDefinition) (Task, error) {
	return GlobalRegistry.Construct(decoder, def)
}

var GlobalRegistry = Registry{}
