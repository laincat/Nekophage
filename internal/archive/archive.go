// Package archive reads official `.venera` backups, which Core parses.
//
// The official client overwrites the matching local files on import, with no
// preview and no way back. This reads the archive and builds a plan the user
// confirms: the sources it carries are installed through the app's own
// install flow, so each script is validated like any other, and source data
// is written key by key into that source's store.
package archive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/laincat/veneracat-core/backup"
)

// Backup is what one archive contains.
type Backup struct {
	Path    string
	Content *backup.Content
	// Plan restores what this app can apply today: sources and their data.
	Plan *backup.Plan
	// warnings are problems found while reading, shown with the summary.
	warnings []string
}

// Selection is what an import takes. Image favourites, settings and cookies
// are read but not applied yet, so they are not in it.
func Selection() backup.Selection {
	return backup.Selection{History: true, Favorites: true, Sources: true, SourceData: true}
}

// Read parses the archive at path. knownKeys are the sources already
// installed; they help resolve the type hashes in history and favourites.
func Read(ctx context.Context, path string, knownKeys []string) (*Backup, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	a, err := backup.Open(f, st.Size(), backup.Limits{})
	if err != nil {
		return nil, fmt.Errorf("archive: %w", err)
	}
	// Databases are extracted while Read runs, so the file stays open.
	content, err := a.Read(ctx, backup.Options{KnownKeys: knownKeys})
	if err != nil {
		return nil, fmt.Errorf("archive: %w", err)
	}
	plan, err := content.Plan(Selection())
	if err != nil {
		return nil, fmt.Errorf("archive: %w", err)
	}
	return &Backup{Path: path, Content: content, Plan: plan}, nil
}

// Summary is what the confirmation screen shows.
type Summary struct {
	Path string
	// Sources the archive carries scripts for, and how many carry data too.
	Sources, WithData int
	// History and favourites are applied; image favourites, settings and
	// cookies are read only.
	History, FavoriteFolders, FavoriteItems, ImageFavorites int
	SearchHistory, Settings                                 int
	// KeptPages is how many pages of the image favourites can be restored.
	KeptPages  int
	Unresolved int
	Warnings   []string
	// Importable is what this import will actually install.
	Importable int
	// RestoredData counts the sources whose data can be written.
	RestoredData int
}

// Summarize counts what an import would do.
func (b *Backup) Summarize() Summary {
	s := Summary{Path: b.Path, Warnings: append(append([]string(nil), b.Content.Warnings...), b.warnings...)}
	s.History = len(b.Content.History)
	s.FavoriteFolders = len(b.Content.Folders)
	for _, f := range b.Content.Folders {
		s.FavoriteItems += len(f.Items)
	}
	s.ImageFavorites = len(b.Content.ImageFavorites)
	s.SearchHistory = len(b.Content.SearchHistory)
	s.Settings = len(b.Content.Settings)
	if pages, err := b.Pages(); err == nil {
		s.KeptPages = len(pages)
	}
	s.Unresolved = len(b.Content.Unresolved)
	for _, src := range b.Plan.Sources {
		if src.Script != "" {
			s.Sources++
		}
		if src.Data != nil {
			s.WithData++
		}
	}
	s.Importable = s.Sources
	s.RestoredData = s.WithData
	return s
}

// Position is one reading position from a backup, with the source resolved.
type Position struct {
	Source string
	ID     string
	Title  string
	Author string
	Cover  string
	// Episode is 1-based, as history.db wrote it.
	Episode      int
	Page         int
	ReadEpisodes []string
	// Time is in seconds, converted from the millisecond column.
	Time int64
}

// Positions lists the reading positions of a backup whose source was
// recognised. Rows naming a source the client does not know are in the
// plan's pending list instead.
func (b *Backup) Positions() []Position {
	out := make([]Position, 0, len(b.Plan.History))
	for _, h := range b.Plan.History {
		out = append(out, Position{Source: h.SourceKey, ID: h.ID, Title: h.Title,
			Author: h.Subtitle, Cover: h.Cover, Episode: int(h.Episode), Page: int(h.Page),
			ReadEpisodes: h.ReadEpisodes, Time: h.TimeMillis / 1000})
	}
	return out
}

