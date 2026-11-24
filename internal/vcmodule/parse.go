// Package vcmodule parses, validates and evaluates declarative `.vcmodule`
// configuration overlay modules (`veneracat-module/1-draft`).
//
// Only allowlisted declarative setting keys are permitted. Proxy, MITM,
// rewrite and script directives are explicitly refused; this format is not
// Surge-compatible. Modules are evaluated as pure overlays on top of base
// user settings so disabling or removing a module cleanly withdraws its effect
// without mutating base configuration.
package vcmodule

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Format is the only supported `.vcmodule` format declaration.
const Format = "veneracat-module/1-draft"

// Capability is the collection capability identifier for settings modules.
const Capability = "settings-module.v1"

// MaxModuleBytes bounds the raw size of one `.vcmodule` document.
const MaxModuleBytes = 256 << 10

var (
	// ErrInvalidFormat reports missing or unsupported module metadata/syntax.
	ErrInvalidFormat = errors.New("vcmodule: invalid format")
	// ErrForbiddenDirective reports Surge, proxy, MITM, rewrite or script directives.
	ErrForbiddenDirective = errors.New("vcmodule: forbidden directive (proxy/MITM/script and Surge syntax are not supported)")
	// ErrUnknownSection reports a section outside the host allowlist.
	ErrUnknownSection = errors.New("vcmodule: unknown section")
	// ErrUnknownKey reports a setting key outside the host allowlist.
	ErrUnknownKey = errors.New("vcmodule: unknown setting key")
	// ErrInvalidValue reports a value that fails type or range validation.
	ErrInvalidValue = errors.New("vcmodule: invalid setting value")
	// ErrInvalidID reports a module identifier that is malformed or unsafe as a path segment.
	ErrInvalidID = errors.New("vcmodule: invalid module id")
	// ErrNotFound reports a module id that is not installed.
	ErrNotFound = errors.New("vcmodule: module not found")
	// ErrUnsupportedCapability reports a `#!requires` capability this host does not provide.
	ErrUnsupportedCapability = errors.New("vcmodule: unsupported capability")
)

var supportedCapabilities = map[string]bool{
	Capability: true,
}

const (
	minPageGap  = 0
	maxPageGap  = 200
	minFitWidth = 320
	maxFitWidth = 2400
)

// forbiddenSections explicitly catches Surge/proxy/MITM/script section names so
// callers get a clear security boundary error rather than a generic parse error.
var forbiddenSections = map[string]bool{
	"general":        true,
	"replica":        true,
	"proxy":          true,
	"proxy group":    true,
	"rule":           true,
	"host":           true,
	"url rewrite":    true,
	"header rewrite": true,
	"body rewrite":   true,
	"rewrite":        true,
	"mitm":           true,
	"script":         true,
	"map local":      true,
	"panel":          true,
	"http":           true,
	"network":        true,
	"sgmodule":       true,
}

// forbiddenMetaKeys catches Surge-specific `#!` directives.
var forbiddenMetaKeys = map[string]bool{
	"system":         true,
	"arguments":      true,
	"arguments-desc": true,
	"category":       true,
	"icon":           true,
	"open-url":       true,
	"script":         true,
	"mitm":           true,
}

// Meta holds the `#!` metadata header of a `.vcmodule` file.
type Meta struct {
	ID       string   `json:"id,omitempty"`
	Name     string   `json:"name"`
	Desc     string   `json:"desc,omitempty"`
	Version  string   `json:"version"`
	Format   string   `json:"format"`
	Requires []string `json:"requires,omitempty"`
}

// ReaderSettings holds the declarative reader settings that can be configured
// as base user settings and overlaid by `.vcmodule` files.
type ReaderSettings struct {
	PageGap      int    `json:"pageGap"`
	KeepScreenOn bool   `json:"keepScreenOn"`
	FitWidth     int    `json:"fitWidth"`
	Spread       bool   `json:"spread"`
	Direction    string `json:"direction"`
	AutoNext     bool   `json:"autoNext"`
	Theme        string `json:"theme"`
}

// DefaultReaderSettings returns the built-in reader defaults.
func DefaultReaderSettings() ReaderSettings {
	return ReaderSettings{
		PageGap:      0,
		KeepScreenOn: false,
		FitWidth:     900,
		Spread:       false,
		Direction:    "ltr",
		AutoNext:     false,
		Theme:        "system",
	}
}

