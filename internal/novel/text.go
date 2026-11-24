// Package novel prepares chapter text for the native desktop reader.
package novel

import (
	"errors"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

const maxChapterBytes = 4 << 20

// Paragraphs extracts visible text without retaining markup, attributes or URLs.
// The result is intended for a plain-text widget, as entities can decode to
// literal angle brackets. It must never be passed to an HTML renderer.
func Paragraphs(source string) ([]string, error) {
	if len(source) > maxChapterBytes {
		return nil, errors.New("章节正文超过 4 MiB")
	}
	doc, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return nil, err
	}

	var paragraphs []string
	var text strings.Builder
	flush := func() {
		if paragraph := strings.Join(strings.Fields(text.String()), " "); paragraph != "" {
			paragraphs = append(paragraphs, paragraph)
		}
		text.Reset()
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			text.WriteString(strings.Map(func(r rune) rune {
				if unicode.IsControl(r) && !unicode.IsSpace(r) {
					return -1
				}
				return r
			}, n.Data))
			return
		case html.DocumentNode:
		case html.ElementNode:
			if hidden(n) {
				return
			}
		default:
			return
		}
		if paragraphBoundary(n.Data) {
			flush()
			defer flush()
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	flush()
	return paragraphs, nil
}

func hidden(n *html.Node) bool {
	if n.Namespace != "" {
		return true
	}
	switch n.Data {
	case "head", "script", "style", "template", "noscript", "iframe", "object", "embed", "audio", "video", "canvas", "form", "button", "select", "textarea":
		return true
	}
	for _, attr := range n.Attr {
		if attr.Key == "hidden" || (attr.Key == "aria-hidden" && strings.EqualFold(attr.Val, "true")) {
			return true
		}
	}
	return false
}

func paragraphBoundary(tag string) bool {
	switch tag {
	case "br", "hr", "p", "div", "section", "article", "main", "blockquote", "pre", "li", "dt", "dd", "tr", "td", "th",
		"h1", "h2", "h3", "h4", "h5", "h6":
		return true
	}
	return false
}

// Pages groups paragraphs for page-based reading and progress. A long block
// without paragraph breaks is split at rune boundaries so it remains readable.
func Pages(paragraphs []string) []string {
	const pageRunes = 1000
	var pages []string
	var page strings.Builder
	used := 0
	flush := func() {
		if used > 0 {
			pages = append(pages, page.String())
			page.Reset()
			used = 0
		}
	}
	for _, paragraph := range paragraphs {
		if paragraph == "" {
			continue
		}
		start, length := 0, 0
		for i := range paragraph {
			if length == pageRunes {
				flush()
				pages = append(pages, paragraph[start:i])
				start, length = i, 0
			}
			length++
		}
		if used+2+length > pageRunes {
			flush()
		}
		if used > 0 {
			page.WriteString("\n\n")
			used += 2
		}
		page.WriteString(paragraph[start:])
		used += length
	}
	flush()
	return pages
}
