// Command kimi-no-name-wa serves the name generator's UI and API and opens it
// in a browser.
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/db"
	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/history"
	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/httpapi"
	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/namecheck"
	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/settings"
	"github.com/Jarrod-Bob/kimi-no-name-wa/internal/web"
)

func main() {
	// 127.0.0.1, never :7799: binding all interfaces prompts the Windows
	// firewall on every rebuild and exposes the API to the LAN.
	addr := flag.String("addr", "127.0.0.1:7799", "address to listen on")
	dbPath := flag.String("db", "", "database file (default: %AppData%\\kimi-no-name-wa\\kimi.db)")
	open := flag.Bool("open", true, "open a browser on start")
	flag.Parse()

	path := *dbPath
	if path == "" {
		resolved, err := db.DefaultPath()
		if err != nil {
			log.Fatalf("locating database: %v", err)
		}
		path = resolved
	}

	database, err := db.Open(path)
	if err != nil {
		log.Fatalf("opening database: %v", err)
	}
	defer database.Close()

	frontend, err := web.Handler()
	if err != nil {
		log.Fatalf("loading frontend: %v", err)
	}

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listening on %s: %v", *addr, err)
	}

	url := "http://" + *addr
	log.Printf("kimi-no-name-wa is at %s (db: %s)", url, path)
	if *open {
		go func() {
			time.Sleep(200 * time.Millisecond)
			if err := openBrowser(url); err != nil {
				log.Printf("could not open a browser: %v", err)
			}
		}()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Ollama is never contacted at startup: the app comes up whether or not
	// it is installed, and /api/v1/health says what's missing.
	handler := httpapi.NewServer(httpapi.Deps{
		Settings: settings.NewStore(database),
		History:  history.NewStore(database),
		Checker:  namecheck.New(),
		Frontend: frontend,
	})
	if host, _, err := net.SplitHostPort(listener.Addr().String()); err == nil {
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
			handler = httpapi.LocalOnly(handler)
		}
	}

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	select {
	case err := <-served:
		log.Fatalf("serving: %v", err)
	case <-ctx.Done():
	}
	stop() // a second Ctrl+C now kills the process outright

	log.Printf("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutting down: %v", err)
	}
}

// openBrowser launches the default browser. For a chromeless window instead,
// run: msedge --app=http://127.0.0.1:7799
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		// The empty string is start's window-title argument; without it a
		// quoted URL would be swallowed as the title.
		return exec.Command("cmd", "/c", "start", "", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
