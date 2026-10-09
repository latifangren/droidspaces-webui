package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/latifangren/droidspaces-webui/internal/api"
)

var listenAndServe = func(srv *http.Server) error {
	return srv.ListenAndServe()
}

func resolvePort(portFlag int, customFiles ...string) int {
	port := portFlag
	if envPort := os.Getenv("DSWEB_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil && p > 0 {
			return p
		}
	} else if portFlag == 84 {
		candidates := []string{
			"/data/local/Droidspaces/webui_port",
			"/data/adb/modules/droidspaces/port",
		}
		if len(customFiles) > 0 {
			candidates = customFiles
		}
		for _, pf := range candidates {
			if data, err := os.ReadFile(pf); err == nil {
				if p, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && p > 0 {
					return p
				}
			}
		}
	}
	return port
}

func buildServer(port int) *http.Server {
	server := api.NewServer(port)
	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: server.Handler(),
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("dsweb", flag.ContinueOnError)
	portFlag := fs.Int("port", 84, "Port to listen on (default 84)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	port := resolvePort(*portFlag)
	srv := buildServer(port)
	log.Printf("[+] Droidspaces WebUI running on http://0.0.0.0:%d\n", port)
	return listenAndServe(srv)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
