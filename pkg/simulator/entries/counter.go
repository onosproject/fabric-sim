// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package entries

import (
	"github.com/onosproject/onos-lib-go/pkg/errors"
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
)

// Counter represents all cells of a specific counter
type Counter struct {
	info  *p4info.Counter
	cells []*p4api.CounterEntry
}

// Counters represents a set of P4 counters
type Counters struct {
	counters map[uint32]*Counter
}

// NewCounters creates a new counters store
func NewCounters(c []*p4info.Counter) *Counters {
	return &Counters{
		counters: make(map[uint32]*Counter),
	}
}

// NewCounter creates a new counter and all its cell entries
func NewCounter(info *p4info.Counter) *Counter {
	cells := make([]*p4api.CounterEntry, info.Size)
	for i := 0; i < int(info.Size); i++ {
		// TODO: properly setup the counter spec
		cells[i] = &p4api.CounterEntry{CounterId: info.Preamble.Id, Index: &p4api.Index{Index: int64(i)}}
	}
	return &Counter{
		info:  info,
		cells: cells,
	}
}

// ModifyCounterEntry modifies the specified counter entry cell
func (cs *Counters) ModifyCounterEntry(entry *p4api.CounterEntry, insert bool) error {
	if insert {
		return errors.NewInvalid("counter cannot be inserted")
	}

	counter, ok := cs.counters[entry.CounterId]
	if !ok {
		return errors.NewNotFound("counter not found")
	}
	if entry.Index == nil || entry.Index.Index < 0 || int(entry.Index.Index) >= len(counter.cells) {
		return errors.NewNotFound("counter index out of bounds")
	}

	counter.cells[entry.Index.Index] = entry
	return nil
}
