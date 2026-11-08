class HLSNoCookieProbe extends VideoSource {
  name = "Local HLS no-cookie negative";
  key = "hls_no_cookie";
  version = "1.0.0";
  search = {
    // Deliberately no /login call: no fixture Cookie is ever set.
    load: async () => ({ videos: [{ id: "clip", title: "Local HLS clip" }], maxPage: 1 }),
  };
  video = {
    loadInfo: async () => ({
      id: "clip", title: "Local HLS clip",
      lines: [{ id: "local", name: "Local", episodes: [{ id: "ep1", title: "Episode 1" }] }],
    }),
    loadPlay: async () => ({
      url: "http://127.0.0.1:18732/master.m3u8",
      format: "hls",
      headers: { Referer: "https://fixture.invalid/" },
    }),
  };
}
