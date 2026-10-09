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

func main() {
	portFlag := flag.Int("port", 84, "Port to listen on (default 84)")
	flag.Parse()

	port := *portFlag
	if envPort := os.Getenv("DSWEB_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil && p > 0 {
			port = p
		}
	} else if *portFlag == 84 {
		for _, pf := range []string{
			"/data/local/Droidspaces/webui_port",
			"/data/adb/modules/droidspaces/port",
		} {
			if data, err := os.ReadFile(pf); err == nil {
				if p, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && p > 0 {
					port = p
					break
				}
			}
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
