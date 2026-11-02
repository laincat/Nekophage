package ui

import (
	"context"
	"sync"
)

// generation hands out tickets for asynchronous work, so a slow reply cannot
// replace a screen the user has already moved past.
//
// It keeps its own lock on purpose: a reply checks its ticket while holding
// App.mu, and taking App.mu again there would deadlock the window thread.
// Callers take App.mu first and this lock second, never the other way round.
type generation struct {
	mu     sync.Mutex
	n      int
	ctx    context.Context
	cancel context.CancelFunc
}

// next starts a new generation and cancels requests belonging to its predecessor.
func (g *generation) next() int {
	g.mu.Lock()
	previous := g.cancel
	g.n++
	ticket := g.n
	g.ctx, g.cancel = nil, nil
	g.mu.Unlock()
	if previous != nil {
		previous()
	}
	return ticket
}

// requestContext belongs to the ticket, not one page. Each caller derives its own
// timeout from it, so finishing a page does not cancel another page's request.
// A goroutine scheduled after its ticket was replaced gets an already-cancelled
// context: it must not join the new search just because it started running late.
func (g *generation) requestContext(ticket int) context.Context {
	g.mu.Lock()
	defer g.mu.Unlock()
	if ticket != g.n {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}
	if g.ctx == nil {
		g.ctx, g.cancel = context.WithCancel(context.Background())
	}
	return g.ctx
}

// peek returns the current generation without starting a new one. Pagination
// joins the search it continues: bumping here would make a still-running page
// stale and silently drop it from the list.
func (g *generation) peek() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.n
}

// current reports whether ticket is still the newest generation.
func (g *generation) current(ticket int) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return ticket == g.n
}
