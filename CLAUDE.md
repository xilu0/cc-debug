# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

cc-debug is a Claude API debug proxy written in Go. It intercepts Claude client requests, logs them to console or JSON files, and supports streaming responses.

## Build and Run Commands

```bash
# Build
go build -o cc-debug ./cmd/server

# Run
./cc-debug -port 8080 -output console    # Log to console
./cc-debug -port 8080 -output json       # Save to JSON files

# Test
go test ./...

# Run single test
go test -run TestName ./path/to/package
```

## Architecture

```
cmd/server/         - Main entry point
internal/
  proxy/            - HTTP proxy handler for Claude API requests
  logger/           - Request/response logging (console and JSON output)
  stream/           - SSE stream handling for streaming responses
```

The proxy intercepts requests to Claude's Messages API, logs the full request/response cycle, then forwards to the actual Claude API and streams responses back to the client.