// Validate checks that every field in ReaderSettings is within allowed bounds.
func (r ReaderSettings) Validate() error {
	if r.PageGap < minPageGap || r.PageGap > maxPageGap {
		return fmt.Errorf("%w: Reader.page-gap must be between %d and %d", ErrInvalidValue, minPageGap, maxPageGap)
	}
	if r.FitWidth < minFitWidth || r.FitWidth > maxFitWidth {
		return fmt.Errorf("%w: Reader.fit-width must be between %d and %d", ErrInvalidValue, minFitWidth, maxFitWidth)
	}
	switch r.Direction {
	case "ltr", "rtl":
	default:
		return fmt.Errorf("%w: Reader.direction must be ltr or rtl", ErrInvalidValue)
	}
	switch r.Theme {
	case "system", "light", "dark":
	default:
		return fmt.Errorf("%w: Reader.theme must be system, light or dark", ErrInvalidValue)
	}
	return nil
}

// Settings holds all declarative settings supported by `.vcmodule` overlays.
type Settings struct {
	Reader ReaderSettings `json:"reader"`
}

// DefaultSettings returns the built-in default settings.
func DefaultSettings() Settings {
	return Settings{Reader: DefaultReaderSettings()}
}

// Validate checks all settings groups.
func (s Settings) Validate() error {
	return s.Reader.Validate()
}

// ReaderOverlay stores optional overrides for the `[Reader]` section.
// Nil fields are not declared by the module and leave underlying values intact.
type ReaderOverlay struct {
	PageGap      *int    `json:"pageGap,omitempty"`
	KeepScreenOn *bool   `json:"keepScreenOn,omitempty"`
	FitWidth     *int    `json:"fitWidth,omitempty"`
	Spread       *bool   `json:"spread,omitempty"`
	Direction    *string `json:"direction,omitempty"`
	AutoNext     *bool   `json:"autoNext,omitempty"`
	Theme        *string `json:"theme,omitempty"`
}

// Module is one parsed and validated `.vcmodule` document.
type Module struct {
	Meta   Meta          `json:"meta"`
	Reader ReaderOverlay `json:"reader,omitempty"`
	Keys   []string      `json:"keys"`
}

// Layer pairs a module identity with its parsed overlay for ordered evaluation.
type Layer struct {
	ID     string
	Module *Module
}

// Resolved holds base settings, effective settings after applying enabled
// overlays in order, and provenance of which module won each key.
type Resolved struct {
	Base      Settings          `json:"base"`
	Effective Settings          `json:"effective"`
	Applied   []string          `json:"applied,omitempty"`
	Sources   map[string]string `json:"sources,omitempty"`
}

// AllowlistedKeys returns a sorted copy of all canonical `Section.key` names
// accepted by this executor.
func AllowlistedKeys() []string {
	return []string{
		"Reader.auto-next",
		"Reader.direction",
		"Reader.fit-width",
		"Reader.keep-screen-on",
		"Reader.page-gap",
		"Reader.spread",
		"Reader.theme",
	}
}

