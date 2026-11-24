package adapt

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/magpie"
)

const (
	// maxRefImageBytes caps local style/character reference images loaded into memory.
	maxRefImageBytes = 20 << 20
	// maxPanelImageBytes caps rendered panel images persisted to disk.
	maxPanelImageBytes = 50 << 20
)

// ErrUnsafePath is returned when an identifier or relative path attempts to
// escape the profile-scoped adapt directory.
var ErrUnsafePath = errors.New("adapt: unsafe path outside profile directory")

// Store persists adaptation workspaces, storyboards, multi-chunk checkpoints,
// non-secret settings, and rendered panel images under <dataDir>/adapt.
type Store struct {
	root string
	mu   sync.Mutex
}

// OpenStore initializes profile-scoped adaptation persistence under
// filepath.Join(dataDir, "adapt").
func OpenStore(dataDir string) (*Store, error) {
	if strings.TrimSpace(dataDir) == "" {
		return nil, errors.New("adapt: data directory is required")
	}
	absData, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("adapt: resolve data directory: %w", err)
	}
	root := filepath.Join(absData, "adapt")
	if err := os.MkdirAll(filepath.Join(root, "projects"), 0o700); err != nil {
		return nil, fmt.Errorf("adapt: create adapt directory: %w", err)
	}
	return &Store{root: root}, nil
}

// Root returns the absolute <dataDir>/adapt root directory.
func (s *Store) Root() string { return s.root }

// ProjectID returns a deterministic, filesystem-safe project ID for one novel.
func ProjectID(novelSourceKey, novelID string) string {
	src := strings.TrimSpace(novelSourceKey)
	nid := strings.TrimSpace(novelID)
	raw := src + "\x00" + nid
	sum := sha256.Sum256([]byte(raw))
	prefix := sanitizeASCII(src+"_"+nid, 24)
	if prefix == "" {
		prefix = "novel"
	}
	return "proj_" + prefix + "_" + hex.EncodeToString(sum[:8])
}

type persistedProject struct {
	ID             string                   `json:"id"`
	NovelSourceKey string                   `json:"novelSourceKey,omitempty"`
	NovelID        string                   `json:"novelId,omitempty"`
	Title          string                   `json:"title"`
	StyleRef       *core.AdaptStyleRef      `json:"styleRef,omitempty"`
	Characters     core.AdaptCharacterSheet `json:"characters"`
	Budget         core.AdaptBudget         `json:"budget,omitempty"`
	Usage          core.AdaptUsage          `json:"usage"`
	UpdatedAt      time.Time                `json:"updatedAt,omitempty"`
}

