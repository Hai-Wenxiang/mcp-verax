// Command badge-server runs the verax badge service.
//
// Usage: go run ./apps/badge/cmd -pulse docs/registry/pulse-latest.json -addr :8080
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	veraxbadge "github.com/Hai-Wenxiang/mcp-verax/apps/badge"
)

func main() {
	pulse := flag.String("pulse", "docs/registry/pulse-latest.json", "pulse JSON to serve badges from")
	addr := flag.String("addr", ":8080", "listen address")
	refresh := flag.Duration("refresh", time.Hour, "pulse reload interval")
	flag.Parse()

	store := &veraxbadge.Store{}
	if err := store.Load(*pulse); err != nil {
		log.Fatalf("badge-server: load pulse: %v", err)
	}
	go func() {
		for range time.Tick(*refresh) {
			if err := store.Load(*pulse); err != nil {
				log.Printf("badge-server: reload: %v", err)
			}
		}
	}()
	log.Printf("badge-server: listening on %s (pulse %s)", *addr, *pulse)
	log.Fatal(http.ListenAndServe(*addr, veraxbadge.Handler(store)))
}
