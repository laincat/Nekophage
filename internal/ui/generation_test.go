package ui

import (
	"sync"
	"testing"
)

func TestGenerationAcceptsOnlyTheNewestTicket(t *testing.T) {
	var g generation
	first := g.next()
	if !g.current(first) {
		t.Fatal("the first ticket was not current")
	}
	second := g.next()
	if g.current(first) {
		t.Fatal("an older reply would still replace the newer screen")
	}
	if !g.current(second) {
		t.Fatal("the newest ticket was rejected")
	}
}

func TestGenerationPeekDoesNotStartOne(t *testing.T) {
	var g generation
	ticket := g.next()
	if got := g.peek(); got != ticket {
		t.Fatalf("peek = %d, want %d", got, ticket)
	}
	// A pagination request joining the current search must not make an
	// in-flight page stale, so peek must not advance the generation.
	if g.peek() != ticket {
		t.Fatal("peek advanced the generation")
	}
	if !g.current(ticket) {
		t.Fatal("peek made the current generation stale")
	}
}

func TestGenerationIsSafeForConcurrentUse(t *testing.T) {
	var g generation
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticket := g.next()
			if !g.current(ticket) && !g.current(g.peek()) {
				t.Errorf("ticket %d was neither current nor superseded", ticket)
			}
		}()
	}
	wg.Wait()
	if g.n != 50 {
		t.Fatalf("generation = %d after 50 calls", g.n)
	}
}