// LoadOrCreateProject loads an existing novel workspace or initializes a new
// in-memory workspace for (novelSourceKey, novelID, title).
func (s *Store) LoadOrCreateProject(novelSourceKey, novelID, title string) (*Workspace, error) {
	if strings.TrimSpace(novelSourceKey) == "" || strings.TrimSpace(novelID) == "" {
		return nil, errors.New("adapt: novel source key and novel ID are required")
	}
	projectID := ProjectID(novelSourceKey, novelID)
	ws, err := s.LoadProject(projectID)
	if err == nil {
		if title != "" && ws.Project.Title == "" {
			ws.Project.Title = title
		}
		if ws.Project.NovelSourceKey == "" {
			ws.Project.NovelSourceKey = novelSourceKey
		}
		if ws.Project.NovelID == "" {
			ws.Project.NovelID = novelID
		}
		return ws, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if strings.TrimSpace(title) == "" {
		title = novelID
	}
	return &Workspace{
		Project: &core.AdaptProject{
			ID:             projectID,
			NovelSourceKey: novelSourceKey,
			NovelID:        novelID,
			Title:          title,
			Storyboards:    map[string]*core.AdaptStoryboard{},
		},
		Checkpoints: map[string]*core.AdaptCheckpoint{},
	}, nil
}

// LoadProject reads project metadata, all saved chapter storyboards, and all
// saved multi-chunk checkpoints from disk.
func (s *Store) LoadProject(projectID string) (*Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadProjectLocked(projectID)
}

func (s *Store) loadProjectLocked(projectID string) (*Workspace, error) {
	projDir, err := s.projectDir(projectID)
	if err != nil {
		return nil, err
	}
	metaPath, err := safeJoin(projDir, "project.json")
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}
	var stored persistedProject
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, fmt.Errorf("adapt: corrupt project metadata: %w", err)
	}
	if stored.ID == "" {
		stored.ID = projectID
	}
	proj := &core.AdaptProject{
		ID:             stored.ID,
		NovelSourceKey: stored.NovelSourceKey,
		NovelID:        stored.NovelID,
		Title:          stored.Title,
		StyleRef:       stored.StyleRef,
		Characters:     stored.Characters,
		Storyboards:    map[string]*core.AdaptStoryboard{},
		Budget:         stored.Budget,
		Usage:          stored.Usage,
	}

	sbDir, err := safeJoin(projDir, "storyboards")
	if err != nil {
		return nil, err
	}
	if entries, err := os.ReadDir(sbDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			p, err := safeJoin(sbDir, entry.Name())
			if err != nil {
				continue
			}
			raw, err := os.ReadFile(p)
			if err != nil {
				return nil, fmt.Errorf("adapt: read storyboard %s: %w", entry.Name(), err)
			}
			var sb core.AdaptStoryboard
			if err := json.Unmarshal(raw, &sb); err != nil {
				return nil, fmt.Errorf("adapt: decode storyboard %s: %w", entry.Name(), err)
			}
			if sb.ChapterID != "" {
				proj.Storyboards[sb.ChapterID] = &sb
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	checkpoints := map[string]*core.AdaptCheckpoint{}
	cpDir, err := safeJoin(projDir, "checkpoints")
	if err != nil {
		return nil, err
	}
	if entries, err := os.ReadDir(cpDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			p, err := safeJoin(cpDir, entry.Name())
			if err != nil {
				continue
			}
			raw, err := os.ReadFile(p)
			if err != nil {
				return nil, fmt.Errorf("adapt: read checkpoint %s: %w", entry.Name(), err)
			}
			var cp core.AdaptCheckpoint
			if err := json.Unmarshal(raw, &cp); err != nil {
				return nil, fmt.Errorf("adapt: decode checkpoint %s: %w", entry.Name(), err)
			}
			if cp.ChapterID != "" {
				checkpoints[cp.ChapterID] = &cp
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return &Workspace{
		Project:     proj,
		Checkpoints: checkpoints,
		UpdatedAt:   stored.UpdatedAt,
	}, nil
}

// SaveProject persists project metadata, character sheet, budget/usage,
// storyboards, and active checkpoints atomically.
func (s *Store) SaveProject(ws *Workspace) error {
	if ws == nil || ws.Project == nil {
		return errors.New("adapt: workspace project is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveProjectLocked(ws)
}

func (s *Store) saveProjectLocked(ws *Workspace) error {
	proj := ws.Project
	if strings.TrimSpace(proj.ID) == "" {
		return errors.New("adapt: project ID is required")
	}
	projDir, err := s.projectDir(proj.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(projDir, 0o700); err != nil {
		return err
	}
	now := time.Now().UTC()
	ws.UpdatedAt = now
	meta := persistedProject{
		ID:             proj.ID,
		NovelSourceKey: proj.NovelSourceKey,
		NovelID:        proj.NovelID,
		Title:          proj.Title,
		StyleRef:       proj.StyleRef,
		Characters:     proj.Characters,
		Budget:         proj.Budget,
		Usage:          proj.Usage,
		UpdatedAt:      now,
	}
	metaPath, err := safeJoin(projDir, "project.json")
	if err != nil {
		return err
	}
	if err := writeJSONAtomic(metaPath, meta); err != nil {
		return err
	}

	for chapterID, sb := range proj.Storyboards {
		if sb == nil {
			continue
		}
		if sb.ChapterID == "" {
			sb.ChapterID = chapterID
		}
		if err := s.saveStoryboardLocked(projDir, sb); err != nil {
			return err
		}
	}

	// Sync checkpoints: save active ones and remove checkpoints for chapters
	// no longer present in ws.Checkpoints.
	cpDir, err := safeJoin(projDir, "checkpoints")
	if err != nil {
		return err
	}
	keepFiles := make(map[string]struct{}, len(ws.Checkpoints))
	for chapterID, cp := range ws.Checkpoints {
		if cp == nil {
			continue
		}
		if cp.ChapterID == "" {
			cp.ChapterID = chapterID
		}
		fileName, err := s.saveCheckpointLocked(projDir, cp)
		if err != nil {
			return err
		}
		keepFiles[fileName] = struct{}{}
	}
	if entries, err := os.ReadDir(cpDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			if _, keep := keepFiles[entry.Name()]; !keep {
				if p, err := safeJoin(cpDir, entry.Name()); err == nil {
					_ = os.Remove(p)
				}
			}
		}
	}
	return nil
}

// SaveStoryboard persists one chapter storyboard inside the project directory.
func (s *Store) SaveStoryboard(projectID string, sb *core.AdaptStoryboard) error {
	if sb == nil || strings.TrimSpace(sb.ChapterID) == "" {
		return errors.New("adapt: storyboard with chapter ID is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	projDir, err := s.projectDir(projectID)
	if err != nil {
		return err
	}
	return s.saveStoryboardLocked(projDir, sb)
}

func (s *Store) saveStoryboardLocked(projDir string, sb *core.AdaptStoryboard) error {
	slug, err := safeSegment("chapter", sb.ChapterID)
	if err != nil {
		return err
	}
	sbDir, err := safeJoin(projDir, "storyboards")
	if err != nil {
		return err
	}
	target, err := safeJoin(sbDir, slug+".json")
	if err != nil {
		return err
	}
	return writeJSONAtomic(target, sb)
}

// LoadStoryboard loads one chapter storyboard from disk.
func (s *Store) LoadStoryboard(projectID, chapterID string) (*core.AdaptStoryboard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	projDir, err := s.projectDir(projectID)
	if err != nil {
		return nil, err
	}
	slug, err := safeSegment("chapter", chapterID)
	if err != nil {
		return nil, err
	}
	target, err := safeJoin(projDir, "storyboards", slug+".json")
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}
	var sb core.AdaptStoryboard
	if err := json.Unmarshal(raw, &sb); err != nil {
		return nil, fmt.Errorf("adapt: decode storyboard: %w", err)
	}
	return &sb, nil
}

// SaveCheckpoint persists one chapter's multi-chunk AdaptCheckpoint to disk.
func (s *Store) SaveCheckpoint(projectID string, cp *core.AdaptCheckpoint) error {
	if cp == nil || strings.TrimSpace(cp.ChapterID) == "" {
		return errors.New("adapt: checkpoint with chapter ID is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	projDir, err := s.projectDir(projectID)
	if err != nil {
		return err
	}
	_, err = s.saveCheckpointLocked(projDir, cp)
	return err
}

func (s *Store) saveCheckpointLocked(projDir string, cp *core.AdaptCheckpoint) (string, error) {
	slug, err := safeSegment("chapter", cp.ChapterID)
	if err != nil {
		return "", err
	}
	fileName := slug + ".json"
	cpDir, err := safeJoin(projDir, "checkpoints")
	if err != nil {
		return "", err
	}
	target, err := safeJoin(cpDir, fileName)
	if err != nil {
		return "", err
	}
	if err := writeJSONAtomic(target, cp); err != nil {
		return "", err
	}
	return fileName, nil
}

// LoadCheckpoint reads one chapter's persisted checkpoint.
func (s *Store) LoadCheckpoint(projectID, chapterID string) (*core.AdaptCheckpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	projDir, err := s.projectDir(projectID)
	if err != nil {
		return nil, err
	}
	slug, err := safeSegment("chapter", chapterID)
	if err != nil {
		return nil, err
	}
	target, err := safeJoin(projDir, "checkpoints", slug+".json")
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}
	var cp core.AdaptCheckpoint
	if err := json.Unmarshal(raw, &cp); err != nil {
		return nil, fmt.Errorf("adapt: decode checkpoint: %w", err)
	}
	return &cp, nil
}

// DeleteCheckpoint removes a persisted chapter checkpoint if present.
func (s *Store) DeleteCheckpoint(projectID, chapterID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	projDir, err := s.projectDir(projectID)
	if err != nil {
		return err
	}
	slug, err := safeSegment("chapter", chapterID)
	if err != nil {
		return err
	}
	target, err := safeJoin(projDir, "checkpoints", slug+".json")
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// SavePanelImage writes one rendered panel image atomically inside the
// project's panels directory and returns a safe POSIX-relative refID.
func (s *Store) SavePanelImage(projectID string, meta core.AdaptPanelImageMeta, mimeType string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("adapt: empty panel image data")
	}
	if len(data) > maxPanelImageBytes {
		return "", fmt.Errorf("adapt: panel image exceeds %d bytes", maxPanelImageBytes)
	}
	if strings.TrimSpace(projectID) == "" {
		projectID = meta.ProjectID
	}
	if strings.TrimSpace(meta.ChapterID) == "" || strings.TrimSpace(meta.PanelID) == "" {
		return "", errors.New("adapt: panel image metadata requires chapter and panel IDs")
	}

	resolvedMIME := normalizeImageMIME(mimeType, data)
	ext, ok := imageExtForMIME(resolvedMIME)
	if !ok {
		return "", fmt.Errorf("adapt: unsupported panel image MIME type %q", resolvedMIME)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	projDir, err := s.projectDir(projectID)
	if err != nil {
		return "", err
	}
	chapterSlug, err := safeSegment("ch", meta.ChapterID)
	if err != nil {
		return "", err
	}
	sceneID := strings.TrimSpace(meta.SceneID)
	if sceneID == "" {
		sceneID = "scene"
	}
	sceneSlug, err := safeSegment("sc", sceneID)
	if err != nil {
		return "", err
	}
	panelSlug, err := safeSegment("pn", meta.PanelID)
	if err != nil {
		return "", err
	}

	baseName := sceneSlug + "__" + panelSlug
	fileName := baseName + ext
	chapterDir, err := safeJoin(projDir, "panels", chapterSlug)
	if err != nil {
		return "", err
	}
	target, err := safeJoin(chapterDir, fileName)
	if err != nil {
		return "", err
	}
	if err := writeFileAtomic(target, data, 0o600); err != nil {
		return "", fmt.Errorf("adapt: save panel image: %w", err)
	}
	// Clean up stale images with another extension if ForceRerender changed formats.
	for _, otherExt := range []string{".png", ".jpg", ".webp", ".gif"} {
		if otherExt == ext {
			continue
		}
		if stale, err := safeJoin(chapterDir, baseName+otherExt); err == nil {
			_ = os.Remove(stale)
		}
	}
	return "panels/" + chapterSlug + "/" + fileName, nil
}

// ResolvePanelImagePath validates refID and returns the absolute filesystem
// path strictly confined within the project's panels directory.
func (s *Store) ResolvePanelImagePath(projectID, refID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resolvePanelImagePathLocked(projectID, refID)
}

func (s *Store) resolvePanelImagePathLocked(projectID, refID string) (string, error) {
	parts, err := validatePanelRefID(refID)
	if err != nil {
		return "", err
	}
	projDir, err := s.projectDir(projectID)
	if err != nil {
		return "", err
	}
	panelsRoot, err := safeJoin(projDir, "panels")
	if err != nil {
		return "", err
	}
	target, err := safeJoin(panelsRoot, parts[1], parts[2])
	if err != nil {
		return "", err
	}
	if err := verifyNoSymlinkEscape(panelsRoot, target); err != nil {
		return "", err
	}
	return target, nil
}

// ReadPanelImage reads a previously rendered panel image after verifying that
// refID stays inside the project's panels directory.
func (s *Store) ReadPanelImage(projectID, refID string) ([]byte, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	target, err := s.resolvePanelImagePathLocked(projectID, refID)
	if err != nil {
		return nil, "", err
	}
	info, err := os.Lstat(target)
	if err != nil {
		return nil, "", err
	}
	if !info.Mode().IsRegular() {
		return nil, "", ErrUnsafePath
	}
	if info.Size() > maxPanelImageBytes {
		return nil, "", fmt.Errorf("adapt: panel image exceeds %d bytes", maxPanelImageBytes)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return nil, "", err
	}
	return data, normalizeImageMIME("", data), nil
}

// LoadSettings reads non-secret adaptation preferences from <root>/settings.json.
func (s *Store) LoadSettings() (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := safeJoin(s.root, "settings.json")
	if err != nil {
		return Settings{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultSettings(), nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("adapt: read settings: %w", err)
	}
	var out Settings
	if err := json.Unmarshal(data, &out); err != nil {
		return Settings{}, fmt.Errorf("adapt: decode settings: %w", err)
	}
	if out.TextMode == "" {
		out.TextMode = TextModeMagpie
	}
	if out.MagpieTier == "" {
		out.MagpieTier = magpie.TaskComplex
	}
	if out.TextProtocol == "" {
		out.TextProtocol = core.AIProtocolChat
	}
	if out.ImageProvider == "" {
		out.ImageProvider = core.AdaptImageGemini
	}
	return out, nil
}

// SaveSettings writes non-secret adaptation preferences atomically.
func (s *Store) SaveSettings(settings Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := safeJoin(s.root, "settings.json")
	if err != nil {
		return err
	}
	return writeJSONAtomic(path, settings)
}

// LoadImageRef reads a local style or character reference image from path,
// validating that it is a readable regular image file within size limits.
func LoadImageRef(path string, kind core.AdaptImageRefKind, label string) (core.AdaptImageRef, error) {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" || strings.ContainsRune(cleaned, '\x00') {
		return core.AdaptImageRef{}, errors.New("adapt: reference image path is invalid")
	}
	info, err := os.Stat(cleaned)
	if err != nil {
		return core.AdaptImageRef{}, fmt.Errorf("adapt: open reference image: %w", err)
	}
	if !info.Mode().IsRegular() {
		return core.AdaptImageRef{}, errors.New("adapt: reference image path must be a regular file")
	}
	if info.Size() == 0 {
		return core.AdaptImageRef{}, errors.New("adapt: reference image file is empty")
	}
	if info.Size() > maxRefImageBytes {
		return core.AdaptImageRef{}, fmt.Errorf("adapt: reference image exceeds %d bytes", maxRefImageBytes)
	}
	f, err := os.Open(cleaned)
	if err != nil {
		return core.AdaptImageRef{}, fmt.Errorf("adapt: open reference image: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxRefImageBytes+1))
	if err != nil {
		return core.AdaptImageRef{}, fmt.Errorf("adapt: read reference image: %w", err)
	}
	if len(data) == 0 || int64(len(data)) > maxRefImageBytes {
		return core.AdaptImageRef{}, fmt.Errorf("adapt: reference image size invalid (%d bytes)", len(data))
	}
	mimeType := detectImageMIME(data)
	if _, ok := imageExtForMIME(mimeType); !ok {
		return core.AdaptImageRef{}, fmt.Errorf("adapt: unsupported reference image format %q", mimeType)
	}
	sum := sha256.Sum256(data)
	if strings.TrimSpace(label) == "" {
		label = filepath.Base(cleaned)
	}
	return core.AdaptImageRef{
		ID:       "local:" + hex.EncodeToString(sum[:8]),
		Kind:     kind,
		Label:    label,
		MIMEType: mimeType,
		Data:     data,
	}, nil
}

func (s *Store) projectDir(projectID string) (string, error) {
	slug, err := safeSegment("proj", projectID)
	if err != nil {
		return "", err
	}
	return safeJoin(s.root, "projects", slug)
}

func safeSegment(prefix, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%w: empty %s identifier", ErrUnsafePath, prefix)
	}
	sum := sha256.Sum256([]byte(trimmed))
	clean := sanitizeASCII(trimmed, 24)
	if clean == "" {
		clean = prefix
	}
	return clean + "_" + hex.EncodeToString(sum[:8]), nil
}

func sanitizeASCII(raw string, maxLen int) string {
	var b strings.Builder
	b.Grow(min(len(raw), maxLen))
	lastSep := true
	for i := 0; i < len(raw) && b.Len() < maxLen; i++ {
		ch := raw[i]
		isAlphaNum := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
		if isAlphaNum {
			b.WriteByte(ch)
			lastSep = false
		} else if (ch == '-' || ch == '_') && !lastSep {
			b.WriteByte(ch)
			lastSep = true
		}
	}
	out := strings.Trim(b.String(), "-_")
	upper := strings.ToUpper(out)
	if slices.Contains([]string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "LPT1", "LPT2", "LPT3"}, upper) {
		return ""
	}
	return out
}

func validatePanelRefID(refID string) ([]string, error) {
	if refID == "" || strings.ContainsAny(refID, "\\:\x00") {
		return nil, fmt.Errorf("%w: invalid panel ref %q", ErrUnsafePath, refID)
	}
	if strings.HasPrefix(refID, "/") || strings.HasSuffix(refID, "/") {
		return nil, fmt.Errorf("%w: panel ref must be relative", ErrUnsafePath)
	}
	for i := 0; i < len(refID); i++ {
		if refID[i] < 0x20 || refID[i] == 0x7f {
			return nil, fmt.Errorf("%w: control character in panel ref", ErrUnsafePath)
		}
	}
	parts := strings.Split(refID, "/")
	if len(parts) != 3 || parts[0] != "panels" {
		return nil, fmt.Errorf("%w: unexpected panel ref layout", ErrUnsafePath)
	}
	for _, seg := range parts[1:] {
		if seg == "" || seg == "." || seg == ".." || strings.Contains(seg, "..") {
			return nil, fmt.Errorf("%w: unsafe segment in panel ref", ErrUnsafePath)
		}
	}
	return parts, nil
}

func safeJoin(root string, elems ...string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnsafePath, err)
	}
	for _, elem := range elems {
		if elem == "" || elem == "." || elem == ".." {
			return "", fmt.Errorf("%w: invalid path element %q", ErrUnsafePath, elem)
		}
		if strings.ContainsAny(elem, `/\:`+"\x00") || filepath.VolumeName(elem) != "" {
			return "", fmt.Errorf("%w: forbidden characters in %q", ErrUnsafePath, elem)
		}
		for i := 0; i < len(elem); i++ {
			if elem[i] < 0x20 || elem[i] == 0x7f {
				return "", fmt.Errorf("%w: control character in path element", ErrUnsafePath)
			}
		}
	}
	parts := append([]string{absRoot}, elems...)
	target := filepath.Clean(filepath.Join(parts...))
	rel, err := filepath.Rel(absRoot, target)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: path escapes root", ErrUnsafePath)
	}
	return target, nil
}

func verifyNoSymlinkEscape(root, target string) error {
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	realTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	rel, err := filepath.Rel(realRoot, realTarget)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("%w: symlink escapes root", ErrUnsafePath)
	}
	return nil
}

func writeJSONAtomic(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, append(data, '\n'), 0o600)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func detectImageMIME(data []byte) string {
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	detected := http.DetectContentType(data)
	if strings.HasPrefix(detected, "image/") {
		return detected
	}
	return "application/octet-stream"
}

func normalizeImageMIME(declared string, data []byte) string {
	clean := strings.ToLower(strings.TrimSpace(declared))
	if idx := strings.IndexByte(clean, ';'); idx >= 0 {
		clean = strings.TrimSpace(clean[:idx])
	}
	if _, ok := imageExtForMIME(clean); ok {
		return clean
	}
	return detectImageMIME(data)
}

func imageExtForMIME(mimeType string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/png":
		return ".png", true
	case "image/jpeg", "image/jpg":
		return ".jpg", true
	case "image/webp":
		return ".webp", true
	case "image/gif":
		return ".gif", true
	default:
		return "", false
	}
}
