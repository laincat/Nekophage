module github.com/laincat/nekophage

go 1.27.4

require (
	github.com/egoist/mygo v0.10.1-0.20261123085535-da81e0ca50f6
	github.com/laincat/veneracat-core v0.0.0
	golang.org/x/image v0.48.0
	modernc.org/sqlite v1.63.0
)

require (
	github.com/MoeForge/moego v0.1.0-alpha.5 // indirect
	github.com/MoeForge/moejs v0.1.1 // indirect
	github.com/andybalholm/cascadia v1.3.5 // indirect
	github.com/antchfx/htmlquery v1.3.7 // indirect
	github.com/antchfx/xpath v1.3.9 // indirect
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/ebitengine/purego v0.11.3 // indirect
	github.com/go-text/typesetting v0.3.5 // indirect
	github.com/go-text/typesetting-utils v0.0.0-20261009215314-bb530f4842f3 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.1.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/net v0.63.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/text v0.44.0 // indirect
	modernc.org/libc v1.80.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)

// The Core checkout scripts/fetch-core.sh clones into build/core (at the
// latest upstream main). Drop the replace once Core is
// tagged and the require names that version.
replace github.com/laincat/veneracat-core => ./build/core
