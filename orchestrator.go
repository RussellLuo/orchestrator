package orchestrator

import (
	"context"
	"fmt"
	"time"
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
	Execute(context.Context, *Decoder) (Output, error)
}

type Constructor func(*TaskDefinition) (Task, error)

type Orchestrator struct {
	constructors map[string]Constructor
}

func New() *Orchestrator {
	return &Orchestrator{constructors: make(map[string]Constructor)}
}

func (o *Orchestrator) Register(typ string, constructor Constructor) error {
	if _, ok := o.constructors[typ]; ok {
		return fmt.Errorf("constructor for task type %q is already registered", typ)
	}

	o.constructors[typ] = constructor
	return nil
}

// MustRegister is like Register but panics if there is an error.
func (o *Orchestrator) MustRegister(typ string, constructor Constructor) {
	if err := o.Register(typ, constructor); err != nil {
		panic(err)
	}
}

func (o *Orchestrator) Construct(def *TaskDefinition) (Task, error) {
	constructor, ok := o.constructors[def.Type]
	if !ok {
		return nil, fmt.Errorf("constructor for task type %q is not found", def.Type)
	}
	return constructor(def)
}
