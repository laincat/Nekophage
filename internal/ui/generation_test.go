package ui

import (
	"context"
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

func TestGenerationCancelsLateRequestsFromAnOldTicket(t *testing.T) {
	var g generation
	old := g.next()
	current := g.next()
	if err := g.requestContext(old).Err(); err != context.Canceled {
		t.Fatalf("late old request context = %v, want canceled", err)
	}
	if err := g.requestContext(current).Err(); err != nil {
		t.Fatalf("new request was canceled with the old one: %v", err)
	}
}

func TestLeavingSearchModeCancelsItsRequests(t *testing.T) {
	var app App
	ticket := app.searchGen.next()
	ctx := app.searchGen.requestContext(ticket)
	app.setMode(0)
	if err := ctx.Err(); err != nil {
		t.Fatalf("remaining in search mode canceled the request: %v", err)
	}
	app.setMode(2) // Shelf: no source or native window is needed to switch modes.
	if err := ctx.Err(); err != context.Canceled {
		t.Fatalf("leaving search mode kept the request alive: %v", err)
	}
}

func TestGenerationPageCompletionDoesNotCancelSibling(t *testing.T) {
	var g generation
	ticket := g.next()
	first, finishFirst := context.WithCancel(g.requestContext(ticket))
	second := g.requestContext(g.peek())
	finishFirst()
	if first.Err() != context.Canceled {
		t.Fatal("the first page did not finish")
	}
	if err := second.Err(); err != nil {
		t.Fatalf("finishing one page canceled the shared search: %v", err)
	}
	g.next()
	if err := second.Err(); err != context.Canceled {
		t.Fatalf("replacing the search left its page active: %v", err)
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
