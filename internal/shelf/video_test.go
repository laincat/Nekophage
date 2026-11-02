package shelf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVideoProgressSurvivesReopenAndReplacesThePosition(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	first := VideoHistory{Source: "probe", ItemID: "show", Title: "Show", LineName: "主线",
		EpisodeID: "ep1", EpisodeName: "第一集", Position: 30, Duration: 600}
	if err := s.RecordVideo(first); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordVideo(VideoHistory{Source: "probe", ItemID: "show", EpisodeID: "ep2",
		EpisodeName: "第二集", Position: 5, Duration: 600}); err != nil {
		t.Fatal(err)
	}
	// Watching the first episode again replaces its position instead of
	// adding a second record for the same episode.
	if err := s.RecordVideo(VideoHistory{Source: "probe", ItemID: "show", EpisodeID: "ep1",
		LineName: "备用", Position: 120, Duration: 600}); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	list := reopened.VideoHistory(0)
	if len(list) != 2 {
		t.Fatalf("video history = %+v", list)
	}
	if list[0].EpisodeID != "ep1" || list[0].Position != 120 || list[0].LineName != "备用" {
		t.Fatalf("the re-watched episode did not replace its record: %+v", list[0])
	}
	if list[1].EpisodeID != "ep2" {
		t.Fatalf("newest first order = %+v", list)
	}
	got, ok := reopened.VideoProgress("probe", "show", "ep1")
	if !ok || got.Position != 120 || got.EpisodeName != "" {
		t.Fatalf("episode progress = %+v, %v", got, ok)
	}
	if _, ok := reopened.VideoProgress("probe", "show", "ep9"); ok {
		t.Fatal("an unwatched episode reported a position")
	}
	// Video and comic history are separate: the same key is an ordinary
	// comic source name too.
	if err := reopened.Record(History{Source: "probe", ID: "show", Chapter: "第一章"}); err != nil {
		t.Fatal(err)
	}
	if hist, favs := reopened.Counts(); hist != 1 || favs != 0 {
		t.Fatalf("comic counts = %d/%d", hist, favs)
	}
	if len(reopened.VideoHistory(0)) != 2 {
		t.Fatal("recording a comic changed video history")
	}
	if err := reopened.ForgetVideo("probe", "show", "ep1"); err != nil {
		t.Fatal(err)
	}
	if list := reopened.VideoHistory(0); len(list) != 1 || list[0].EpisodeID != "ep2" {
		t.Fatalf("after forgetting = %+v", list)
	}
	raw, err := os.ReadFile(filepath.Join(root, "shelf.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"videos"`) || strings.Contains(string(raw), `"videos": null`) {
		t.Fatalf("shelf file = %s", raw)
	}
	if err := reopened.RecordVideo(VideoHistory{ItemID: "no-source"}); err == nil {
		t.Fatal("a video record without a source was accepted")
	}
}
