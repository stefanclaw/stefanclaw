package session

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/stefanclaw/stefanclaw/internal/provider"
)

// Session represents a conversation session.
type Session struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store defines the interface for session persistence.
type Store interface {
	Create(title, model string) (*Session, error)
	Get(id string) (*Session, error)
	List() ([]*Session, error)
	Append(sessionID string, msg provider.Message) error
	Delete(id string) error
	Current() (*Session, error)
	SetCurrent(id string) error
	LoadTranscript(sessionID string) ([]provider.Message, error)
}

// FileStore implements Store using the filesystem.
type FileStore struct {
	baseDir string
}

// NewFileStore creates a new FileStore rooted at the given directory.
func NewFileStore(baseDir string) *FileStore {
	return &FileStore{baseDir: baseDir}
}

func (fs *FileStore) sessionDir(id string) string {
	return filepath.Join(fs.baseDir, id)
}

func (fs *FileStore) metaPath(id string) string {
	return filepath.Join(fs.sessionDir(id), "meta.json")
}

func (fs *FileStore) transcriptPath(id string) string {
	return filepath.Join(fs.sessionDir(id), "transcript.jsonl")
}

func (fs *FileStore) currentPath() string {
	return filepath.Join(fs.baseDir, ".current")
}

func generateID() string {
	now := time.Now()
	chars := "abcdefghijklmnopqrstuvwxyz0123456789"
	suffix := make([]byte, 4)
	for i := range suffix {
		suffix[i] = chars[rand.Intn(len(chars))]
	}
	return fmt.Sprintf("%s-%s", now.Format("20060102-150405"), string(suffix))
}

// Create starts a new session.
func (fs *FileStore) Create(title, model string) (*Session, error) {
	s := &Session{
		ID:        generateID(),
		Title:     title,
		Model:     model,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	dir := fs.sessionDir(s.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating session directory: %w", err)
	}

	if err := fs.saveMeta(s); err != nil {
		return nil, err
	}

	return s, nil
}

func (fs *FileStore) saveMeta(s *Session) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session meta: %w", err)
	}
	return os.WriteFile(fs.metaPath(s.ID), data, 0o644)
}

// Get retrieves a session by ID.
func (fs *FileStore) Get(id string) (*Session, error) {
	data, err := os.ReadFile(fs.metaPath(id))
	if err != nil {
		return nil, fmt.Errorf("reading session %s: %w", id, err)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("decoding session %s: %w", id, err)
	}

	return &s, nil
}

// List returns all sessions, sorted by creation time (newest first).
func (fs *FileStore) List() ([]*Session, error) {
	entries, err := os.ReadDir(fs.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing sessions: %w", err)
	}

	var sessions []*Session
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		s, err := fs.Get(entry.Name())
		if err != nil {
			continue // skip invalid sessions
		}
		sessions = append(sessions, s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	return sessions, nil
}

// Append adds a message to the session's transcript.
func (fs *FileStore) Append(sessionID string, msg provider.Message) error {
	return AppendMessage(fs.transcriptPath(sessionID), msg)
}

// LoadTranscript reads all messages from a session's transcript.
func (fs *FileStore) LoadTranscript(sessionID string) ([]provider.Message, error) {
	return ReadTranscript(fs.transcriptPath(sessionID))
}

// Delete removes a session and all its data.
func (fs *FileStore) Delete(id string) error {
	return os.RemoveAll(fs.sessionDir(id))
}

// Current returns the current active session.
func (fs *FileStore) Current() (*Session, error) {
	data, err := os.ReadFile(fs.currentPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	id := string(data)
	return fs.Get(id)
}

// SetCurrent sets the current active session ID.
func (fs *FileStore) SetCurrent(id string) error {
	if err := os.MkdirAll(fs.baseDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(fs.currentPath(), []byte(id), 0o644)
}

// UpdateTitle changes the session's title.
func (fs *FileStore) UpdateTitle(id, title string) error {
	s, err := fs.Get(id)
	if err != nil {
		return err
	}
	s.Title = title
	s.UpdatedAt = time.Now()
	return fs.saveMeta(s)
}
