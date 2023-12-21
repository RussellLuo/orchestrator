package orchestrator

import (
	"context"
	"encoding/json"
)

// Actor represents a long-running flow that is capable of interacting with
// the outside world through its inbox and outbox.
type Actor struct {
	inbox  chan map[string]any
	outbox chan Result
}

func NewActor(ctx context.Context, f func(ab *ActorBehavior)) *Actor {
	inbox := make(chan map[string]any)
	outbox := make(chan Result)

	ab := &ActorBehavior{
		ctx:    ctx,
		inbox:  inbox,
		outbox: outbox,
	}
	go f(ab)

	return &Actor{
		inbox:  inbox,
		outbox: outbox,
	}
}

func (a *Actor) Inbox() chan<- map[string]any {
	return a.inbox
}

func (a *Actor) Outbox() <-chan Result {
	return a.outbox
}

func (a *Actor) String() string {
	return "<Actor>"
}

func (a *Actor) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// ActorBehavior is a helper for sending and receiving data to/from the outside
// world on behalf of an actor (i.e. within the context of task execution).
type ActorBehavior struct {
	ctx    context.Context
	inbox  <-chan map[string]any
	outbox chan<- Result
}

// Send sends data to the outside world through the outbox. If the internal
// context is done (cancelled or timed out), it will return immediately.
func (ab *ActorBehavior) Send(output Output, err error) {
	select {
	case ab.outbox <- Result{Output: output, Err: err}:
	case <-ab.ctx.Done():
	}
}

// Receive receives data from the outside world through the inbox. If the
// internal context is done (cancelled or timed out), it will return nil immediately.
func (ab *ActorBehavior) Receive() map[string]any {
	select {
	case input := <-ab.inbox:
		return input
	case <-ab.ctx.Done():
		// Return nil to indicate that the corresponding actor has been canceled.
		return nil
	}
}
