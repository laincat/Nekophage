package archive

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/laincat/nekophage/internal/shelf"
	"github.com/laincat/veneracat-core/backup"
	_ "modernc.org/sqlite"
)

// official builds an archive shaped like the official export: a history
// database, a favourites database with one folder, and one source script.
func official(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	history := filepath.Join(dir, "history.db")
	db, err := sql.Open("sqlite", history)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`create table history(id text primary key, title text, subtitle text,
		cover text, time int, type int, ep int, page int, readEpisode text,
		max_page int, chapter_group int)`); err != nil {
		t.Fatal(err)
	}
	hash := strconv.FormatInt(backup.DartStringHash("wnacg"), 10)
	// Two rows: one source the client knows, one it cannot resolve.
	if _, err := db.Exec(`insert into history values(?,?,?,?,?,?,?,?,?,?,?)`,
		"c1", "海贼王", "尾田荣一郎", "http://img/1.jpg", 1700000000000, hash, 209, 7, "1,2,3", 20, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`insert into history values(?,?,?,?,?,?,?,?,?,?,?)`,
		"c2", "未知来源", "", "", 1700000100000, 12345, 1, 1, "", 0, 0); err != nil {
		t.Fatal(err)
	}
	// Image favourites live in the same database as history.
	if _, err := db.Exec(`create table image_favorites(id text, title text, sub_title text,
		author text, tags text, translated_tags text, time int, max_page int,
		source_key text, image_favorites_ep text, other text)`); err != nil {
		t.Fatal(err)
	}
	episodes := `[{"eid":"ch1","ep":0,"maxPage":20,"epName":"第 1 话","imageFavorites":[
		{"page":3,"imageKey":"https://img/a.jpg","eid":"ch1","ep":0},
		{"page":4,"imageKey":"https://img/b.jpg","eid":"ch1","ep":0},
		{"page":5,"imageKey":"","eid":"ch1","ep":0}]}]`
	if _, err := db.Exec(`insert into image_favorites values(?,?,?,?,?,?,?,?,?,?,?)`,
		"c1", "海贼王", "", "尾田荣一郎", "", "", 1700000000000, 20, "wnacg", episodes, "{}"); err != nil {
		t.Fatal(err)
	}
	// A row whose source cannot be resolved is not restored.
	if _, err := db.Exec(`insert into image_favorites values(?,?,?,?,?,?,?,?,?,?,?)`,
		"c9", "另一个源", "", "", "", "", 1700000000000, 5, "", episodes, "{}"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	favourites := filepath.Join(dir, "local_favorite.db")
	db, err = sql.Open("sqlite", favourites)
	if err != nil {
		t.Fatal(err)
	}
	stmts := []string{
		`create table folder_order(folder_name text, order_value int)`,
		`create table folder_sync(folder_name text, source_key text, source_folder text)`,
		`create table "默认收藏"(id text, name TEXT, author TEXT, type int, tags text,
			cover_path TEXT, time TEXT, display_order int, translated_tags text,
			primary key (id, type))`,
		`insert into folder_order values('默认收藏', 0)`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`insert into "默认收藏" values(?,?,?,?,?,?,?,?,?)`,
		"c1", "海贼王", "尾田荣一郎", backup.DartStringHash("wnacg"), "冒险,热血", "http://img/1.jpg", "1700000000000", 0, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`insert into "默认收藏" values(?,?,?,?,?,?,?,?,?)`,
		"c9", "另一个源的作品", "", 999999, "", "", "1700000000000", 1, ""); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	add := func(name string, data []byte) {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"history.db", "local_favorite.db"} {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		add(name, b)
	}
	// The official app's single jar, as a backup carries it.
	cookieDB := filepath.Join(dir, "cookie.db")
	db, err = sql.Open("sqlite", cookieDB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`create table cookies (name text, value text, domain text, path text,
		expires integer, secure integer, httpOnly integer)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`insert into cookies values('sid','secret','.wnacg.com','/',1900000000000,1,1)`); err != nil {
		t.Fatal(err)
	}
	// A row that could never be sent anywhere, to prove it is dropped rather
	// than restored as a cookie that never matches.
	if _, err := db.Exec(`insert into cookies values('junk','x','https://wnacg.com','/',0,0,0)`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	cookieBytes, err := os.ReadFile(cookieDB)
	if err != nil {
		t.Fatal(err)
	}
	add("cookie.db", cookieBytes)
	add("appdata.json", []byte(`{"settings":{},"searchHistory":[]}`))
	add("comic_source/wnacg.js", []byte(`class W extends ComicSource { key = 'wnacg' }`))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "official.venera")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPositionsAndFavouritesResolveSources(t *testing.T) {
	path := official(t)
	b, err := Read(context.Background(), path, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	pos := b.Positions()
	if len(pos) != 1 {
		t.Fatalf("positions = %+v (unresolved rows must stay pending)", pos)
	}
	p := pos[0]
	if p.Source != "wnacg" || p.ID != "c1" || p.Title != "海贼王" || p.Author != "尾田荣一郎" {
		t.Errorf("position = %+v", p)
	}
	if p.Episode != 209 || p.Page != 7 {
		t.Errorf("episode/page = %d/%d", p.Episode, p.Page)
	}
	if p.Time != 1700000000 {
		t.Errorf("time = %d, want seconds", p.Time)
	}
	if len(p.ReadEpisodes) != 3 {
		t.Errorf("readEpisodes = %v", p.ReadEpisodes)
	}

	folders := b.Favourites()
	if len(folders) != 1 || folders[0].Name != "默认收藏" {
		t.Fatalf("folders = %+v", folders)
	}
	if len(folders[0].Items) != 1 || folders[0].Items[0].ID != "c1" {
		t.Fatalf("items = %+v (unresolved ones stay pending)", folders[0].Items)
	}
	if got := folders[0].Items[0].Tags; len(got) != 2 || got[0] != "冒险" {
		t.Errorf("tags = %v", got)
	}
	if pending := len(b.Plan.Pending.History); pending != 1 {
		t.Errorf("pending history = %d", pending)
	}
}

// TestApplyIntoAnEmptyShelf writes a backup into a fresh shelf, the way the
// import window does.
func TestApplyIntoAnEmptyShelf(t *testing.T) {
	b, err := Read(context.Background(), official(t), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	sh, err := shelf.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range b.Positions() {
		if _, err := sh.Merge(shelf.History{
			Comic:   shelf.Comic{Source: p.Source, ID: p.ID, Title: p.Title, Author: p.Author, Cover: p.Cover},
			Chapter: "第 " + strconv.Itoa(p.Episode) + " 话", ChapterIndex: p.Episode - 1,
			Page: p.Page, Time: p.Time}); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range b.Favourites() {
		for _, it := range f.Items {
			if _, err := sh.Add(f.Name, shelf.Comic{Source: it.Source, ID: it.ID,
				Title: it.Title, Author: it.Author, Cover: it.Cover}); err != nil {
				t.Fatal(err)
			}
		}
	}
	history, favourites := sh.Counts()
	if history != 1 || favourites != 1 {
		t.Fatalf("counts = %d/%d", history, favourites)
	}
	h := sh.History(0)[0]
	if h.ChapterIndex != 208 || h.Page != 7 || h.Time != 1700000000 {
		t.Errorf("history = %+v", h)
	}
	if folder, ok := sh.Holds("wnacg", "c1"); !ok || folder != "默认收藏" {
		t.Errorf("holds = %q, %v", folder, ok)
	}
}

func TestCookiesAreOnlyReadWhenAsked(t *testing.T) {
	// The default import leaves credentials behind. Asking for them must name the
	// partitions Core picked, and never anything else.
	plain, err := Read(context.Background(), official(t), []string{"wnacg"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(plain.Plan.Cookies) != 0 || len(plain.Plan.CookieTargets) != 0 {
		t.Fatalf("default import carried cookies: %+v", plain.Plan.Cookies)
	}

	asked, err := Read(context.Background(), official(t), []string{"wnacg"}, Options{Cookies: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(asked.Plan.Cookies) == 0 {
		t.Fatal("asked for cookies and got none")
	}
	if len(asked.Plan.CookieTargets) != 1 || asked.Plan.CookieTargets[0] != "wnacg" {
		t.Fatalf("targets = %v", asked.Plan.CookieTargets)
	}
}

func TestKeptPagesComeFromTheEpisodesColumn(t *testing.T) {
	b, err := Read(context.Background(), official(t), nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	pages, err := b.Pages()
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 {
		t.Fatalf("pages = %+v (the row without a source and the page without a key must not appear)", pages)
	}
	p := pages[0]
	if p.Source != "wnacg" || p.ComicID != "c1" || p.ComicTitle != "海贼王" {
		t.Errorf("page = %+v", p)
	}
	if p.ChapterID != "ch1" || p.ChapterIndex != 0 || p.ChapterName != "第 1 话" {
		t.Errorf("chapter = %+v", p)
	}
	if p.Page != 3 || p.URL != "https://img/a.jpg" {
		t.Errorf("where = %+v", p)
	}
	if p.Time != 1700000000 {
		t.Errorf("time = %d, want seconds", p.Time)
	}
	if got := b.Summarize().KeptPages; got != 2 {
		t.Errorf("summary kept pages = %d", got)
	}
}
