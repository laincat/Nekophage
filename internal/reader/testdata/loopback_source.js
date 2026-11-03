// A comic source with no site behind it: the chapter's pages live on
// 127.0.0.1:18731, which `cmd/readercheck -serve` fills with one tall PNG per
// path. Two pages, because the double-page layout cannot be measured on a
// one-page chapter — `-layout` used to need a real source for that reason.
class LoopbackPages extends ComicSource {
  name = "Loopback Pages"
  key = "loopback_pages"
  version = "1.0.0"
  minAppVersion = "1.6.0"

  search = {
    load: async (keyword) => ({
      comics: keyword === "probe"
        ? [{ id: "one", title: "Loopback Comic", cover: "http://127.0.0.1:18731/cover.png" }]
        : [],
      maxPage: 1,
    }),
  }

  comic = {
    // Chapters are a map of id to title, and a chapter's pages come back from
    // loadEp: that is the shape Core reads for v2 sources.
    loadInfo: async (id) => ({
      id, title: "Loopback Comic", tags: {},
      chapters: {"1": "Chapter 1"},
    }),
    loadEp: async () => ({
      images: ["http://127.0.0.1:18731/1.png", "http://127.0.0.1:18731/2.png"],
    }),
    // A thread and a favourites list, so the probes that need one are not
    // forced to borrow a script from another repository. Both answer from the
    // fixture itself: no site, no login.
    loadComments: async () => ({
      comments: [{
        id: "c1",
        userName: "Fixture Reader",
        content: "a comment from the fixture",
        time: "2026-01-01",
      }],
      maxPage: 1,
    }),
  }

  favorites = {
    multiFolder: false,
    addOrDelFavorite: async () => true,
    loadFolders: async () => ({ folders: { f1: "收藏" }, favorited: [] }),
    loadComics: async () => ({
      comics: [{ id: "one", title: "Loopback Comic", cover: "" }],
      maxPage: 1,
    }),
  }
}
