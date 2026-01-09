package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xilu0/cc-debug/internal/logger"
)

const MockResponse = `I'm Claude, an AI assistant made by Anthropic. I'm running as Claude Code, Anthropic's CLI tool for software engineering tasks. I'm powered by the Claude Opus 4.5 model.

I can help you with:
- Reading, writing, and editing code
- Running bash commands
- Exploring codebases
- Debugging issues
- Planning implementations
- Answering questions about your projects

What can I help you with?`

type Handler struct {
	logger *logger.Logger
}

func NewHandler(l *logger.Logger) *Handler {
	return &Handler{logger: l}
}

func (h *Handler) Proxy(c *gin.Context) {
	start := time.Now()

	// Read request body
	bodyBytes, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var bodyJSON interface{}
	json.Unmarshal(bodyBytes, &bodyJSON)

	log := &logger.RequestLog{
		Timestamp: start,
		Method:    c.Request.Method,
		Path:      c.Request.URL.Path,
		Headers:   c.Request.Header,
		Body:      bodyJSON,
	}

	// Check if streaming
	isStream := strings.Contains(string(bodyBytes), `"stream":true`) || strings.Contains(string(bodyBytes), `"stream": true`)

	if isStream {
		h.handleMockStream(c, log)
	} else {
		h.handleMockResponse(c, log)
	}

	log.Duration = time.Since(start).String()
	h.logger.Log(log)
}

func (h *Handler) handleMockResponse(c *gin.Context, log *logger.RequestLog) {
	resp := map[string]interface{}{
		"id":    "msg_mock_" + fmt.Sprintf("%d", time.Now().UnixNano()),
		"type":  "message",
		"role":  "assistant",
		"model": "claude-opus-4-5-20251101",
		"content": []map[string]string{
			{"type": "text", "text": MockResponse},
		},
		"stop_reason": "end_turn",
	}
	log.Response = resp
	c.JSON(200, resp)
}

func (h *Handler) handleMockStream(c *gin.Context, log *logger.RequestLog) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	msgID := "msg_mock_" + fmt.Sprintf("%d", time.Now().UnixNano())

	// message_start
	startEvent := map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":      msgID,
			"type":    "message",
			"role":    "assistant",
			"model":   "claude-opus-4-5-20251101",
			"content": []interface{}{},
		},
	}
	h.writeSSE(c, startEvent, log)

	// content_block_start
	blockStart := map[string]interface{}{
		"type":  "content_block_start",
		"index": 0,
		"content_block": map[string]string{
			"type": "text",
			"text": "",
		},
	}
	h.writeSSE(c, blockStart, log)

	// content_block_delta - send text in chunks
	for _, char := range MockResponse {
		delta := map[string]interface{}{
			"type":  "content_block_delta",
			"index": 0,
			"delta": map[string]string{
				"type": "text_delta",
				"text": string(char),
			},
		}
		h.writeSSE(c, delta, log)
		time.Sleep(5 * time.Millisecond)
	}

	// content_block_stop
	blockStop := map[string]interface{}{
		"type":  "content_block_stop",
		"index": 0,
	}
	h.writeSSE(c, blockStop, log)

	// message_delta
	msgDelta := map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]string{
			"stop_reason": "end_turn",
		},
	}
	h.writeSSE(c, msgDelta, log)

	// message_stop
	msgStop := map[string]interface{}{"type": "message_stop"}
	h.writeSSE(c, msgStop, log)
}

func (h *Handler) writeSSE(c *gin.Context, data interface{}, log *logger.RequestLog) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", data.(map[string]interface{})["type"], jsonData)
	c.Writer.Flush()
	h.logger.LogStream(log, data)
}
