package adapt

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/magpie"
)

// BuildTextBackend builds a runtime TextBackend from non-secret Settings plus
// an in-memory API key (used only in TextModeAPI; ignored in TextModeMagpie).
func (s Settings) BuildTextBackend(dataDir, apiKey string) TextBackend {
	mode := s.TextMode
	if mode == "" {
		mode = TextModeMagpie
	}
	if mode == TextModeMagpie {
		tier := s.MagpieTier
		if tier == "" {
			tier = magpie.TaskComplex
		}
		return TextBackend{
			Mode:               TextModeMagpie,
			MagpieSettingsPath: filepath.Join(dataDir, "magpie.json"),
			MagpieTier:         tier,
			Config: core.AdaptTextConfig{
				Model:           strings.TrimSpace(s.MagpieModel),
				MaxTokens:       s.TextMaxTokens,
				JSONMode:        s.TextJSONMode,
				ReasoningEffort: strings.TrimSpace(s.TextReasoningEffort),
			},
		}
	}
	proto := s.TextProtocol
	if proto == "" {
		proto = core.AIProtocolChat
	}
	return TextBackend{
		Mode: TextModeAPI,
		Config: core.AdaptTextConfig{
			Protocol:         proto,
			BaseURL:          strings.TrimSpace(s.TextBaseURL),
			APIKey:           strings.TrimSpace(apiKey),
			AnthropicVersion: strings.TrimSpace(s.TextAnthropicVersion),
			Model:            strings.TrimSpace(s.TextModel),
			MaxTokens:        s.TextMaxTokens,
			JSONMode:         s.TextJSONMode,
			ReasoningEffort:  strings.TrimSpace(s.TextReasoningEffort),
		},
	}
}

// BuildImageBackend builds a runtime ImageBackend from non-secret Settings plus
// an in-memory image provider API key.
func (s Settings) BuildImageBackend(apiKey string) ImageBackend {
	provider := s.ImageProvider
	if provider == "" {
		provider = core.AdaptImageGemini
	}
	var charPaths map[string][]string
	if len(s.CharacterRefPaths) > 0 {
		charPaths = make(map[string][]string, len(s.CharacterRefPaths))
		for k, paths := range s.CharacterRefPaths {
			charPaths[k] = append([]string(nil), paths...)
		}
	}
	return ImageBackend{
		Config: core.AdaptImageConfig{
			Provider:       provider,
			BaseURL:        strings.TrimSpace(s.ImageBaseURL),
			APIKey:         strings.TrimSpace(apiKey),
			Model:          strings.TrimSpace(s.ImageModel),
			Size:           strings.TrimSpace(s.ImageSize),
			Quality:        strings.TrimSpace(s.ImageQuality),
			OutputFormat:   strings.TrimSpace(s.ImageOutputFormat),
			ResponseFormat: strings.TrimSpace(s.ImageResponseFormat),
		},
		AspectRatio:           strings.TrimSpace(s.ImageAspectRatio),
		Size:                  strings.TrimSpace(s.ImageSize),
		FeedbackCharacterRefs: s.FeedbackCharacterRefs,
		StyleRefPaths:         append([]string(nil), s.StyleRefPaths...),
		CharacterRefPaths:     charPaths,
	}
}

