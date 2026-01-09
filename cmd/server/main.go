package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/xilu0/cc-debug/internal/logger"
	"github.com/xilu0/cc-debug/internal/proxy"
)

func main() {
	port := flag.Int("port", 8080, "Server port")
	output := flag.String("output", "console", "Output mode: console or json")
	outDir := flag.String("dir", "logs", "Output directory for JSON files")
	flag.Parse()

	mode := logger.ModeConsole
	if *output == "json" {
		mode = logger.ModeJSON
	}

	l := logger.New(mode, *outDir)
	h := proxy.NewHandler(l)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.Any("/*path", h.Proxy)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting Claude API debug proxy on %s (output: %s)", addr, *output)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
