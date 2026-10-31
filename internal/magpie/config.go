package magpie

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Task is how demanding a request is. It selects a model or routing group the
// user configured; it is a choice, not a classifier. Deciding what "complex"
// means stays with the user and with Magpie's own rules.
type Task string

const (
	// TaskSimple is a short lookup or a formatting request.
	TaskSimple Task = "simple"
	// TaskNormal is the default when nothing else is configured.
	TaskNormal Task = "normal"
	// TaskComplex is work that needs the user's strongest group.
	TaskComplex Task = "complex"
)

// Tasks lists every selectable tier, in the order a settings screen shows them.
var Tasks = []Task{TaskSimple, TaskNormal, TaskComplex}

// Settings is what the connection screen stores. It holds no provider account:
// those belong to Magpie.
type Settings struct {
	// BaseURL is the gateway root. Empty means DefaultAddr.
	BaseURL string `json:"baseUrl"`
	// Protocol is the request shape this client sends. Empty means Chat.
	Protocol Protocol `json:"protocol"`
	// Token is the local gateway identity. Empty means the documented local
	// token. It is not a provider key.
	Token string `json:"token,omitempty"`
	// Model is an explicit model or group chosen by the user. Empty means the
	// gateway's own default route.
	Model string `json:"model,omitempty"`
	// Tiers maps a Task to a model or group. A tier that is absent or empty
	// falls back to Model, then to the gateway's default.
	Tiers map[string]string `json:"tiers,omitempty"`
}

// DefaultSettings is the state before the user configures anything.
func DefaultSettings() Settings {
	return Settings{BaseURL: "http://" + DefaultAddr, Protocol: Chat, Tiers: map[string]string{}}
}

// Validate reports what would make the connection fail, before it is used.
// A typo in a tier name is an error rather than a silently ignored setting.
func (s Settings) Validate() error {
	if _, err := normalizeBase(s.BaseURL); err != nil {
		return err
	}
	protocol := s.Protocol
	if protocol == "" {
		protocol = Chat
	}
	if _, ok := paths[protocol]; !ok {
		return fmt.Errorf("magpie: 不支持的协议 %q(可用 chat、responses、messages)", string(protocol))
	}
	for i := 0; i < len(s.Token); i++ {
		if s.Token[i] < 0x20 || s.Token[i] == 0x7f {
			return errors.New("magpie: 网关标识含有控制字符")
		}
	}
	for tier := range s.Tiers {
		if !knownTask(Task(tier)) {
			return fmt.Errorf("magpie: 不认识的任务档位 %q(可用 simple、normal、complex)", tier)
		}
	}
	return nil
}

func knownTask(t Task) bool {
	for _, candidate := range Tasks {
		if candidate == t {
			return true
		}
	}
	return false
}

// ClientConfig is the connection described by these settings.
func (s Settings) ClientConfig() Config {
	return Config{BaseURL: s.BaseURL, Token: s.Token, Protocol: s.Protocol}
}

// ModelFor returns the model or group to send for task: the tier's own choice,
// then the explicit choice, then "" for the gateway's default route.
//
// It never invents a model and never reorders candidates: picking among the
// members of a group is Magpie's job.
func (s Settings) ModelFor(task Task) string {
	if s.Tiers != nil {
		if choice := strings.TrimSpace(s.Tiers[string(task)]); choice != "" {
			return choice
		}
	}
	return strings.TrimSpace(s.Model)
}

// LoadSettings reads the stored settings. A missing file is the default state,
// but a damaged file is an error: silently starting over would drop the user's
// gateway address and look like Magpie had disappeared.
func LoadSettings(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultSettings(), nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("magpie: 读取连接设置失败: %w", err)
	}
	var stored Settings
	if err := json.Unmarshal(data, &stored); err != nil {
		return Settings{}, errors.New("magpie: 连接设置文件已损坏,请重新填写网关地址")
	}
	if stored.Protocol == "" {
		stored.Protocol = Chat
	}
	if stored.BaseURL == "" {
		stored.BaseURL = "http://" + DefaultAddr
	}
	if stored.Tiers == nil {
		stored.Tiers = map[string]string{}
	}
	if err := stored.Validate(); err != nil {
		return Settings{}, err
	}
	return stored, nil
}

// SaveSettings writes the settings atomically, so an interrupted write cannot
// leave half a file behind.
func SaveSettings(path string, s Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if s.BaseURL == "" {
		s.BaseURL = "http://" + DefaultAddr
	}
	if s.Protocol == "" {
		s.Protocol = Chat
	}
	if s.Tiers == nil {
		s.Tiers = map[string]string{}
	}
	// Sorted keys keep the file stable between runs, which matters when a
	// person compares two machines' settings by eye.
	data, err := json.MarshalIndent(orderedSettings(s), "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("magpie: 保存连接设置失败: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("magpie: 保存连接设置失败: %w", err)
	}
	return nil
}

// orderedSettings returns a value whose JSON keys are predictable. encoding/json
// already sorts map keys; this keeps the tier map sorted as well and drops
// empty tier values so the file shows only real choices.
func orderedSettings(s Settings) Settings {
	for tier, choice := range s.Tiers {
		if strings.TrimSpace(choice) == "" {
			delete(s.Tiers, tier)
		}
	}
	tiers := make(map[string]string, len(s.Tiers))
	keys := make([]string, 0, len(s.Tiers))
	for tier := range s.Tiers {
		keys = append(keys, tier)
	}
	sort.Strings(keys)
	for _, tier := range keys {
		tiers[tier] = s.Tiers[tier]
	}
	s.Tiers = tiers
	return s
}
