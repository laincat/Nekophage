package browser

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/laincat/veneracat-core/host"
)

// Closing the window is how a person says "I will not pass this". It has to end
// the wait at once and report the dismissal, not sit out the deadline and report
// something the user cannot act on.
func TestChallengeWindowCloseEndsTheWait(t *testing.T) {
	c := &challenge{
		ch:   &host.Challenge{URL: "https://example.test/challenge", Partition: "probe"},
		done: make(chan struct{}),
	}
	close(c.done)

	start := time.Now()
	clearance, err := c.watch(context.Background())
	if !errors.Is(err, host.ErrChallengeDismissed) {
		t.Fatalf("want ErrChallengeDismissed, got %v", err)
	}
	if clearance != nil {
		t.Fatalf("a dismissed challenge returned a clearance: %+v", clearance)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("the wait took %v after the window closed", elapsed)
	}
}

// The deadline is a different reason and must not be reported as a dismissal:
// one means the person gave up, the other means nobody was there.
func TestChallengeDeadlineIsNotADismissal(t *testing.T) {
	c := &challenge{
		ch:   &host.Challenge{URL: "https://example.test/challenge", Partition: "probe"},
		done: make(chan struct{}),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.watch(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
	if errors.Is(err, host.ErrChallengeDismissed) {
		t.Fatal("a deadline was reported as a dismissal")
	}
}
