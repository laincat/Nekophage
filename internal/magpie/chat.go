package magpie

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/laincat/veneracat-core/host/sse"
)

// EventKind labels one streamed event.
type EventKind string

const (
	// EventText carries a piece of the answer.
	EventText EventKind = "text"
	// EventDone means the answer is complete; the channel closes after it.
	EventDone EventKind = "done"
	// EventError carries a failure that happened mid-stream.
	EventError EventKind = "error"
)

// Event is one thing that happened while the answer was arriving.
type Event struct {
	Kind EventKind
	Text string
	// Raw is the gateway's own JSON for this event, kept so a field the UI
	// does not understand yet is still available to a caller that does.
	Raw string
}

// Message is one turn of the conversation as sent to the gateway.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Request is one task.
type Request struct {
	// Model is a model or routing-group ID from the catalog. Empty uses the
	// gateway's own choice, which is the documented behaviour of its default
	// route.
	Model    string
	System   string
	Messages []Message
	// MaxTokens is the answer limit for the protocols that require one.
	MaxTokens int
}

// Chat sends a task and returns the answer as it arrives.
//
// The channel closes when the answer ends; a failure is delivered as one
// EventError before the close, so a partially rendered answer stays on screen
// and the reason is shown beneath it. Cancelling ctx stops the request: the
// gateway sees the connection close, and this client keeps nothing running.
func (c *Client) Chat(ctx context.Context, req Request) (<-chan Event, error) {
	if len(req.Messages) == 0 {
		return nil, errors.New("magpie: 没有要发送的消息")
	}
	body, err := c.body(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := c.newRequest(ctx, "POST", paths[c.protocol], body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, transportError(ctx, err)
	}
	if err := statusError(resp.StatusCode, nil); err != nil {
		resp.Body.Close()
		return nil, err
	}
	decoder, err := sse.NewDecoder(resp.Body, streamLimit)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}

	out := make(chan Event, 8)
	go func() {
		defer close(out)
		defer resp.Body.Close()
		done := false
		for {
			msg, err := decoder.Next()
			if err != nil {
				if errors.Is(err, io.EOF) {
					if !done {
						// The stream stopped without saying it finished: that is
						// an interrupted answer, not a complete one.
						send(ctx, out, Event{Kind: EventError, Text: ErrStreamInterrupted.Error()})
					}
					return
				}
				if err := ctx.Err(); err != nil {
					send(ctx, out, Event{Kind: EventError, Text: err.Error()})
					return
				}
				send(ctx, out, Event{Kind: EventError, Text: streamReason(err)})
				return
			}
			event, ok := decodeEvent(c.protocol, msg)
			if !ok {
				continue
			}
			if event.Kind == EventDone {
				done = true
			}
			if !send(ctx, out, event) {
				return
			}
			if event.Kind == EventError {
				return
			}
		}
	}()
	return out, nil
}

// send delivers one event unless the caller has already given up, so a closed
// channel or a cancelled context never blocks the reader.
func send(ctx context.Context, out chan<- Event, event Event) bool {
	select {
	case out <- event:
		return true
	case <-ctx.Done():
		return false
	}
}

// streamReason describes a broken stream in terms a user can act on.
func streamReason(err error) string {
	if errors.Is(err, sse.ErrEventTooLarge) {
		return "Magpie 发来的单个流式事件超过了大小上限"
	}
	return ErrStreamInterrupted.Error() + "(" + err.Error() + ")"
}

// body builds the native request body for this protocol. Fields are written as
// the protocol expects; the client does not translate between protocols.
func (c *Client) body(req Request) ([]byte, error) {
	doc := map[string]any{"stream": true}
	if req.Model != "" {
		doc["model"] = req.Model
	}
	messages := make([]Message, 0, len(req.Messages)+1)
	if req.System != "" && c.protocol != Messages {
		messages = append(messages, Message{Role: "system", Content: req.System})
	}
	messages = append(messages, req.Messages...)
	switch c.protocol {
	case Messages:
		doc["messages"] = messages
		// Anthropic requires a limit; the gateway refuses the request without it.
		limit := req.MaxTokens
		if limit <= 0 {
			limit = 4096
		}
		doc["max_tokens"] = limit
		if req.System != "" {
			doc["system"] = req.System
		}
	case Chat:
		doc["messages"] = messages
		if req.MaxTokens > 0 {
			doc["max_tokens"] = req.MaxTokens
		}
	default: // Responses
		doc["input"] = messages
		if req.MaxTokens > 0 {
			doc["max_output_tokens"] = req.MaxTokens
		}
	}
	return json.Marshal(doc)
}
