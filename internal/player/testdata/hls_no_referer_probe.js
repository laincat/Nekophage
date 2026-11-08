class HLSNoRefererProbe extends VideoSource {
  name = "Local HLS no-referer negative";
  key = "hls_no_referer";
  version = "1.0.0";
  search = {
    load: async () => {
      await Network.get("http://127.0.0.1:18732/login", {});
      return { videos: [{ id: "clip", title: "Local HLS clip" }], maxPage: 1 };
    },
  };
  video = {
    loadInfo: async () => ({
      id: "clip", title: "Local HLS clip",
      lines: [{ id: "local", name: "Local", episodes: [{ id: "ep1", title: "Episode 1" }] }],
    }),
    // Deliberately no Referer: the fixture's -video-auth check must reject it.
    loadPlay: async () => ({
      url: "http://127.0.0.1:18732/master.m3u8",
      format: "hls",
    }),
  };
}
