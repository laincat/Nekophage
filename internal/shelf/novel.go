package shelf

import "errors"

// Novel identifies a book within its source. Source and ID are opaque keys.
type Novel struct {
	Source string `json:"source"`
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author,omitempty"`
}

// NovelEntry keeps a bookmark and the latest reading position independently.
// Page is zero-based within the chapter identified by ChapterID.
type NovelEntry struct {
	Novel
	Kept      bool   `json:"kept,omitempty"`
	ChapterID string `json:"chapterId,omitempty"`
	Page      int    `json:"page,omitempty"`
}

func (s *Shelf) KeepNovel(book Novel, kept bool) error {
	return s.updateNovel(book, func(entry *NovelEntry) { entry.Kept = kept })
}

func (s *Shelf) RecordNovelProgress(book Novel, chapterID string, page int) error {
	if chapterID == "" || page < 0 {
		return errors.New("shelf: invalid novel reading position")
	}
	return s.updateNovel(book, func(entry *NovelEntry) {
		entry.ChapterID, entry.Page = chapterID, page
	})
}

// Novels returns a snapshot, most recently changed first.
func (s *Shelf) Novels() []NovelEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]NovelEntry(nil), s.novels...)
}

func (s *Shelf) updateNovel(book Novel, update func(*NovelEntry)) error {
	if book.Source == "" || book.ID == "" {
		return errors.New("shelf: novel needs a source and an id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := NovelEntry{Novel: book}
	previous := s.novels
	index := -1
	for i, saved := range previous {
		if saved.Source == book.Source && saved.ID == book.ID {
			index = i
			entry = saved
			entry.Novel = book
			break
		}
	}
	update(&entry)
	keep := entry.Kept || entry.ChapterID != ""
	if (index < 0 && !keep) || (index >= 0 && previous[index] == entry) {
		return nil
	}
	next := make([]NovelEntry, 0, len(previous)+1)
	if keep {
		next = append(next, entry)
	}
	for i, saved := range previous {
		if i != index {
			next = append(next, saved)
		}
	}
	s.novels = next
	if err := s.save(); err != nil {
		s.novels = previous
		return err
	}
	return nil
}
