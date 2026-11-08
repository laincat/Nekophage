// A chapter that the reader window cannot load. The first ChapterImages call
// (while readercheck picks a chapter) answers with pages; the reader's own
// chapter fetch then fails, so reader.html shows an error and never creates
// #pages figure. readercheck must reject that page instead of letting later
// checks - in particular a retry that succeeds - report a pass.
//
// Used only by scripts/desktop-probes.sh as a negative case. No network is
// involved: the fixture is entirely local.
class NoFiguresPages extends ComicSource {
  name = "No Figures Pages"
  key = "no_figures_pages"
  version = "1.0.0"
  minAppVersion = "1.6.0"

  search = {
    load: async (keyword) => ({
      comics: keyword === "probe"
        ? [{ id: "one", title: "No Figures Comic", cover: "" }]
        : [],
      maxPage: 1,
    }),
  }

  comic = {
    loadInfo: async (id) => ({
      id, title: "No Figures Comic", tags: {},
      chapters: {"1": "Chapter 1"},
    }),
    loadEp: async () => {
      globalThis.__loadEpCalls = (globalThis.__loadEpCalls || 0) + 1;
      if (globalThis.__loadEpCalls > 1) {
        throw new Error("deliberate failure in the reader window");
      }
      return { images: ["http://127.0.0.1:18731/1.png", "http://127.0.0.1:18731/2.png"] };
    },
  }
}
