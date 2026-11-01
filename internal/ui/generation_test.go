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

// Tickets are handed out one at a time and never repeat, and no reply can be
// newer than the generation it asked about.
//
// The check reads the counter once. Comparing `current(ticket)` and then
// `current(peek())` would test nothing: another goroutine can advance the
// generation between the two calls, so the second answer may be about a value
// that was already superseded.
func TestGenerationIsSafeForConcurrentUse(t *testing.T) {
	var g generation
	var mu sync.Mutex
	var wg sync.WaitGroup
	seen := make(map[int]bool, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticket := g.next()
			// A ticket handed out now is never newer than the counter read a
			// moment later, whatever the other goroutines did in between.
			if latest := g.peek(); ticket > latest {
				t.Errorf("ticket %d is newer than the generation %d", ticket, latest)
			}
			mu.Lock()
			seen[ticket] = true
			mu.Unlock()
		}()
	}
	wg.Wait()
	if len(seen) != 50 {
		t.Fatalf("50 calls produced %d distinct tickets", len(seen))
	}
	if g.n != 50 {
		t.Fatalf("generation = %d after 50 calls", g.n)
	}
}
