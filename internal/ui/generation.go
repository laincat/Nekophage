package ui

import "sync"

// generation hands out tickets for asynchronous work, so a slow reply cannot
// replace a screen the user has already moved past.
//
// It keeps its own lock on purpose: a reply checks its ticket while holding
// App.mu, and taking App.mu again there would deadlock the window thread.
// Callers take App.mu first and this lock second, never the other way round.
type generation struct {
	mu sync.Mutex
	n  int
}

// next starts a new generation and returns its ticket.
func (g *generation) next() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.n++
	return g.n
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
