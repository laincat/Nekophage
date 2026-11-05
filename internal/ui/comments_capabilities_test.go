package ui

import (
	"context"
	"testing"
	"time"

	"github.com/laincat/nekophage/internal/core"
)

// Opening a thread must not wait on the source.
//
// Whether the source can post or vote is answered by loading and compiling its
// script, and that used to run inside the call that built the window: a slow
// source held the window shut, with `context.Background()` behind it so nothing
// could stop it. The question now runs beside the window.
func TestOpeningAThreadDoesNotWaitForTheSource(t *testing.T) {
	c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	// The `comic` getter spins for a measured while, which is the cheapest way to
	// make "answering this takes time" true without a network.
	script := `class SlowCaps extends ComicSource {
 name = "Slow capabilities"; key = "slow_caps"; version = "1.0.0"
 get comic() {
   const until = Date.now() + 700
   while (Date.now() < until) {}
   return { sendComment: true, likeComment: true }
 }
}`
	if _, err := c.Install(context.Background(), script, "file", ""); err != nil {
		t.Fatal(err)
	}

	a := &App{core: c}
	a.comments.key = "slow_caps"
	a.comments.open = true
	a.comments.lifetime, a.comments.cancel = context.WithCancel(context.Background())
	defer a.comments.cancel()

	start := time.Now()
	a.askCommentCapabilities()
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("asking whether a source can post took %v in the open path", elapsed)
	}

	// The answer still arrives, just not on the way in.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		send := a.comments.canSend
		a.mu.Unlock()
		if send {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the source can post, but the window never learned it")
}

// A source with no comment methods leaves both answers false rather than
// claiming an ability it does not have.
func TestASourceWithoutCommentMethodsCannotPost(t *testing.T) {
	c, err := core.Open(core.Options{DataDir: t.TempDir(), AppVersion: "1.6.3"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	script := `class NoComments extends ComicSource {
 name = "No comments"; key = "no_comments"; version = "1.0.0"
}`
	if _, err := c.Install(context.Background(), script, "file", ""); err != nil {
		t.Fatal(err)
	}
	a := &App{core: c}
	a.comments.key = "no_comments"
	a.comments.open = true
	a.comments.lifetime, a.comments.cancel = context.WithCancel(context.Background())
	defer a.comments.cancel()
	a.askCommentCapabilities()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		send, like := a.comments.canSend, a.comments.canLike
		a.mu.Unlock()
		if !send && !like {
			// Both stay false; give the goroutine a moment to have run at all.
			time.Sleep(100 * time.Millisecond)
			a.mu.Lock()
			defer a.mu.Unlock()
			if a.comments.canSend || a.comments.canLike {
				t.Fatal("a source without comment methods was said to support them")
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the question never settled")
}
