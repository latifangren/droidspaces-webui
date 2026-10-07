package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/latifangren/droidspaces-webui/internal/api"
)

func main() {
	portFlag := flag.Int("port", 84, "Port to listen on (default 84)")
	flag.Parse()

	port := *portFlag
	if envPort := os.Getenv("DSWEB_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil && p > 0 {
			port = p
		}
	}

	server := api.NewServer(port)
	addr := fmt.Sprintf(":%d", port)

	log.Printf("[+] Droidspaces WebUI running on http://0.0.0.0:%d\n", port)
	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
