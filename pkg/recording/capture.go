package recording

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/VulpineOS/foxbridge/pkg/cdp"
)

type Entry struct {
	Seq       int          `json:"seq"`
	Direction string       `json:"direction"`
	Message   *cdp.Message `json:"message"`
}

type Recorder struct {
	mu   sync.Mutex
	file *os.File
	enc  *json.Encoder
	seq  int
}

func NewRecorder(path string) (*Recorder, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create recording directory: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create recording file: %w", err)
	}
	enc := json.NewEncoder(file)
	enc.SetEscapeHTML(false)
	return &Recorder{
		file: file,
		enc:  enc,
	}, nil
}

func (r *Recorder) Record(direction string, msg *cdp.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	entry := Entry{
		Seq:       r.seq,
		Direction: direction,
		Message:   cloneMessage(msg),
	}
	if err := r.enc.Encode(entry); err != nil {
		return fmt.Errorf("encode entry: %w", err)
	}
	return nil
}

func (r *Recorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.file.Close()
}

func Load(path string) ([]Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open recording file: %w", err)
	}
	defer file.Close()

	entries := []Entry{}
	dec := json.NewDecoder(file)
	for {
		var entry Entry
		if err := dec.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode recording entry: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func cloneMessage(msg *cdp.Message) *cdp.Message {
	if msg == nil {
		return nil
	}
	clone := *msg
	clone.Params = append([]byte(nil), msg.Params...)
	clone.Result = append([]byte(nil), msg.Result...)
	if msg.Error != nil {
		errCopy := *msg.Error
		clone.Error = &errCopy
	}
	return &clone
}
