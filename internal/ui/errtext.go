package ui

import "strings"

// explain turns a Core error into a sentence the interface can show.
//
// Core answers in English on purpose: it is the shared layer and has no locale.
// The window is Chinese, so the failures a user actually meets get a Chinese
// sentence, and Core's own words stay after a prefix so a bug report still
// carries them. Anything unrecognised keeps that prefix rather than inventing a
// reason — a wrong explanation is worse than an English one.
func explain(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	lower := strings.ToLower(message)
	has := func(needle string) bool { return strings.Contains(lower, needle) }

	switch {
	case has("context canceled") || has("cancelled"):
		return "操作已取消"
	case has("no such source") || has("source not found") || has("unknown source"):
		return "找不到这个漫画源,可能已被删除"
	case has("not installed"):
		return "漫画源还未安装"
	case has("timeout") || has("deadline exceeded"):
		return "网络超时,请稍后重试"
	case has("no such comic") || has("comic not found"):
		return "源里找不到这部漫画"
	case has("challenge") || has("cloudflare") || has("forbidden") || has("403"):
		return "源要求人机验证或拒绝了请求"
	case has("permission") || has("not allowed"):
		return "没有权限执行这个操作"
	case strings.TrimSpace(message) == "":
		return "操作失败,源没有给出原因"
	}
	return "源返回错误:" + message
}
