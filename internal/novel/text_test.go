package novel

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestParagraphsKeepReadingOrderAndDropActiveContent(t *testing.T) {
	input := `<html><head><title>Page title</title></head><body>
		<h2>第一章</h2><div>甲<em>乙</em>&nbsp;&amp; 丙<br><br>丁</div>
		<script>script payload</script><style>css payload</style>
		<iframe src="https://example.invalid/">frame payload</iframe>
		<template><p>template payload</p></template><svg><text>svg payload</text></svg>
		<div hidden>hidden text</div><div aria-hidden="true">other hidden text</div>
		<img src=x onerror="handler payload"><a href="javascript:alert(1)">链接文字</a>
		<p>&lt;script&gt;这是书中的原文&lt;/script&gt;<!-- comment --></p>
		<p>未闭合段落<script>unclosed script`
	want := []string{"第一章", "甲乙 & 丙", "丁", "链接文字", "<script>这是书中的原文</script>", "未闭合段落"}
	got, err := Paragraphs(input)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("paragraphs = %q, %v", got, err)
	}
	if got, err := Paragraphs(strings.Repeat("a", maxChapterBytes+1)); err == nil || len(got) != 0 {
		t.Fatalf("oversized chapter returned %d paragraphs, %v", len(got), err)
	}
}

func TestPagesPreserveParagraphsAndUTF8(t *testing.T) {
	long := strings.Repeat("汉", 2001)
	got := Pages([]string{"甲", "乙", long, "丙"})
	if len(got) != 4 || got[0] != "甲\n\n乙" || got[3] != "汉\n\n丙" || got[1]+got[2]+"汉" != long {
		t.Fatalf("page count = %d or text changed", len(got))
	}
	for _, page := range got {
		if !utf8.ValidString(page) || utf8.RuneCountInString(page) > 1000 {
			t.Fatal("invalid page boundary")
		}
	}
}
