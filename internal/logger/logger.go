package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type OutputMode string

const (
	ModeConsole OutputMode = "console"
	ModeJSON    OutputMode = "json"
)

type RequestLog struct {
	Timestamp   time.Time              `json:"timestamp"`
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	Headers     map[string][]string    `json:"headers"`
	Body        interface{}            `json:"body"`
	Response    interface{}            `json:"response,omitempty"`
	StreamChunks []interface{}         `json:"stream_chunks,omitempty"`
	Duration    string                 `json:"duration,omitempty"`
}

type Logger struct {
	mode    OutputMode
	outDir  string
	mu      sync.Mutex
}

func New(mode OutputMode, outDir string) *Logger {
	if mode == ModeJSON && outDir != "" {
		os.MkdirAll(outDir, 0755)
	}
	return &Logger{mode: mode, outDir: outDir}
}

func (l *Logger) Log(log *RequestLog) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return err
	}

	if l.mode == ModeConsole {
		fmt.Println(string(data))
		return nil
	}

	filename := filepath.Join(l.outDir, fmt.Sprintf("%d.json", log.Timestamp.UnixNano()))
	return os.WriteFile(filename, data, 0644)
}

func (l *Logger) LogStream(log *RequestLog, chunk interface{}) {
	l.mu.Lock()
	log.StreamChunks = append(log.StreamChunks, chunk)
	l.mu.Unlock()

	if l.mode == ModeConsole {
		data, _ := json.Marshal(chunk)
		fmt.Printf("[STREAM] %s\n", string(data))
	}
}
