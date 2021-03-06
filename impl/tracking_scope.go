package impl

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"strings"
)

type TrackingScope interface {
	eval.Scope

	// GetRead returns a map of all variables that has been read from this scope. The
	// map contains the last value read.
	GetRead() map[string]eval.Value
}

type trackingScope struct {
	tracked eval.Scope
	read map[string]eval.Value
}

func NewTrackingScope(tracked eval.Scope) TrackingScope {
	return &trackingScope{tracked, make(map[string]eval.Value, 13)}
}

func (t *trackingScope) Fork() eval.Scope {
	// Multi threaded use of TrackingScope is not permitted
	panic(`attempt to fork TrackingScope`)
}

func (t *trackingScope) Get(name string) (eval.Value, bool) {
	value, found := t.tracked.Get(name)

	key := name
	if strings.HasPrefix(name, `::`) {
		key = name[2:]
	}
	if found {
		// A Global variable that has a value is immutable. No need to track it
		if t.tracked.State(name) == eval.Global {
			delete(t.read, key)
		} else {
			t.read[key] = value
		}
	} else {
		t.read[key] = nil // explicit nil denotes "not found"
	}
	return value, found
}

func (t *trackingScope) GetRead() map[string]eval.Value {
	return t.read
}

func (t *trackingScope) RxGet(index int) (eval.Value, bool) {
	return t.tracked.RxGet(index)
}

func (t *trackingScope) RxSet(variables []string) {
	t.tracked.RxSet(variables)
}

func (t *trackingScope) Set(name string, value eval.Value) bool {
	return t.tracked.Set(name, value)
}

func (t *trackingScope) State(name string) eval.VariableState {
	return t.tracked.State(name)
}

func (t *trackingScope) WithLocalScope(producer eval.Producer) eval.Value {
	return t.tracked.WithLocalScope(producer)
}
