package magpie

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Model is one entry of the gateway's catalog: a model or a routing group. The
// ID is kept exactly as sent: a group is not a provider model, and unrolling it
// here would take the routing decision away from Magpie.
type Model struct {
	ID   string
	Name string
}

// Models lists what the gateway serves. A gateway that cannot list models is an
// error, not an empty catalog.
func (c *Client) Models(ctx context.Context) ([]Model, error) {
	req, err := c.newRequest(ctx, "GET", "/v1/models", nil)
	if err != nil {
		return nil, err
	}
	body, err := c.do(req)
	if err != nil {
		return nil, err
	}
	var answer struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.UseNumber()
	if err := decoder.Decode(&answer); err != nil {
		return nil, errors.New("magpie: Magpie 的模型目录不是预期的 JSON")
	}
	out := make([]Model, 0, len(answer.Data))
	for _, entry := range answer.Data {
		if entry.ID == "" {
			continue
		}
		name := entry.Name
		if name == "" {
			name = entry.ID
		}
		out = append(out, Model{ID: entry.ID, Name: name})
	}
	return out, nil
}

// Health reports whether the gateway answers at all, for the settings page's
// "test connection" button. It does not touch a provider account.
func (c *Client) Health(ctx context.Context) error {
	_, err := c.Models(ctx)
	return err
}

// Describe is what the settings page shows after a successful connection.
func (c *Client) Describe(ctx context.Context) (string, error) {
	models, err := c.Models(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Magpie 可用,共 %d 个模型或路由组", len(models)), nil
}
