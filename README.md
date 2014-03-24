# seq #

`seq` is a Go package for sequencing items received out-of-order.

The `Sequenced` interface provides the position of an item in a
sequence.  Out of order items are queued until the next expected
position is received.  The queue is then drained as much as possible.

Items are added by calling `Add()` on the `Sequencer` type, and
are returned in-order via a channel, `C`, on the `Sequencer`.

## Use cases ##

Perhaps you are using a queuing system that does not guarantee FIFO.
If you attach a sequence number to your queue items, you can push them
into a `Sequencer` and range over its channel to get them in the order
you expect.

Perhaps you are writing a UDP file transfer tool and want to ensure
the packets you receive are saved in the right order.  Add a sequence
number, feed them into `Sequencer`, and get them in the right order.

## Note about memory usage ##

This package has a major limitation: **It has potentially unbounded
memory usage.**

Items are queued in memory, and there is no backpressure as items are
added.

The pathological case is one where the very first item is never
received, but other items are continuously added.

You may be able to deal with this limitation in your application by
proactively fetching missing items.

Possible ways to solve this in the library include:

* Allowing serialization of the queue to a larger data store, like
  disk.

* Add some backpressure to prevent items from being added to the
  queue once it reaches a certain size.

* Error once the queue reaches a certain size.

## Example ##

```go
package main

import (
	"fmt"

	"github.com/joeshaw/seq"
)

type Team struct {
	Seed int
	Name string
}

func (t Team) Pos() int64 {
	return int64(t.Seed)
}

func (t Team) String() string {
	return fmt.Sprintf("%2d. %s", t.Seed, t.Name)
}

func main() {
	marchMadness := []Team{
		{1, "Florida"},
		{16, "Albany"},

		{8, "Colorado"},
		{9, "Pittsburgh"},

		{5, "VCU"},
		{12, "Stephen F. Austin"},

		{4, "UCLA"},
		{13, "Tulsa"},

		{6, "The Ohio State University"},
		{11, "Dayton"},

		{3, "Syracuse"},
		{14, "Western Michigan"},

		{7, "New Mexico"},
		{10, "Stanford"},

		{2, "Kansas"},
		{15, "Eastern Kentucky"},
	}

	s := seq.NewSequencer()
	s.NextPos = 1
	go func() {
		for _, team := range marchMadness {
			s.Add(team)
		}
		if err := s.Done(); err != nil {
			fmt.Println("Missed a team!")
		}
	}()

	for team := range s.C {
		fmt.Println(team)
	}
}
```

Output:
```
 1. Florida
 2. Kansas
 3. Syracuse
 4. UCLA
 5. VCU
 6. The Ohio State University
 7. New Mexico
 8. Colorado
 9. Pittsburgh
10. Stanford
11. Dayton
12. Stephen F. Austin
13. Tulsa
14. Western Michigan
15. Eastern Kentucky
16. Albany
```
