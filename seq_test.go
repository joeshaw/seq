package seq

import (
	"math/rand"
	"sync"
	"testing"
)

type seqint int64

func (si seqint) Pos() int64 {
	return int64(si)
}

func run(t *testing.T, producer func(*Sequencer)) error {
	s := NewSequencer()

	var wg sync.WaitGroup
	wg.Add(2)

	var err error

	go func() {
		defer wg.Done()

		producer(s)
		err = s.Done()
	}()

	go func() {
		defer wg.Done()

		i := int64(0)
		for x := range s.C {
			if i != x.Pos() {
				t.Fatalf("Got %d, expected %d", x.Pos(), i)
			}
			i++
		}
	}()

	wg.Wait()

	return err
}

func assertInt(t *testing.T, actual, expected int) {
	if actual != expected {
		t.Fatal("%d does not equal expected value %d", actual, expected)
	}
}

func TestSequencer(t *testing.T) {
	// A simple out-of-order test
	err := run(t, func(s *Sequencer) {
		assertInt(t, s.QueueLen(), 0)
		s.Add(seqint(2)) // buffered
		assertInt(t, s.QueueLen(), 1)
		s.Add(seqint(1)) // buffered
		assertInt(t, s.QueueLen(), 2)
		s.Add(seqint(1)) // ignored
		assertInt(t, s.QueueLen(), 2)
		s.Add(seqint(0)) // 0 sent and 1 & 2 drained
		assertInt(t, s.QueueLen(), 0)
		s.Add(seqint(1)) // ignored
		assertInt(t, s.QueueLen(), 0)
		s.Add(seqint(3)) // sent
		assertInt(t, s.QueueLen(), 0)
	})
	if err != nil {
		t.Fatal(err)
	}

	// Larger test with random-ish sequence
	err = run(t, func(s *Sequencer) {
		// Intentionally don't seed the random number
		// generator so we get predictable runs.  It's enough
		// to know that these are not in order.
		for _, x := range rand.Perm(1000) {
			s.Add(seqint(x))
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	// Test where we never get the 0th element
	err = run(t, func(s *Sequencer) {
		s.Add(seqint(1))
	})
	if err == nil {
		t.Fatal("Expected an error since not all values were flushed")
	}

	// Test where we attempt to send after the sequencer is closed.
	var outerSeq *Sequencer
	err = run(t, func(s *Sequencer) {
		outerSeq = s
	})
	if err != nil {
		t.Fatal(err)
	}

	func() {
		defer func() {
			if p := recover(); p == nil {
				t.Fatal("Expected a panic but didn't get one")
			}
		}()

		outerSeq.Add(seqint(1))
	}()
}
