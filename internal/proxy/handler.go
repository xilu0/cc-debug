package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xilu0/cc-debug/internal/logger"
)

// extractSessionID extracts the session ID from body.metadata.user_id.
// Format: "user_..._session_<uuid>"
func extractSessionID(bodyJSON interface{}) string {
	body, ok := bodyJSON.(map[string]interface{})
	if !ok {
		return "unknown"
	}
	metadata, ok := body["metadata"].(map[string]interface{})
	if !ok {
		return "unknown"
	}
	userID, ok := metadata["user_id"].(string)
	if !ok {
		return "unknown"
	}
	const sep = "_session_"
	idx := strings.LastIndex(userID, sep)
	if idx == -1 {
		return "unknown"
	}
	return userID[idx+len(sep):]
}

type Handler struct {
	logger    *logger.Logger
	targetURL string
	client    *http.Client
}

func NewHandler(l *logger.Logger, targetURL string) *Handler {
	return &Handler{
		logger:    l,
		targetURL: strings.TrimRight(targetURL, "/"),
		client: &http.Client{
			Timeout: 0, // no timeout for streaming
		},
	}
}

func (h *Handler) Proxy(c *gin.Context) {
	start := time.Now()

	// Read request body
	bodyBytes, _ := io.ReadAll(c.Request.Body)

	var bodyJSON interface{}
	json.Unmarshal(bodyBytes, &bodyJSON)

	sessionID := extractSessionID(bodyJSON)

	reqLog := &logger.RequestLog{
		Timestamp: start,
		Method:    c.Request.Method,
		Path:      c.Request.URL.Path,
		Headers:   c.Request.Header,
		Body:      bodyJSON,
	}
	h.logger.LogRequest(sessionID, reqLog)

	// Build target URL
	targetURL := h.targetURL + c.Request.URL.Path
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	// Create upstream request
	upstreamReq, err := http.NewRequest(c.Request.Method, targetURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		log.Printf("Failed to create upstream request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upstream request"})
		return
	}

	// Copy headers (skip hop-by-hop headers)
	for k, vv := range c.Request.Header {
		for _, v := range vv {
			upstreamReq.Header.Add(k, v)
		}
	}

	// Send upstream request
	resp, err := h.client.Do(upstreamReq)
	if err != nil {
		log.Printf("Upstream request failed: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream request failed", "detail": err.Error()})
		return
	}
	defer resp.Body.Close()

	// Determine if response is a stream based on Content-Type
	isStreamResp := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")

	if isStreamResp {
		h.handleStream(c, resp, start, sessionID)
	} else {
		h.handleResponse(c, resp, start, sessionID)
	}
}

func (h *Handler) handleResponse(c *gin.Context, resp *http.Response, reqTime time.Time, sessionID string) {
	body, _ := io.ReadAll(resp.Body)

	var bodyJSON interface{}
	json.Unmarshal(body, &bodyJSON)

	respLog := &logger.ResponseLog{
		Timestamp:  reqTime,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       bodyJSON,
		Duration:   time.Since(reqTime).String(),
	}
	h.logger.LogResponse(sessionID, respLog)

	// Copy response headers to client
	for k, vv := range resp.Header {
		for _, v := range vv {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Write(body)
}

func (h *Handler) handleStream(c *gin.Context, resp *http.Response, reqTime time.Time, sessionID string) {
	// Copy response headers to client
	for k, vv := range resp.Header {
		for _, v := range vv {
			c.Writer.Header().Set(k, v)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)

	reader := bufio.NewReader(resp.Body)

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			// Write to client immediately
			fmt.Fprint(c.Writer, line)
			c.Writer.Flush()

			// Parse data lines and log each chunk immediately
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "data: ") {
				data := strings.TrimPrefix(trimmed, "data: ")
				var chunk interface{}
				if json.Unmarshal([]byte(data), &chunk) == nil {
					h.logger.LogStreamEvent(sessionID, &logger.EventLog{
						Timestamp: time.Now(),
						Data:      chunk,
					})
				}
			}
		}
		if err != nil {
			break
		}
	}

	// Write a summary line at stream end
	respLog := &logger.ResponseLog{
		Timestamp:  reqTime,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Duration:   time.Since(reqTime).String(),
	}
	h.logger.LogResponse(sessionID, respLog)
}
