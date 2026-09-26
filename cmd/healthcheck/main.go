// Command healthcheck probes a service's /readyz endpoint for container health
// checks; the runtime images have no shell or HTTP client of their own.
//
//	healthcheck ADMIN_PORT 8086
//
// It reads the port from the named environment variable, falling back to the
// default, and exits 0 only if GET http://127.0.0.1:<port>/readyz returns 2xx.
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: healthcheck PORT_ENV_VAR DEFAULT_PORT")
		os.Exit(2)
	}
	port := os.Getenv(os.Args[1])
	if port == "" {
		port = os.Args[2]
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/readyz")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		fmt.Fprintln(os.Stderr, "readyz returned", resp.Status)
		os.Exit(1)
	}
}
