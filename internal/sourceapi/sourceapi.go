// Package sourceapi names the Venera source API level this client implements.
//
// A source declares `minAppVersion`, and the official client compares it with
// its own version, so this number is what every entry point has to report:
// the app itself and every checking tool. When the tools reported an older
// level than the app, a source gated on the newer one installed in the app and
// was refused by the tool, which made the tool's verdict wrong about the
// product.
package sourceapi

// Level is the source API level. It tracks the official client's current
// release, which is 1.6.3: sources gated on 1.6.3 (cxc, mangabz) were run end
// to end — search, details, chapters, images — before it was raised, so the
// claim is tested rather than assumed. A source that needs an interface this
// client lacks still fails at the call that needs it, which is what its own
// error says.
const Level = "1.6.3"
