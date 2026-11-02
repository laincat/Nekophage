// Package shelf keeps the reader's own records: what was read recently and
// which comics were kept.
//
// It is the local counterpart of the official client's history.db and
// local_favorite.db, so a `.venera` backup has somewhere to go. Records are
// kept in one JSON file in the profile; a short history and a single default
// folder are enough to start, and folders can be added without changing the
// file's shape.
package shelf

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DefaultFolder is where a comic goes when the reader keeps it.
const DefaultFolder = "收藏"

// historyLimit caps what is remembered; older entries are dropped.
const historyLimit = 500

// Comic identifies one comic in one source.
type Comic struct {
	Source string `json:"source"`
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author,omitempty"`
	Cover  string `json:"cover,omitempty"`
}

// History is one reading position.
type History struct {
	Comic
	// Chapter is the chapter's title as the source gave it.
	Chapter string `json:"chapter,omitempty"`
	// ChapterIndex is its index in the chapter list when it was read.
	ChapterIndex int `json:"chapterIndex"`
	// Page is the last page seen, 0 when unknown.
	Page int `json:"page,omitempty"`
	// Percent is how far into the chapter reading stopped, 0-100.
	Percent int `json:"percent,omitempty"`
	// Time is when the chapter was last opened, in seconds.
	Time int64 `json:"time"`
}

// ImageFavourite is one page kept from a chapter. Like the official client's
// image_favorites, it is a reference: the page's own address is stored, and
// the image is fetched through the source when it is shown.
type ImageFavourite struct {
	Source string `json:"source"`
	// ComicID and ComicTitle identify the comic the page belongs to.
	ComicID    string `json:"comicId"`
	ComicTitle string `json:"comicTitle,omitempty"`
	// ChapterID, ChapterIndex and ChapterName place the page in its chapter.
	ChapterID    string `json:"chapterId"`
	ChapterIndex int    `json:"chapterIndex"`
	ChapterName  string `json:"chapterName,omitempty"`
	// Page is 1-based.
	Page int `json:"page"`
	// URL is the image's address in the source.
	URL  string `json:"url"`
	Time int64  `json:"time"`
}

