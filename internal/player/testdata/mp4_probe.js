class MP4Probe extends VideoSource {
  name = "Local MP4 probe";
  key = "mp4_probe";
  version = "1.0.0";
  search = {
    load: async () => {
      await Network.get("http://127.0.0.1:18732/login", {});
      return { videos: [{ id: "clip", title: "Local MP4 clip" }], maxPage: 1 };
    },
  };
  video = {
    loadInfo: async () => ({
      id: "clip", title: "Local MP4 clip",
      lines: [{ id: "local", name: "Local", episodes: [{ id: "ep1", title: "Episode 1" }] }],
    }),
    loadPlay: async () => ({
      url: "http://127.0.0.1:18732/clip.mp4",
      format: "mp4",
      headers: { Referer: "https://fixture.invalid/" },
    }),
  };
}