// KeptPage is one page a backup kept, placed in its comic and chapter.
type KeptPage struct {
	Source       string
	ComicID      string
	ComicTitle   string
	ChapterID    string
	ChapterIndex int
	ChapterName  string
	Page         int
	// URL is the page's address in the source, as the backup stored it.
	URL string
	// Time is in seconds.
	Time int64
}

// keptEpisodes is how an image-favourites row stores its chapters:
// a list of {eid, ep, maxPage, epName, imageFavorites: [...]}, as the
// official client writes it.
type keptEpisodes []struct {
	EID        string `json:"eid"`
	EP         int    `json:"ep"`
	EPName     string `json:"epName"`
	Favourites []struct {
		Page     int    `json:"page"`
		ImageKey string `json:"imageKey"`
	} `json:"imageFavorites"`
}

// Pages lists the pages a backup kept whose source is known. Rows naming a
// source the client cannot resolve are left out, and so is anything the
// official client's JSON does not make sense of.
func (b *Backup) Pages() ([]KeptPage, error) {
	var out []KeptPage
	for _, row := range b.Content.ImageFavorites {
		if row.SourceKey == "" {
			continue
		}
		var episodes keptEpisodes
		if len(row.Episodes) == 0 {
			continue
		}
		if err := json.Unmarshal(row.Episodes, &episodes); err != nil {
			b.warn("image favourite " + row.ID + " has unreadable chapters")
			continue
		}
		time := int64(0)
		if row.Time != nil {
			time = *row.Time / 1000
		}
		for _, ep := range episodes {
			for _, fav := range ep.Favourites {
				if fav.ImageKey == "" || fav.Page <= 0 {
					continue
				}
				out = append(out, KeptPage{
					Source:       row.SourceKey,
					ComicID:      row.ID,
					ComicTitle:   row.Title,
					ChapterID:    ep.EID,
					ChapterIndex: ep.EP,
					ChapterName:  ep.EPName,
					Page:         fav.Page,
					URL:          fav.ImageKey,
					Time:         time,
				})
			}
		}
	}
	return out, nil
}

// warn records a problem for the confirmation screen.
func (b *Backup) warn(text string) {
	b.warnings = append(b.warnings, text)
}

// Kept is one comic from a favourite folder.
type Kept struct {
	Source string
	ID     string
	Title  string
	Author string
	Cover  string
	Tags   []string
}

// Favourites lists the folders of a backup whose comics' sources were
// recognised, in the backup's folder order.
func (b *Backup) Favourites() []Folder {
	out := make([]Folder, 0, len(b.Plan.Folders))
	for _, f := range b.Plan.Folders {
		folder := Folder{Name: f.Name}
		for _, it := range f.Items {
			folder.Items = append(folder.Items, Kept{Source: it.SourceKey, ID: it.ID,
				Title: it.Name, Author: it.Author, Cover: it.Cover, Tags: it.Tags})
		}
		out = append(out, folder)
	}
	return out
}

// Folder is one favourite folder.
type Folder struct {
	Name  string
	Items []Kept
}

// Data is one source's stored values, in the order the archive has them.
type Data struct {
	Key    string
	Fields []Field
}

// Field is one data key and its value as written.
type Field struct {
	Name  string
	Value json.RawMessage
}

// SourceData splits each plan source's data object into its keys, so the
// store can keep every value exactly as the backup wrote it.
func (b *Backup) SourceData() ([]Data, error) {
	var out []Data
	for _, src := range b.Plan.Sources {
		if src.Data == nil {
			continue
		}
		fields, err := fields(src.Data)
		if err != nil {
			return nil, fmt.Errorf("archive: %s data: %w", src.Key, err)
		}
		if len(fields) == 0 {
			continue
		}
		out = append(out, Data{Key: src.Key, Fields: fields})
	}
	return out, nil
}

// fields decodes a JSON object keeping its key order.
func fields(raw json.RawMessage) ([]Field, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, errors.New("not an object")
	}
	var out []Field
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		name, _ := keyTok.(string)
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, err
		}
		out = append(out, Field{Name: name, Value: value})
	}
	return out, nil
}
