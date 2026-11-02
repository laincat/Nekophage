package ui

import (
	"testing"

	"github.com/laincat/nekophage/internal/player"
	"github.com/laincat/nekophage/internal/shelf"
)

func TestShortClipPauseKeepsItsRealPosition(t *testing.T) {
	sh, err := shelf.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := &App{shelf: sh}
	meta := player.Meta{Source: "video_fixture", ItemID: "clip", EpisodeID: "ep1"}
	a.saveVideoProgress(meta, player.Progress{Time: 2.375, Duration: 10, Paused: true})
	reopened, err := shelf.Open(sh.Root())
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := reopened.VideoProgress(meta.Source, meta.ItemID, meta.EpisodeID)
	if !ok || entry.Position != 2.375 || videoFinished(entry.Position, entry.Duration) {
		t.Fatalf("a paused ten-second clip cannot resume: %+v (found=%v)", entry, ok)
	}
	// The credits boundary is the same one used when opening an episode.
	a.saveVideoProgress(meta, player.Progress{Time: 9.9, Duration: 10, Paused: true})
	if _, ok := sh.VideoProgress(meta.Source, meta.ItemID, meta.EpisodeID); ok {
		t.Fatal("a completed clip still resumes at the credits")
	}
	// Unknown duration never supplies evidence of completion.
	a.saveVideoProgress(meta, player.Progress{Time: 12.25, Paused: true})
	if got, ok := sh.VideoProgress(meta.Source, meta.ItemID, meta.EpisodeID); !ok || videoFinished(got.Position, got.Duration) {
		t.Fatalf("an unknown-duration stream lost its resume point: %+v", got)
	}
}
