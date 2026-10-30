module github.com/laincat/nekophage

go 1.27.3

require (
	github.com/egoist/mygo v0.7.3
	github.com/laincat/veneracat-core v0.0.0
	modernc.org/sqlite v1.61.0
)

require (
	github.com/MoeForge/moejs v0.1.0-alpha.11.0.20261028053351-0a0ed36ae7db // indirect
	github.com/andybalholm/cascadia v1.3.5 // indirect
	github.com/antchfx/htmlquery v1.3.7 // indirect
	github.com/antchfx/xpath v1.3.9 // indirect
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/ebitengine/purego v0.11.1 // indirect
	github.com/go-text/typesetting v0.3.5 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.1.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/image v0.46.0 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	modernc.org/libc v1.79.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)

// The Core checkout scripts/fetch-core.sh clones into build/core (at the
// branch or commit in scripts/core-version). Drop the replace once Core is
// tagged and the require names that version.
replace github.com/laincat/veneracat-core => ./build/core
