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

func TestLegacySubtitleSurvivesFinishedEpisode(t *testing.T) {
	root := t.TempDir()
	legacy := `{"videos":[{"source":"probe","itemId":"show","episodeId":"ep1","subtitle":{"name":"English","language":"en"}}]}`
	if err := os.WriteFile(filepath.Join(root, "shelf.json"), []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ForgetVideo("probe", "show", "ep1"); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	choice, ok := reopened.VideoSubtitle("probe", "show", "ep1")
	if !ok || choice.Name != "English" || choice.Language != "en" {
		t.Fatalf("legacy choice after finished episode: %+v, %v", choice, ok)
	}
}

func TestVideoSubtitleSurvivesProgressAndReopen(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RecordVideo(VideoHistory{Source: "probe", ItemID: "show", EpisodeID: "ep1",
		Position: 30, Duration: 600}); err != nil {
		t.Fatal(err)
	}
	// A preference for an unknown episode is saved without creating history.
	if ok, err := s.SetVideoSubtitle("probe", "show", "ep9", &VideoSubtitle{Name: "English"}); err != nil || !ok {
		t.Fatalf("unknown episode = %v, %v", ok, err)
	}
	if ok, err := s.SetVideoSubtitle("probe", "show", "ep1",
		&VideoSubtitle{Name: "English", Language: "en"}); err != nil || !ok {
		t.Fatalf("save subtitle = %v, %v", ok, err)
	}
	// Progress arrives every few seconds and replaces the record; it must not
	// drop the choice that record already holds.
	if err := s.RecordVideo(VideoHistory{Source: "probe", ItemID: "show", EpisodeID: "ep1",
		Position: 90, Duration: 600}); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopened.VideoProgress("probe", "show", "ep1")
	if !ok || got.Position != 90 {
		t.Fatalf("progress after reopen = %+v, %v", got, ok)
	}
	choice, ok := reopened.VideoSubtitle("probe", "show", "ep1")
	if !ok || choice.Name != "English" || choice.Language != "en" || choice.Off {
		t.Fatalf("subtitle after progress and reopen = %+v", choice)
	}
	// An explicit "off" is a choice, and it is distinguishable from unset.
	if ok, err := reopened.SetVideoSubtitle("probe", "show", "ep1", &VideoSubtitle{Off: true}); err != nil || !ok {
		t.Fatalf("save off = %v, %v", ok, err)
	}
	again, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	off, ok := again.VideoSubtitle("probe", "show", "ep1")
	if !ok || !off.Off || off.Name != "" {
		t.Fatalf("off choice = %+v", off)
	}
	if err := again.ForgetVideo("probe", "show", "ep1"); err != nil {
		t.Fatal(err)
	}
	last, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if saved, ok := last.VideoSubtitle("probe", "show", "ep1"); !ok || !saved.Off {
		t.Fatalf("choice after removing progress = %+v, %v", saved, ok)
	}
	if saved, ok := last.VideoSubtitle("probe", "show", "ep9"); !ok || saved.Name != "English" {
		t.Fatalf("first-play choice = %+v, %v", saved, ok)
	}
	if history := again.VideoHistory(0); len(history) != 0 {
		t.Fatalf("a preference created history: %+v", history)
	}
}
