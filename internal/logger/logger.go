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
	Timestamp time.Time           `json:"timestamp"`
	Method    string              `json:"method"`
	Path      string              `json:"path"`
	Headers   map[string][]string `json:"headers"`
	Body      interface{}         `json:"body"`
}

type ResponseLog struct {
	Timestamp    time.Time           `json:"timestamp"`
	StatusCode   int                 `json:"status_code"`
	Headers      map[string][]string `json:"headers"`
	Body         interface{}         `json:"body,omitempty"`
	StreamChunks []interface{}       `json:"stream_chunks,omitempty"`
	Duration     string              `json:"duration"`
}

type logEntry struct {
	Type     string       `json:"type"`
	Request  *RequestLog  `json:"request,omitempty"`
	Response *ResponseLog `json:"response,omitempty"`
	Event    *EventLog    `json:"event,omitempty"`
}

type EventLog struct {
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

type Logger struct {
	mode   OutputMode
	outDir string
	files  map[string]*os.File
	mu     sync.Mutex
}

func New(mode OutputMode, outDir string) *Logger {
	if mode == ModeJSON && outDir != "" {
		os.MkdirAll(outDir, 0755)
	}
	return &Logger{mode: mode, outDir: outDir, files: make(map[string]*os.File)}
}

func (l *Logger) Mode() OutputMode {
	return l.mode
}

// getFile returns the JSONL file for a given session ID, creating it if needed.
func (l *Logger) getFile(sessionID string) (*os.File, error) {
	if f, ok := l.files[sessionID]; ok {
		return f, nil
	}
	dir := filepath.Join(l.outDir, sessionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	filename := filepath.Join(dir, "debug.jsonl")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	l.files[sessionID] = f
	return f, nil
}

func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, f := range l.files {
		f.Close()
	}
}

func (l *Logger) LogRequest(sessionID string, req *RequestLog) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.mode == ModeConsole {
		data, err := json.MarshalIndent(req, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("[REQ] %s\n", string(data))
		return nil
	}

	f, err := l.getFile(sessionID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(logEntry{Type: "request", Request: req})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

func (l *Logger) LogStreamEvent(sessionID string, event *EventLog) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.mode == ModeConsole {
		data, err := json.Marshal(event.Data)
		if err != nil {
			return err
		}
		fmt.Printf("[STREAM] %s\n", string(data))
		return nil
	}

	f, err := l.getFile(sessionID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(logEntry{Type: "stream_event", Event: event})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

func (l *Logger) LogResponse(sessionID string, resp *ResponseLog) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.mode == ModeConsole {
		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("[RESP] %s\n", string(data))
		return nil
	}

	f, err := l.getFile(sessionID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(logEntry{Type: "response", Response: resp})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}