// Parse validates and parses a `.vcmodule` document.
func Parse(raw []byte) (*Module, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: empty module", ErrInvalidFormat)
	}
	if len(raw) > MaxModuleBytes {
		return nil, fmt.Errorf("%w: module exceeds %d bytes", ErrInvalidFormat, MaxModuleBytes)
	}
	// Strip UTF-8 BOM if present.
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	var (
		mod          Module
		seenMeta     = map[string]bool{}
		seenKeys     = map[string]bool{}
		section      string
		inBody       bool
		declaredKeys []string
	)

	lines := strings.Split(text, "\n")
	for idx, orig := range lines {
		lineNum := idx + 1
		line := strings.TrimSpace(orig)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#!") {
			if inBody {
				return nil, fmt.Errorf("%w: line %d: metadata header must appear before sections", ErrInvalidFormat, lineNum)
			}
			key, val, ok := strings.Cut(line[2:], "=")
			if !ok {
				return nil, fmt.Errorf("%w: line %d: malformed metadata header", ErrInvalidFormat, lineNum)
			}
			key = strings.ToLower(strings.TrimSpace(key))
			val = strings.TrimSpace(val)
			if forbiddenMetaKeys[key] {
				return nil, fmt.Errorf("%w: line %d: #!%s is not allowed", ErrForbiddenDirective, lineNum, key)
			}
			if seenMeta[key] {
				return nil, fmt.Errorf("%w: line %d: duplicate metadata header #!%s", ErrInvalidFormat, lineNum, key)
			}
			seenMeta[key] = true
			switch key {
			case "id":
				if err := ValidateID(val); err != nil {
					return nil, fmt.Errorf("%w: line %d: invalid #!id %q", ErrInvalidID, lineNum, val)
				}
				mod.Meta.ID = val
			case "name":
				if val == "" || len(val) > 128 {
					return nil, fmt.Errorf("%w: line %d: #!name must be 1..128 bytes", ErrInvalidFormat, lineNum)
				}
				mod.Meta.Name = val
			case "desc":
				if len(val) > 512 {
					return nil, fmt.Errorf("%w: line %d: #!desc exceeds 512 bytes", ErrInvalidFormat, lineNum)
				}
				mod.Meta.Desc = val
			case "version":
				if val == "" || len(val) > 64 {
					return nil, fmt.Errorf("%w: line %d: #!version must be 1..64 bytes", ErrInvalidFormat, lineNum)
				}
				mod.Meta.Version = val
			case "format":
				mod.Meta.Format = val
			case "requires":
				reqs, err := parseRequires(val, lineNum)
				if err != nil {
					return nil, err
				}
				mod.Meta.Requires = reqs
			default:
				return nil, fmt.Errorf("%w: line %d: unknown metadata header #!%s", ErrInvalidFormat, lineNum, key)
			}
			continue
		}
		if strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return nil, fmt.Errorf("%w: line %d: unclosed section header", ErrInvalidFormat, lineNum)
			}
			inBody = true
			secName := strings.TrimSpace(line[1 : len(line)-1])
			if secName == "" {
				return nil, fmt.Errorf("%w: line %d: empty section name", ErrInvalidFormat, lineNum)
			}
			lower := strings.ToLower(secName)
			if forbiddenSections[lower] {
				return nil, fmt.Errorf("%w: line %d: section [%s] is forbidden", ErrForbiddenDirective, lineNum, secName)
			}
			if secName != "Reader" {
				return nil, fmt.Errorf("%w: line %d: [%s]", ErrUnknownSection, lineNum, secName)
			}
			section = secName
			continue
		}

		if section == "" {
			return nil, fmt.Errorf("%w: line %d: setting appears before any section", ErrInvalidFormat, lineNum)
		}
		inBody = true
		rawKey, rawVal, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("%w: line %d: expected key = value", ErrInvalidFormat, lineNum)
		}
		key := strings.TrimSpace(rawKey)
		val := stripInlineComment(strings.TrimSpace(rawVal))
		if key == "" || val == "" {
			return nil, fmt.Errorf("%w: line %d: empty key or value", ErrInvalidFormat, lineNum)
		}
		if looksForbiddenAssignment(key, val) {
			return nil, fmt.Errorf("%w: line %d: directive %q is forbidden", ErrForbiddenDirective, lineNum, key)
		}
		canonical := section + "." + key
		if seenKeys[canonical] {
			return nil, fmt.Errorf("%w: line %d: duplicate setting %s", ErrInvalidFormat, lineNum, canonical)
		}
		if err := applyParsedSetting(&mod, section, key, val, lineNum); err != nil {
			return nil, err
		}
		seenKeys[canonical] = true
		declaredKeys = append(declaredKeys, canonical)
	}

	if mod.Meta.Format != Format {
		return nil, fmt.Errorf("%w: #!format must be %q, got %q", ErrInvalidFormat, Format, mod.Meta.Format)
	}
	if mod.Meta.Name == "" {
		return nil, fmt.Errorf("%w: missing required #!name", ErrInvalidFormat)
	}
	if mod.Meta.Version == "" {
		return nil, fmt.Errorf("%w: missing required #!version", ErrInvalidFormat)
	}
	if len(declaredKeys) == 0 {
		return nil, fmt.Errorf("%w: module declares no settings", ErrInvalidFormat)
	}
	sort.Strings(declaredKeys)
	mod.Keys = declaredKeys
	return &mod, nil
}

func parseRequires(val string, lineNum int) ([]string, error) {
	if strings.TrimSpace(val) == "" {
		return nil, fmt.Errorf("%w: line %d: empty #!requires", ErrInvalidFormat, lineNum)
	}
	parts := strings.Split(val, ",")
	seen := map[string]bool{}
	reqs := make([]string, 0, len(parts))
	for _, part := range parts {
		capName := strings.TrimSpace(part)
		if capName == "" {
			return nil, fmt.Errorf("%w: line %d: empty capability in #!requires", ErrInvalidFormat, lineNum)
		}
		lower := strings.ToLower(capName)
		for _, bad := range []string{"mitm", "proxy", "script", "rewrite", "surge"} {
			if strings.Contains(lower, bad) {
				return nil, fmt.Errorf("%w: line %d: capability %q is forbidden", ErrForbiddenDirective, lineNum, capName)
			}
		}
		if !supportedCapabilities[capName] {
			return nil, fmt.Errorf("%w: line %d: %q", ErrUnsupportedCapability, lineNum, capName)
		}
		if !seen[capName] {
			seen[capName] = true
			reqs = append(reqs, capName)
		}
	}
	sort.Strings(reqs)
	return reqs, nil
}

func stripInlineComment(val string) string {
	if i := strings.IndexByte(val, ';'); i >= 0 {
		val = val[:i]
	}
	return strings.TrimSpace(val)
}

