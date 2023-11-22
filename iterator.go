package orchestrator

import (
	"context"
	"encoding/json"
)

type Result struct {
	Name   string
	Output Output
	Err    error
}

// Iterator represents a iterable object that is capable of returning its
// values one at a time, permitting it to be iterated over in a for-loop.
type Iterator struct {
	c chan Result
}

func NewIterator(ctx context.Context, f func(sender *IteratorSender)) *Iterator {
	c := make(chan Result)
	sender := NewIteratorSender(ctx, c)
	go f(sender)

	return &Iterator{c: c}
}

func (i *Iterator) Next() <-chan Result {
	return i.c
}

func (i *Iterator) String() string {
	return "<Iterator>"
}

func (i *Iterator) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

// IteratorSender is a helper for sending data to an iterator.
type IteratorSender struct {
	ctx context.Context
	ch  chan<- Result
}

func NewIteratorSender(ctx context.Context, ch chan<- Result) *IteratorSender {
	return &IteratorSender{
		ctx: ctx,
		ch:  ch,
	}
}

// Send sends data to the internal channel. If the internal context is done
// (cancelled or timed out), it will mark the continue flag (whether to continue
// sending) as false.
func (s *IteratorSender) Send(output Output, err error) (continue_ bool) {
	select {
	case s.ch <- Result{Output: output, Err: err}:
		return true
	case <-s.ctx.Done():
		return false
	}
}

// End ends the iteration by closing the internal channel.
func (s *IteratorSender) End() {
	close(s.ch)
}
