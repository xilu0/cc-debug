package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/xilu0/cc-debug/internal/logger"
	"github.com/xilu0/cc-debug/internal/proxy"
)

func main() {
	port := flag.Int("port", 8080, "Server port")
	output := flag.String("output", "console", "Output mode: console or json")
	outDir := flag.String("dir", "logs", "Output directory for JSON files")
	target := flag.String("target", "", "Target API base URL (default: ANTHROPIC_BASE_URL env, fallback: https://api.anthropic.com)")
	flag.Parse()

	// Determine target URL: flag > env > default
	targetURL := *target
	if targetURL == "" {
		targetURL = os.Getenv("ANTHROPIC_BASE_URL")
	}
	if targetURL == "" {
		targetURL = "https://api.anthropic.com"
	}

	mode := logger.ModeConsole
	if *output == "json" {
		mode = logger.ModeJSON
	}

	l := logger.New(mode, *outDir)
	defer l.Close()
	h := proxy.NewHandler(l, targetURL)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.Any("/*path", h.Proxy)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting Claude API debug proxy on %s -> %s (output: %s)", addr, targetURL, *output)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