func looksForbiddenAssignment(key, val string) bool {
	lk := strings.ToLower(key)
	lv := strings.ToLower(val)
	for _, token := range []string{"script-path", "mitm", "hostname", "http-request", "http-response", "proxy"} {
		if strings.Contains(lk, token) || strings.Contains(lv, token) {
			return true
		}
	}
	return false
}

func applyParsedSetting(mod *Module, section, key, val string, lineNum int) error {
	switch section {
	case "Reader":
		switch key {
		case "page-gap":
			n, err := parseStrictInt(val, minPageGap, maxPageGap)
			if err != nil {
				return fmt.Errorf("%w: line %d: Reader.page-gap: %v", ErrInvalidValue, lineNum, err)
			}
			mod.Reader.PageGap = &n
			return nil
		case "keep-screen-on":
			b, err := parseStrictBool(val)
			if err != nil {
				return fmt.Errorf("%w: line %d: Reader.keep-screen-on: %v", ErrInvalidValue, lineNum, err)
			}
			mod.Reader.KeepScreenOn = &b
			return nil
		case "fit-width":
			n, err := parseStrictInt(val, minFitWidth, maxFitWidth)
			if err != nil {
				return fmt.Errorf("%w: line %d: Reader.fit-width: %v", ErrInvalidValue, lineNum, err)
			}
			mod.Reader.FitWidth = &n
			return nil
		case "spread":
			b, err := parseStrictBool(val)
			if err != nil {
				return fmt.Errorf("%w: line %d: Reader.spread: %v", ErrInvalidValue, lineNum, err)
			}
			mod.Reader.Spread = &b
			return nil
		case "direction":
			d := strings.ToLower(val)
			if d != "ltr" && d != "rtl" {
				return fmt.Errorf("%w: line %d: Reader.direction must be ltr or rtl", ErrInvalidValue, lineNum)
			}
			mod.Reader.Direction = &d
			return nil
		case "auto-next":
			b, err := parseStrictBool(val)
			if err != nil {
				return fmt.Errorf("%w: line %d: Reader.auto-next: %v", ErrInvalidValue, lineNum, err)
			}
			mod.Reader.AutoNext = &b
			return nil
		case "theme":
			t := strings.ToLower(val)
			if t != "system" && t != "light" && t != "dark" {
				return fmt.Errorf("%w: line %d: Reader.theme must be system, light or dark", ErrInvalidValue, lineNum)
			}
			mod.Reader.Theme = &t
			return nil
		default:
			return fmt.Errorf("%w: line %d: Reader.%s", ErrUnknownKey, lineNum, key)
		}
	default:
		return fmt.Errorf("%w: line %d: [%s]", ErrUnknownSection, lineNum, section)
	}
}

func parseStrictInt(val string, minVal, maxVal int) (int, error) {
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("expected integer, got %q", val)
	}
	if n < minVal || n > maxVal {
		return 0, fmt.Errorf("value %d out of range [%d, %d]", n, minVal, maxVal)
	}
	return n, nil
}

func parseStrictBool(val string) (bool, error) {
	switch strings.ToLower(val) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("expected true or false, got %q", val)
	}
}

// Apply evaluates layers in order on top of a copy of base.
// Base is never mutated; later layers override earlier layers per key.
func Apply(base Settings, layers []Layer) Resolved {
	effective := base
	sources := map[string]string{}
	applied := make([]string, 0, len(layers))

	for _, layer := range layers {
		if layer.Module == nil {
			continue
		}
		applied = append(applied, layer.ID)
		r := layer.Module.Reader
		if r.PageGap != nil {
			effective.Reader.PageGap = *r.PageGap
			sources["Reader.page-gap"] = layer.ID
		}
		if r.KeepScreenOn != nil {
			effective.Reader.KeepScreenOn = *r.KeepScreenOn
			sources["Reader.keep-screen-on"] = layer.ID
		}
		if r.FitWidth != nil {
			effective.Reader.FitWidth = *r.FitWidth
			sources["Reader.fit-width"] = layer.ID
		}
		if r.Spread != nil {
			effective.Reader.Spread = *r.Spread
			sources["Reader.spread"] = layer.ID
		}
		if r.Direction != nil {
			effective.Reader.Direction = *r.Direction
			sources["Reader.direction"] = layer.ID
		}
		if r.AutoNext != nil {
			effective.Reader.AutoNext = *r.AutoNext
			sources["Reader.auto-next"] = layer.ID
		}
		if r.Theme != nil {
			effective.Reader.Theme = *r.Theme
			sources["Reader.theme"] = layer.ID
		}
	}

	if len(sources) == 0 {
		sources = nil
	}
	return Resolved{
		Base:      base,
		Effective: effective,
		Applied:   applied,
		Sources:   sources,
	}
}
