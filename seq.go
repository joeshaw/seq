// Package seq provides a means for receiving values out-of-order and
// producing them in a sequence.
package seq

import (
	"fmt"
	"sort"
)

// The Sequenced interface represents a piece of data belonging within a
// sequence.
type Sequenced interface {
	// Pos represents the current position in the sequence.
	Pos() int64
}

// SequencedSlice is a slice of Sequenced instances, implementing the
// sort.Interface interface.
type SequencedSlice []Sequenced

// Search finds the provided Sequenced instance in a sorted
// SequencedSlice, using the sort.Search function.  If found, it
// returns the current position of the item and true.  If not found,
// it returns the position where this item would be inserted into the
// sorted slice, and false.
func (ss SequencedSlice) Search(seq Sequenced) (int, bool) {
	pos := seq.Pos()

	i := sort.Search(len(ss), func(i int) bool {
		return ss[i].Pos() >= pos
	})

	exists := (i < len(ss) && ss[i].Pos() == pos)
	return i, exists
}

// A Sequencer receives Sequenced instances out of order, and produces
// them on its channel (C) in order.
type Sequencer struct {
	// C is a channel which produces Sequenced items in order as
	// they become available.
	C chan Sequenced

	// NextPos indicates what the next expected value is for a
	// Sequenced Pos() call.  This defaults to 0, but may be set
	// by callers prior to calling Add().
	NextPos int64

	queue SequencedSlice
	done  bool
}

// NewSequencer creates a new Sequencer, initializing the C channel.
func NewSequencer() *Sequencer {
	var s Sequencer
	s.C = make(chan Sequenced)
	return &s
}

func (s *Sequencer) insert(i int, seq Sequenced) {
	s.queue = append(s.queue, nil)
	copy(s.queue[i+1:], s.queue[i:])
	s.queue[i] = seq
}

func (s *Sequencer) send(seq Sequenced) {
	s.C <- seq
	s.NextPos++
}

func (s *Sequencer) drain() {
	for len(s.queue) > 0 {
		seq := s.queue[0]

		pos := seq.Pos()
		if pos != s.NextPos {
			return
		}

		s.send(seq)
		s.queue = s.queue[1:]
	}
}

// The Add function adds a Sequenced instance to the Sequencer.  Items
// can be added in any order.
//
// If an added item's Pos() is equal to NextPos, it is immediately
// sent on the Sequencer's C channel.  Queued items that follow are
// also sent.
//
// If an added item's Pos() is greater than NextPos, it is queued up
// to be sent as soon as it can.
//
// If an added item's Pos() is lower than NextPos -- indicating that
// it is a repeated value -- it is discarded.
//
// Add panics if Done() has previously been called on it.
func (s *Sequencer) Add(seq Sequenced) {
	if s.done {
		panic("cannot add to a closed sequencer")
	}

	pos := seq.Pos()

	switch {
	case pos < s.NextPos:
		// Ignore re-delivered messages with lower sequence
		// numbers than we're expecting

	case pos == s.NextPos:
		s.send(seq)
		s.drain()

	case pos > s.NextPos:
		i, exists := s.queue.Search(seq)
		if !exists {
			s.insert(i, seq)
		}
	}
}

// Done tells the Sequencer to close its channel.  Any attempt to Add
// following this call will panic.  This function will return an error
// if there are any items queued that could not be sent to the
// channel.
func (s *Sequencer) Done() error {
	close(s.C)
	s.done = true

	if len(s.queue) != 0 {
		return fmt.Errorf("Never got item at position %d", s.NextPos)
	}

	return nil
}

// QueueLen returns the current number of items blocked in the queue,
// waiting for the value at s.NextPos.
func (s *Sequencer) QueueLen() int {
	return len(s.queue)
}
