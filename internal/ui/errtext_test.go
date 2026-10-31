package ui

import (
	"errors"
	"testing"
)

// The window is Chinese and Core answers in English. These are the failures a
// user actually meets; each gets a sentence, and Core's words stay behind the
// prefix so a report keeps them.
func TestExplainKnownFailures(t *testing.T) {
	cases := []struct {
		err   error
		want  string
		exact bool
	}{
		{errors.New("context canceled"), "操作已取消", true},
		{errors.New("core: unknown source mini"), "找不到这个漫画源,可能已被删除", true},
		{errors.New("plugin: not installed"), "漫画源还未安装", true},
		{errors.New("Get https://x: i/o timeout"), "网络超时,请稍后重试", true},
		{errors.New("js: Challenge needed"), "源要求人机验证或拒绝了请求", true},
		{errors.New(""), "操作失败,源没有给出原因", true},
	}
	for _, c := range cases {
		if got := explain(c.err); got != c.want {
			t.Fatalf("explain(%q) = %q; want %q", c.err, got, c.want)
		}
	}
}

// An unrecognised failure keeps Core's own words: a wrong explanation would send
// the user looking in the wrong place.
func TestExplainKeepsUnknownText(t *testing.T) {
	err := errors.New("js: Invalid status code: 210")
	got := explain(err)
	if got != "源返回错误:"+err.Error() {
		t.Fatalf("explain = %q", got)
	}
}

func TestExplainNilIsEmpty(t *testing.T) {
	if got := explain(nil); got != "" {
		t.Fatalf("explain(nil) = %q", got)
	}
}
