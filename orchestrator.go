package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/RussellLuo/structool"
)

type InputTemplate map[string]interface{}

type Input struct {
	Decoder *Decoder
}

func NewInput(input map[string]interface{}) Input {
	decoder := NewDecoder()
	decoder.AddInput("context", input)
	return Input{Decoder: decoder}
}

type Output map[string]interface{}

func (o Output) SetTerminated() {
	o["terminated"] = true
}

func (o Output) IsTerminated() bool {
	terminated, ok := o["terminated"].(bool)
	return ok && terminated
}

type TaskDefinition struct {
	Name          string        `json:"name" yaml:"name" orchestrator:"name"`
	Type          string        `json:"type" yaml:"type" orchestrator:"type"`
	Timeout       time.Duration `json:"timeout" yaml:"timeout" orchestrator:"timeout"`
	InputTemplate InputTemplate `json:"input" yaml:"input" orchestrator:"input"`
}

type Task interface {
	// Name returns the name of the task.
	Name() string

	// String returns a string representation of the task.
	String() string

	// Execute executes the task with the given input.
	Execute(context.Context, Input) (Output, error)
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

func (r Registry) ConstructFromJSON(decoder *structool.Codec, data []byte) (Task, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	codec := structool.New().TagName("orchestrator").DecodeHook(
		structool.DecodeStringToDuration,
	)
	var def *TaskDefinition
	if err := codec.Decode(m, &def); err != nil {
		return nil, err
	}

	return r.Construct(decoder, def)
}

func MustRegister(factory *TaskFactory) {
	GlobalRegistry.MustRegister(factory)
}

func Construct(decoder *structool.Codec, def *TaskDefinition) (Task, error) {
	return GlobalRegistry.Construct(decoder, def)
}

func ConstructFromJSON(decoder *structool.Codec, data []byte) (Task, error) {
	return GlobalRegistry.ConstructFromJSON(decoder, data)
}

var GlobalRegistry = Registry{}
