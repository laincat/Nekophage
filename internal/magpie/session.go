package magpie

import (
	"context"
	"errors"
	"sync"
)

// Session is the connection as the interface uses it: it owns the stored
// settings, keeps one client in step with them, and hands out streamed answers.
//
// It exists so the window has one seam to call. Everything policy-shaped stays
// in Magpie: this type chooses a model from the user's own mapping and never
// picks a provider, a fallback or a retry.
type Session struct {
	path string

	mu       sync.Mutex
	settings Settings
	client   *Client
}

// OpenSession reads the stored settings. A missing file is the default state; a
// damaged one is an error, because starting over silently would look like the
// gateway had vanished.
func OpenSession(path string) (*Session, error) {
	settings, err := LoadSettings(path)
	if err != nil {
		return nil, err
	}
	return &Session{path: path, settings: settings}, nil
}

// Settings returns what is stored now.
func (s *Session) Settings() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings
}

// Update stores new settings and drops the client built from the old ones, so
// the next request uses the address the user just typed.
func (s *Session) Update(next Settings) error {
	if err := next.Validate(); err != nil {
		return err
	}
	if err := SaveSettings(s.path, next); err != nil {
		return err
	}
	s.mu.Lock()
	s.settings = next
	s.client = nil
	s.mu.Unlock()
	return nil
}

// clientFor returns a client for the current settings, building one if the
// settings changed since the last request.
func (s *Session) clientFor() (*Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil {
		return s.client, nil
	}
	client, err := New(s.settings.ClientConfig())
	if err != nil {
		return nil, err
	}
	s.client = client
	return client, nil
}

// Test asks the gateway for its catalog and describes it, which is what the
// "test connection" button reports. It touches no provider account.
func (s *Session) Test(ctx context.Context) (string, error) {
	client, err := s.clientFor()
	if err != nil {
		return "", err
	}
	return client.Describe(ctx)
}

// Models lists the gateway's catalog through the current connection.
func (s *Session) Models(ctx context.Context) ([]Model, error) {
	client, err := s.clientFor()
	if err != nil {
		return nil, err
	}
	return client.Models(ctx)
}

// Ask sends one task and returns the answer as it arrives. The model comes from
// the user's tier mapping; an empty result leaves the choice to the gateway.
func (s *Session) Ask(ctx context.Context, task Task, messages []Message) (<-chan Event, error) {
	if len(messages) == 0 {
		return nil, errors.New("magpie: 没有要发送的消息")
	}
	client, err := s.clientFor()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	model := s.settings.ModelFor(task)
	s.mu.Unlock()
	return client.Chat(ctx, Request{Model: model, Messages: messages})
}