// VideoHistory is one episode's playback position. It is kept apart from
// comic history: a video provider and a comic source can share a key, and one
// list cannot answer "where was I" for both.
type VideoHistory struct {
	Source    string `json:"source"`
	ItemID    string `json:"itemId"`
	Title     string `json:"title,omitempty"`
	LineID    string `json:"lineId,omitempty"`
	LineName  string `json:"lineName,omitempty"`
	EpisodeID string `json:"episodeId,omitempty"`
	// EpisodeName is the episode's title as the provider gave it.
	EpisodeName string `json:"episodeName,omitempty"`
	// Position and Duration are seconds; Position is where playback stopped.
	Position float64 `json:"position,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	// Subtitle is the track the viewer picked for this episode. An explicit
	// "off" is a choice too, so it is kept apart from "never chosen".
	Subtitle *VideoSubtitle `json:"subtitle,omitempty"`
	Time     int64          `json:"time"`
}

// VideoSubtitle is one viewer's subtitle selection for an episode. The track
// is identified by name and language rather than index, because a provider can
// reorder or refresh its subtitle groups between plays.
type VideoSubtitle struct {
	// Off records an explicit "关闭" choice.
	Off      bool   `json:"off,omitempty"`
	Name     string `json:"name,omitempty"`
	Language string `json:"language,omitempty"`
}

// Folder is a named list of comics.
type Folder struct {
	Name  string  `json:"name"`
	Items []Comic `json:"items"`
}

// Shelf is the profile's records. Safe for concurrent use.
type Shelf struct {
	mu       sync.Mutex
	root     string
	history  []History
	folders  []Folder
	pictures []ImageFavourite
	videos   []VideoHistory
}

type file struct {
	History  []History        `json:"history"`
	Folders  []Folder         `json:"folders"`
	Pictures []ImageFavourite `json:"pictures,omitempty"`
	Videos   []VideoHistory   `json:"videos,omitempty"`
}

// Open reads the profile's records; a profile without any starts empty.
func Open(root string) (*Shelf, error) {
	if root == "" {
		return nil, errors.New("shelf: root is required")
	}
	s := &Shelf{root: root}
	b, err := os.ReadFile(s.path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s, nil
		}
		return nil, err
	}
	var f file
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("shelf: %s: %w", s.path(), err)
	}
	s.history, s.folders, s.pictures = f.History, f.Folders, f.Pictures
	s.videos = f.Videos
	return s, nil
}

func (s *Shelf) path() string { return filepath.Join(s.root, "shelf.json") }

// Root is the directory the shelf keeps its file in.
func (s *Shelf) Root() string { return s.root }

// Record adds a reading position, moving it to the front. Recording a comic
// again replaces its entry: one comic appears once, at its latest chapter.
func (s *Shelf) Record(h History) error {
	if h.Source == "" || h.ID == "" {
		return errors.New("shelf: history needs a source and an id")
	}
	if h.Time == 0 {
		h.Time = time.Now().Unix()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]History, 0, len(s.history)+1)
	out = append(out, h)
	for _, e := range s.history {
		if e.Source == h.Source && e.ID == h.ID {
			continue
		}
		out = append(out, e)
	}
	if len(out) > historyLimit {
		out = out[:historyLimit]
	}
	s.history = out
	return s.save()
}

// UpdateProgress records how far the reader got in a comic already on the
// shelf. It keeps the entry's place in the list: scrolling is not reading a
// new chapter. A comic that is not on the shelf is not added.
func (s *Shelf) UpdateProgress(source, id, chapter string, chapterIndex, page, percent int) (bool, error) {
	if source == "" || id == "" {
		return false, errors.New("shelf: history needs a source and an id")
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	if page < 0 {
		page = 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.history {
		e := &s.history[i]
		if e.Source != source || e.ID != id {
			continue
		}
		e.Page = page
		e.Percent = percent
		if chapter != "" {
			e.Chapter = chapter
		}
		if chapterIndex >= 0 {
			e.ChapterIndex = chapterIndex
		}
		e.Time = time.Now().Unix()
		return true, s.save()
	}
	return false, nil
}

// Merge records a position from an import without losing a newer local one:
// an entry is written only when the comic is unknown here or when the
// imported position is more recent.
func (s *Shelf) Merge(h History) (bool, error) {
	if h.Source == "" || h.ID == "" {
		return false, errors.New("shelf: history needs a source and an id")
	}
	s.mu.Lock()
	for _, e := range s.history {
		if e.Source == h.Source && e.ID == h.ID && e.Time >= h.Time {
			s.mu.Unlock()
			return false, nil
		}
	}
	s.mu.Unlock()
	return true, s.Record(h)
}

// History returns the most recent entries, newest first.
func (s *Shelf) History(limit int) []History {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || limit > len(s.history) {
		limit = len(s.history)
	}
	return append([]History(nil), s.history[:limit]...)
}

// HistoryEntry returns the reading position of one comic in one source.
// The detail page uses it to mark the chapter that was read last.
func (s *Shelf) HistoryEntry(source, id string) (History, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.history {
		if e.Source == source && e.ID == id {
			return e, true
		}
	}
	return History{}, false
}

// ForgetHistory drops one comic's reading positions.
func (s *Shelf) ForgetHistory(source, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.history[:0:0]
	for _, e := range s.history {
		if e.Source == source && e.ID == id {
			continue
		}
		out = append(out, e)
	}
	s.history = out
	return s.save()
}

// Add keeps a comic in a folder, creating the folder. It reports whether the
// comic was new there.
func (s *Shelf) Add(folder string, c Comic) (bool, error) {
	if folder == "" {
		folder = DefaultFolder
	}
	if c.Source == "" || c.ID == "" {
		return false, errors.New("shelf: comic needs a source and an id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.folders {
		if s.folders[i].Name != folder {
			continue
		}
		for _, it := range s.folders[i].Items {
			if it.Source == c.Source && it.ID == c.ID {
				return false, nil
			}
		}
		s.folders[i].Items = append(s.folders[i].Items, c)
		return true, s.save()
	}
	s.folders = append(s.folders, Folder{Name: folder, Items: []Comic{c}})
	return true, s.save()
}

// Remove drops a comic from a folder.
func (s *Shelf) Remove(folder, source, id string) error {
	if folder == "" {
		folder = DefaultFolder
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.folders {
		if s.folders[i].Name != folder {
			continue
		}
		out := s.folders[i].Items[:0:0]
		for _, it := range s.folders[i].Items {
			if it.Source == source && it.ID == id {
				continue
			}
			out = append(out, it)
		}
		s.folders[i].Items = out
		return s.save()
	}
	return nil
}

// CreateFolder adds an empty folder. A folder that already exists is not an
// error: the caller asked for it to be there.
func (s *Shelf) CreateFolder(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("shelf: a folder needs a name")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range s.folders {
		if f.Name == name {
			return nil
		}
	}
	s.folders = append(s.folders, Folder{Name: name, Items: []Comic{}})
	return s.save()
}

// RenameFolder renames a folder, keeping its comics and its place.
func (s *Shelf) RenameFolder(old, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("shelf: a folder needs a name")
	}
	if old == name {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range s.folders {
		if f.Name == name {
			return fmt.Errorf("shelf: folder %q already exists", name)
		}
	}
	for i := range s.folders {
		if s.folders[i].Name == old {
			s.folders[i].Name = name
			return s.save()
		}
	}
	return fmt.Errorf("shelf: folder %q does not exist", old)
}

// DeleteFolder removes a folder and its comics from the shelf. The comics
// themselves stay in the user's sources; only this list forgets them.
func (s *Shelf) DeleteFolder(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.folders[:0:0]
	found := false
	for _, f := range s.folders {
		if f.Name == name {
			found = true
			continue
		}
		out = append(out, f)
	}
	if !found {
		return fmt.Errorf("shelf: folder %q does not exist", name)
	}
	s.folders = out
	return s.save()
}

// Move puts a comic in a folder, taking it out of the folder it was in. It
// reports whether anything changed.
func (s *Shelf) Move(c Comic, folder string) (bool, error) {
	if folder == "" {
		folder = DefaultFolder
	}
	if c.Source == "" || c.ID == "" {
		return false, errors.New("shelf: comic needs a source and an id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	from := ""
	for _, f := range s.folders {
		for _, it := range f.Items {
			if it.Source == c.Source && it.ID == c.ID {
				from = f.Name
			}
		}
	}
	if from == folder {
		return false, nil
	}
	for i := range s.folders {
		if from != "" && s.folders[i].Name == from {
			kept := s.folders[i].Items[:0:0]
			for _, it := range s.folders[i].Items {
				if it.Source == c.Source && it.ID == c.ID {
					continue
				}
				kept = append(kept, it)
			}
			s.folders[i].Items = kept
			continue
		}
		if s.folders[i].Name != folder {
			continue
		}
		for _, it := range s.folders[i].Items {
			if it.Source == c.Source && it.ID == c.ID {
				return false, nil
			}
		}
		s.folders[i].Items = append(s.folders[i].Items, c)
		return true, s.save()
	}
	s.folders = append(s.folders, Folder{Name: folder, Items: []Comic{c}})
	return true, s.save()
}

// Holds returns the folder holding a comic, if any.
func (s *Shelf) Holds(source, id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range s.folders {
		for _, it := range f.Items {
			if it.Source == source && it.ID == id {
				return f.Name, true
			}
		}
	}
	return "", false
}

// Folders returns a copy of every folder, in creation order.
func (s *Shelf) Folders() []Folder {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Folder, len(s.folders))
	for i, f := range s.folders {
		out[i] = Folder{Name: f.Name, Items: append([]Comic(nil), f.Items...)}
	}
	return out
}

// KeepPage stores one page. The same page kept twice replaces the first.
func (s *Shelf) KeepPage(p ImageFavourite) error {
	if p.Source == "" || p.ComicID == "" || p.URL == "" {
		return errors.New("shelf: a page needs a source, a comic and a url")
	}
	if p.Page < 0 {
		p.Page = 0
	}
	if p.Time == 0 {
		p.Time = time.Now().Unix()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range s.pictures {
		if it.Source != p.Source || it.ComicID != p.ComicID || it.URL != p.URL {
			continue
		}
		// The page is already here: an import must not push an older copy
		// over a newer one.
		if it.Time >= p.Time {
			return nil
		}
		break
	}
	out := make([]ImageFavourite, 0, len(s.pictures)+1)
	out = append(out, p)
	for _, it := range s.pictures {
		if it.Source == p.Source && it.ComicID == p.ComicID && it.URL == p.URL {
			continue
		}
		out = append(out, it)
	}
	s.pictures = out
	return s.save()
}

// Pages returns the kept pages, newest first.
func (s *Shelf) Pages(limit int) []ImageFavourite {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || limit > len(s.pictures) {
		limit = len(s.pictures)
	}
	return append([]ImageFavourite(nil), s.pictures[:limit]...)
}

// ForgetPage drops one kept page.
func (s *Shelf) ForgetPage(source, comicID, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.pictures[:0:0]
	for _, it := range s.pictures {
		if it.Source == source && it.ComicID == comicID && it.URL == url {
			continue
		}
		out = append(out, it)
	}
	s.pictures = out
	return s.save()
}

// RecordVideo keeps where playback stopped for one episode, moving it to the
// front. An episode is one record: watching it again replaces the position.
func (s *Shelf) RecordVideo(v VideoHistory) error {
	if v.Source == "" || v.ItemID == "" {
		return errors.New("shelf: video history needs a source and an item")
	}
	if v.Position < 0 {
		v.Position = 0
	}
	if v.Time == 0 {
		v.Time = time.Now().Unix()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// A progress report carries position only; the subtitle choice lives in the
	// same record, so replacing the record must not drop it.
	if v.Subtitle == nil {
		for _, e := range s.videos {
			if sameVideo(e, v) {
				v.Subtitle = e.Subtitle
				break
			}
		}
	}
	out := make([]VideoHistory, 0, len(s.videos)+1)
	out = append(out, v)
	for _, e := range s.videos {
		if sameVideo(e, v) {
			continue
		}
		out = append(out, e)
	}
	if len(out) > historyLimit {
		out = out[:historyLimit]
	}
	s.videos = out
	return s.save()
}

// SetVideoSubtitle remembers the subtitle track chosen for an episode. It
// updates an existing record and reports false for an episode the shelf does
// not know yet, so a preference never creates history on its own.
func (s *Shelf) SetVideoSubtitle(source, itemID, episodeID string, choice *VideoSubtitle) (bool, error) {
	if source == "" || itemID == "" {
		return false, errors.New("shelf: video history needs a source and an item")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.videos {
		e := &s.videos[i]
		if e.Source != source || e.ItemID != itemID || e.EpisodeID != episodeID {
			continue
		}
		if choice == nil {
			e.Subtitle = nil
		} else {
			copy := *choice
			e.Subtitle = &copy
		}
		return true, s.save()
	}
	return false, nil
}

// VideoProgress returns the saved position of one episode.
func (s *Shelf) VideoProgress(source, itemID, episodeID string) (VideoHistory, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.videos {
		if e.Source == source && e.ItemID == itemID && e.EpisodeID == episodeID {
			return e, true
		}
	}
	return VideoHistory{}, false
}

// VideoHistory returns the most recently watched episodes, newest first.
func (s *Shelf) VideoHistory(limit int) []VideoHistory {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || limit > len(s.videos) {
		limit = len(s.videos)
	}
	return append([]VideoHistory(nil), s.videos[:limit]...)
}

// ForgetVideo drops one episode's position.
func (s *Shelf) ForgetVideo(source, itemID, episodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.videos[:0:0]
	for _, e := range s.videos {
		if e.Source == source && e.ItemID == itemID && e.EpisodeID == episodeID {
			continue
		}
		out = append(out, e)
	}
	s.videos = out
	return s.save()
}

// sameVideo reports whether two records describe the same episode. The line
// is not part of the identity: a provider may move an episode to another CDN.
func sameVideo(a, b VideoHistory) bool {
	return a.Source == b.Source && a.ItemID == b.ItemID && a.EpisodeID == b.EpisodeID
}

// Counts is how much the shelf holds, for reports.
func (s *Shelf) Counts() (history, favourites int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range s.folders {
		favourites += len(f.Items)
	}
	return len(s.history), favourites
}

// save writes the whole file, so a crash leaves either the old or the new.
func (s *Shelf) save() error {
	// Empty lists are written as [] so the file reads the same way for a
	// profile that has never kept anything.
	history, folders := s.history, s.folders
	if history == nil {
		history = []History{}
	}
	if folders == nil {
		folders = []Folder{}
	}
	pictures := s.pictures
	if pictures == nil {
		pictures = []ImageFavourite{}
	}
	videos := s.videos
	if videos == nil {
		videos = []VideoHistory{}
	}
	b, err := json.MarshalIndent(file{History: history, Folders: folders, Pictures: pictures, Videos: videos}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return err
	}
	tmp := s.path() + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path())
}
