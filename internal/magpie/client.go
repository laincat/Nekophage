// Package magpie talks to a Magpie gateway installed on this machine.
//
// Magpie keeps the plugins, provider accounts, sign-in refresh, model catalog
// and routing. This package only connects: it lists the models the gateway
// offers, sends a task in one of the standard protocols, streams the events
// back and stops when the caller cancels. It stores no provider credential and
// makes no routing decision of its own.
package magpie

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/laincat/veneracat-core/host/sse"
)

// DefaultAddr is where Magpie's gateway listens unless the user changed it.
const DefaultAddr = "127.0.0.1:3425"

// Protocol is the wire shape a request uses.
type Protocol string

const (
	// Chat is OpenAI Chat Completions.
	Chat Protocol = "chat"
	// Responses is OpenAI Responses.
	Responses Protocol = "responses"
	// Messages is Anthropic Messages.
	Messages Protocol = "messages"
)

// paths maps a protocol to its endpoint under the gateway root.
var paths = map[Protocol]string{
	Chat:      "/v1/chat/completions",
	Responses: "/v1/responses",
	Messages:  "/v1/messages",
}

// Config is how to reach the gateway.
type Config struct {
	// BaseURL is the gateway root, "http://127.0.0.1:3425" by default. A bare
	// "127.0.0.1:3425" is accepted and given a scheme.
	BaseURL string
	// Token is the local gateway identity. Empty uses the documented local
	// token "magpie"; it is not a provider key and is never logged.
	Token string
	// Protocol selects the request shape. Empty means Chat.
	Protocol Protocol
	// Timeout bounds one request. Zero means two minutes.
	Timeout time.Duration
}

// Client is a gateway connection. It is safe for concurrent use.
type Client struct {
	base     string
	token    string
	protocol Protocol
	http     *http.Client
}

// ErrNotRunning means nothing answered on the configured address.
var ErrNotRunning = errors.New("连不上本机 Magpie:请先启动 Magpie,或检查网关地址")

// ErrRejected means the gateway answered but refused this caller.
var ErrRejected = errors.New("Magpie 拒绝了这个身份:请核对网关标识")

// ErrModelNotFound means the gateway does not serve the requested model or group.
var ErrModelNotFound = errors.New("Magpie 的模型目录里没有这个模型或路由组")

// ErrStreamInterrupted means the stream ended before the gateway said it was done.
var ErrStreamInterrupted = errors.New("Magpie 的流式响应中断了")