// ResolveTextConfig turns a TextBackend into a Core AdaptTextConfig and returns
// any secrets that must be redacted from logs or persisted error text.
func ResolveTextConfig(backend TextBackend, dataDir string) (core.AdaptTextConfig, []string, error) {
	mode := backend.Mode
	if mode == "" {
		if strings.TrimSpace(backend.Config.APIKey) != "" || strings.TrimSpace(backend.Config.BaseURL) != "" {
			mode = TextModeAPI
		} else {
			mode = TextModeMagpie
		}
	}

	switch mode {
	case TextModeMagpie:
		var ms magpie.Settings
		if backend.MagpieSettings != nil {
			ms = *backend.MagpieSettings
		} else {
			path := strings.TrimSpace(backend.MagpieSettingsPath)
			if path == "" && strings.TrimSpace(dataDir) != "" {
				path = filepath.Join(dataDir, "magpie.json")
			}
			if path == "" {
				ms = magpie.DefaultSettings()
			} else {
				loaded, err := magpie.LoadSettings(path)
				if err != nil {
					return core.AdaptTextConfig{}, nil, err
				}
				ms = loaded
			}
		}
		client, err := magpie.New(ms.ClientConfig())
		if err != nil {
			return core.AdaptTextConfig{}, nil, err
		}
		tier := backend.MagpieTier
		if tier == "" {
			tier = magpie.TaskComplex
		}
		model := strings.TrimSpace(backend.Config.Model)
		if model == "" {
			model = ms.ModelFor(tier)
		}
		if model == "" && tier != magpie.TaskNormal {
			model = ms.ModelFor(magpie.TaskNormal)
		}
		if model == "" {
			return core.AdaptTextConfig{}, nil, magpie.ErrNoModel
		}
		token := strings.TrimSpace(ms.Token)
		var secrets []string
		if token == "" {
			token = "magpie"
		} else if token != "magpie" {
			secrets = append(secrets, token)
		}
		proto := core.AIProtocol(client.Protocol())
		if backend.Config.Protocol != "" {
			proto = backend.Config.Protocol
		}
		return core.AdaptTextConfig{
			Protocol:         proto,
			BaseURL:          client.BaseURL() + "/v1",
			APIKey:           token,
			AnthropicVersion: strings.TrimSpace(backend.Config.AnthropicVersion),
			Partition:        strings.TrimSpace(backend.Config.Partition),
			Model:            model,
			MaxTokens:        backend.Config.MaxTokens,
			Temperature:      backend.Config.Temperature,
			JSONMode:         backend.Config.JSONMode,
			ReasoningEffort:  strings.TrimSpace(backend.Config.ReasoningEffort),
		}, secrets, nil

	case TextModeAPI:
		cfg := backend.Config
		if cfg.Protocol == "" {
			cfg.Protocol = core.AIProtocolChat
		}
		cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
		cfg.APIKey = strings.TrimSpace(cfg.APIKey)
		cfg.Model = strings.TrimSpace(cfg.Model)
		cfg.AnthropicVersion = strings.TrimSpace(cfg.AnthropicVersion)
		cfg.Partition = strings.TrimSpace(cfg.Partition)
		cfg.ReasoningEffort = strings.TrimSpace(cfg.ReasoningEffort)
		if cfg.APIKey == "" {
			return core.AdaptTextConfig{}, nil, errors.New("adapt: 请填写文本模型 API Key")
		}
		if cfg.Model == "" {
			return core.AdaptTextConfig{}, nil, errors.New("adapt: 请填写文本模型名称")
		}
		return cfg, []string{cfg.APIKey}, nil

	default:
		return core.AdaptTextConfig{}, nil, fmt.Errorf("adapt: unsupported text backend mode %q", backend.Mode)
	}
}

// ResolveImageConfig validates and normalizes an ImageBackend into Core's
// AdaptImageConfig and returns secrets to redact from errors.
func ResolveImageConfig(backend ImageBackend) (core.AdaptImageConfig, []string, error) {
	cfg := backend.Config
	cfg.Provider = core.AdaptImageProvider(strings.ToLower(strings.TrimSpace(string(cfg.Provider))))
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.Model = strings.TrimSpace(cfg.Model)
	cfg.Partition = strings.TrimSpace(cfg.Partition)
	cfg.Size = strings.TrimSpace(cfg.Size)
	cfg.Quality = strings.TrimSpace(cfg.Quality)
	cfg.OutputFormat = strings.TrimSpace(cfg.OutputFormat)
	cfg.ResponseFormat = strings.TrimSpace(cfg.ResponseFormat)

	switch cfg.Provider {
	case core.AdaptImageGemini, core.AdaptImageOpenAI:
	default:
		return core.AdaptImageConfig{}, nil, fmt.Errorf("adapt: 不支持的生图供应商 %q（仅支持 gemini 或 openai）", backend.Config.Provider)
	}
	if cfg.APIKey == "" {
		return core.AdaptImageConfig{}, nil, errors.New("adapt: 请填写生图 API Key")
	}
	if cfg.Model == "" {
		return core.AdaptImageConfig{}, nil, errors.New("adapt: 请填写生图模型名称")
	}
	return cfg, []string{cfg.APIKey}, nil
}

// LoadRenderReferences combines inline reference images and local reference
// image files configured on backend.
func LoadRenderReferences(backend ImageBackend) ([]core.AdaptImageRef, map[string][]core.AdaptImageRef, error) {
	var styleRefs []core.AdaptImageRef
	for _, ref := range backend.StyleReferences {
		if ref.Kind == "" {
			ref.Kind = core.AdaptImageRefStyle
		}
		styleRefs = append(styleRefs, ref)
	}
	for _, path := range backend.StyleRefPaths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		ref, err := LoadImageRef(path, core.AdaptImageRefStyle, "")
		if err != nil {
			return nil, nil, fmt.Errorf("adapt: load style reference %q: %w", filepath.Base(path), err)
		}
		styleRefs = append(styleRefs, ref)
	}

	var charRefs map[string][]core.AdaptImageRef
	if len(backend.CharacterReferences) > 0 || len(backend.CharacterRefPaths) > 0 {
		charRefs = make(map[string][]core.AdaptImageRef)
		for key, list := range backend.CharacterReferences {
			for _, ref := range list {
				if ref.Kind == "" {
					ref.Kind = core.AdaptImageRefCharacter
				}
				charRefs[key] = append(charRefs[key], ref)
			}
		}
		for key, paths := range backend.CharacterRefPaths {
			for _, path := range paths {
				if strings.TrimSpace(path) == "" {
					continue
				}
				ref, err := LoadImageRef(path, core.AdaptImageRefCharacter, key)
				if err != nil {
					return nil, nil, fmt.Errorf("adapt: load character reference for %q: %w", key, err)
				}
				charRefs[key] = append(charRefs[key], ref)
			}
		}
	}
	return styleRefs, charRefs, nil
}
