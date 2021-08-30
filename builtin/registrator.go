package builtin

import (
	"github.com/RussellLuo/orchestrator"
)

// RegisterIn registers all the builtin tasks in the given orchestrator.
func RegisterIn(o *orchestrator.Orchestrator) {
	o.MustRegister(TypeSerial, func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) { return NewSerial(o, def) })
	o.MustRegister(TypeParallel, func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) { return NewParallel(o, def) })
	o.MustRegister(TypeFunc, NewFunc)
	o.MustRegister(TypeHTTP, NewHTTP)
	o.MustRegister(TypeDecision, func(def *orchestrator.TaskDefinition) (orchestrator.Task, error) { return NewDecision(o, def) })
	o.MustRegister(TypeTerminate, NewTerminate)
}