// New builds a client. It validates the address here so a typo is reported
// before a request is attempted.
func New(cfg Config) (*Client, error) {
	base, err := normalizeBase(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	protocol := cfg.Protocol
	if protocol == "" {
		protocol = Chat
	}
	if _, ok := paths[protocol]; !ok {
		return nil, fmt.Errorf("magpie: 不支持的协议 %q(可用 chat、responses、messages)", string(protocol))
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	token := cfg.Token
	if token == "" {
		token = "magpie"
	}
	for i := 0; i < len(token); i++ {
		if token[i] < 0x20 || token[i] == 0x7f {
			return nil, errors.New("magpie: 网关标识含有控制字符")
		}
	}
	return &Client{
		base:     base,
		token:    token,
		protocol: protocol,
		http:     &http.Client{Timeout: timeout},
	}, nil
}

// Protocol is the request shape this client sends.
func (c *Client) Protocol() Protocol { return c.protocol }

// BaseURL is the normalized gateway root.
func (c *Client) BaseURL() string { return c.base }

// normalizeBase accepts what a user types into a settings field.
func normalizeBase(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = DefaultAddr
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("magpie: 网关地址不是有效的 URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("magpie: 网关地址必须是 http 或 https")
	}
	if u.Hostname() == "" {
		return "", errors.New("magpie: 网关地址缺少主机名")
	}
	if u.User != nil {
		return "", errors.New("magpie: 网关地址不能包含用户名或密码")
	}
	if u.RawQuery != "" || u.Fragment != "" || u.ForceQuery {
		return "", errors.New("magpie: 网关地址不能带查询串或片段")
	}
	return strings.TrimRight(u.String(), "/"), nil
}

// newRequest builds one authenticated request for path.
func (c *Client) newRequest(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	target := c.base + path
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return nil, err
	}
	// The local identity is what the reference gateway expects; a provider key
	// never appears here because this client does not hold one.
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// do performs one request and returns the body, mapping transport failures to
// the diagnostics a user can act on.
func (c *Client) do(req *http.Request) ([]byte, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, transportError(req.Context(), err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil {
		return nil, transportError(req.Context(), err)
	}
	if int64(len(body)) > maxResponse {
		return nil, errors.New("magpie: Magpie 的响应超过了大小上限")
	}
	if err := statusError(resp.StatusCode, body); err != nil {
		return nil, err
	}
	return body, nil
}

// maxResponse bounds one non-streamed answer.
const maxResponse = 8 << 20

// transportError turns "connection refused" into the advice it deserves, while
// keeping a real cancellation a cancellation.
func transportError(ctx context.Context, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
		return err
	}
	return fmt.Errorf("%w(%v)", ErrNotRunning, err)
}

// statusError maps a non-2xx answer to a diagnostic, using the gateway's own
// message when it sent one.
func statusError(status int, body []byte) error {
	if status >= 200 && status < 300 {
		return nil
	}
	detail := strings.TrimSpace(string(body))
	if len(detail) > 300 {
		detail = detail[:300]
	}
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ErrRejected
	case status == http.StatusNotFound:
		return ErrModelNotFound
	case status >= 500:
		return fmt.Errorf("Magpie 网关出错(HTTP %d)%s", status, suffix(detail))
	default:
		return fmt.Errorf("Magpie 拒绝了这个请求(HTTP %d)%s", status, suffix(detail))
	}
}

func suffix(detail string) string {
	if detail == "" {
		return ""
	}
	return ":" + detail
}

// streamLimit bounds one SSE event.
const streamLimit = sse.DefaultMaxEventBytes

// decodeEvent turns one SSE message into an event. The dialects differ in
// envelope but agree on "text arrived" and "it ended", which is all a caller
// needs to render a reply and to know when to stop.
func decodeEvent(protocol Protocol, msg sse.Event) (Event, bool) {
	data := strings.TrimSpace(msg.Data)
	if data == "" {
		return Event{}, false
	}
	if data == "[DONE]" {
		return Event{Kind: EventDone}, true
	}
	var payload map[string]json.RawMessage
	decoder := json.NewDecoder(strings.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		// Not JSON: never pass an unparsed event to the UI, but do not end the
		// stream over one stray comment-like block either.
		return Event{}, false
	}
	event := Event{Raw: data}
	if failure := errorText(payload); failure != "" {
		event.Kind = EventError
		event.Text = failure
		return event, true
	}
	switch protocol {
	case Messages:
		if text, ok := anthropicText(payload); ok {
			event.Kind = EventText
			event.Text = text
			return event, true
		}
	default:
		if text, ok := openAIText(protocol, payload); ok {
			event.Kind = EventText
			event.Text = text
			return event, true
		}
	}
	if done, ok := finished(payload); ok && done {
		return Event{Kind: EventDone, Raw: data}, true
	}
	return Event{}, false
}

// errorText finds a provider error carried inside a 200 stream.
func errorText(payload map[string]json.RawMessage) string {
	raw, ok := payload["error"]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var shape struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	}
	if err := json.Unmarshal(raw, &shape); err != nil {
		return strings.TrimSpace(string(raw))
	}
	if shape.Message != "" {
		return shape.Message
	}
	if shape.Type != "" {
		return shape.Type
	}
	return strings.TrimSpace(string(raw))
}

// openAIText reads Chat Completions and Responses deltas.
func openAIText(protocol Protocol, payload map[string]json.RawMessage) (string, bool) {
	if protocol == Responses {
		// Only `delta` carries new text. `text` is the finished text: the
		// stream sends response.output_text.delta with delta="ready" and then
		// response.output_text.done with text="ready", so reading both appends
		// the whole answer a second time and the interface shows it twice.
		raw, ok := payload["delta"]
		if !ok {
			return "", false
		}
		var text string
		if json.Unmarshal(raw, &text) != nil {
			return "", false
		}
		return text, true
	}
	raw, ok := payload["choices"]
	if !ok {
		return "", false
	}
	var choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(raw, &choices); err != nil {
		return "", false
	}
	var out strings.Builder
	for _, choice := range choices {
		if choice.Delta.Content != "" {
			out.WriteString(choice.Delta.Content)
			continue
		}
		out.WriteString(choice.Message.Content)
	}
	if out.Len() == 0 {
		return "", false
	}
	return out.String(), true
}

// anthropicText reads Messages deltas.
func anthropicText(payload map[string]json.RawMessage) (string, bool) {
	raw, ok := payload["delta"]
	if !ok {
		return "", false
	}
	var delta struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &delta); err != nil || delta.Text == "" {
		return "", false
	}
	return delta.Text, true
}

// finished reports whether an event says the answer is complete.
func finished(payload map[string]json.RawMessage) (bool, bool) {
	if raw, ok := payload["type"]; ok {
		var kind string
		if json.Unmarshal(raw, &kind) == nil && (kind == "message_stop" || kind == "response.completed") {
			return true, true
		}
	}
	if raw, ok := payload["choices"]; ok {
		var choices []struct {
			FinishReason *string `json:"finish_reason"`
		}
		if json.Unmarshal(raw, &choices) == nil {
			for _, choice := range choices {
				if choice.FinishReason != nil && *choice.FinishReason != "" {
					return true, true
				}
			}
		}
	}
	return false, false
}
