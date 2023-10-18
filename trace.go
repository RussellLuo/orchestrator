package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/RussellLuo/structool"
)

var traceEncoder = structool.New().TagName("json").EncodeHook(
	structool.EncodeTimeToString("2006-01-02T15:04:05.000000Z07:00"),
	structool.EncodeDurationToString,
	structool.EncodeErrorToString,
)

// Event is the individual component of a trace. It represents a single
// leaf task that is traced.
type Event struct {
	When time.Time `json:"when"`
	// Since the previous event in the trace.
	Elapsed time.Duration `json:"elapsed"`

	Name   string         `json:"name"`
	Output map[string]any `json:"output,omitempty"`
	Error  error          `json:"error,omitempty"`

	// Events hold the events of the child trace, if any.
	Events []Event `json:"events,omitempty"`
}

// Map converts an event to a map.
func (e Event) Map() (map[string]any, error) {
	out, err := traceEncoder.Encode(e)
	if err != nil {
		return nil, err
	}
	return out.(map[string]any), nil
}

func (e Event) MarshalJSON() ([]byte, error) {
	m, err := e.Map()
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

// Trace provides tracing for the execution of a composite task.
type Trace interface {
	// New creates a child trace of the current trace. A child trace
	// provides tracing for the execution of a composite sub-task.
	New(name string) Trace

	// Wrap wraps a task to return a new task, which will automatically
	// add the execution result as an event to the trace.
	Wrap(task Task) Task

	// AddEvent adds an event to the trace.
	AddEvent(name string, output map[string]any, err error)

	// Events return the events stored in the trace.
	Events() []Event
}

type contextKey struct{}

func ContextWithTrace(ctx context.Context, tr Trace) context.Context {
	return context.WithValue(ctx, contextKey{}, tr)
}

func TraceFromContext(ctx context.Context) Trace {
	if tr, ok := ctx.Value(contextKey{}).(Trace); ok {
		return tr
	}
	return nilTrace{}
}

func NewTrace(name string) Trace {
	return &trace{
		name:     name,
		start:    time.Now(),
		children: make(map[string]Trace),
	}
}

type trace struct {
	name  string
	start time.Time

	mu       sync.RWMutex
	children map[string]Trace
	events   []Event
}

func (tr *trace) New(name string) Trace {
	child := NewTrace(name)
	tr.mu.Lock()
	tr.children[name] = child
	tr.mu.Unlock()
	return child
}

func (tr *trace) Wrap(task Task) Task {
	return traceTask{Task: task}
}

func (tr *trace) AddEvent(name string, output map[string]any, err error) {
	when := time.Now()
	fmt.Printf("trace.name: %v, event.name: %v, child.trace: %#v\n", tr.name, name, tr.children[name])

	tr.mu.Lock()
	var events []Event
	if child, ok := tr.children[name]; ok {
		// The current event to add is associated with a child trace, whose
		// events should be attached to the event.
		//
		// Since currently we only record the output of the task, it's guaranteed
		// that the task has completed, which means that all events of its children
		// traces, if any, have already been populated.
		events = child.Events()
	}
	tr.events = append(tr.events, Event{
		When:    when,
		Elapsed: tr.delta(when),
		Name:    name,
		Output:  output,
		Error:   err,
		Events:  events,
	})
	tr.mu.Unlock()
}

func (tr *trace) Events() []Event {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.events
}

func (tr *trace) delta(t time.Time) time.Duration {
	if len(tr.events) == 0 {
		return t.Sub(tr.start)
	}
	prev := tr.events[len(tr.events)-1].When
	return t.Sub(prev)
}

type traceTask struct {
	Task
}

func (t traceTask) Execute(ctx context.Context, input Input) (Output, error) {
	trace := TraceFromContext(ctx)
	output, err := t.Task.Execute(ctx, input)
	//fmt.Printf("trace: %#v, output: %v, err: %v\n", trace, output, err)
	trace.AddEvent(t.Task.Name(), output, err)
	return output, err
}

type nilTrace struct{}

func (tr nilTrace) New(name string) Trace                                  { return tr }
func (tr nilTrace) Wrap(task Task) Task                                    { return task }
func (tr nilTrace) AddEvent(name string, output map[string]any, err error) {}
func (tr nilTrace) Events() []Event                                        { return nil }
