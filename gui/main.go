package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

//go:embed frontend/dist/*
var staticFiles embed.FS

const (
	appName    = "Nimbus Backup"
	appVersion = "0.4.0"
	serverPort = "8765"
)

func main() {
	log.Printf("=== %s v%s - RDEM Systems ===\n", appName, appVersion)
	log.Println("Démarrage du serveur web...")

	// Create app instance
	app := NewApp()

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Serve static files
	mux.Handle("/", http.FileServer(http.FS(staticFiles)))

	// API endpoints
	mux.HandleFunc("/api/config", app.handleConfig)
	mux.HandleFunc("/api/test-connection", app.handleTestConnection)
	mux.HandleFunc("/api/backup/start", app.handleBackupStart)
	mux.HandleFunc("/api/snapshots/list", app.handleSnapshotList)
	mux.HandleFunc("/api/snapshots/restore", app.handleSnapshotRestore)
	mux.HandleFunc("/api/info", app.handleInfo)

	// Create server
	srv := &http.Server{
		Addr:    ":" + serverPort,
		Handler: mux,
	}

	// Open browser
	url := fmt.Sprintf("http://localhost:%s/frontend/dist/index.html", serverPort)
	go func() {
		time.Sleep(500 * time.Millisecond)
		log.Printf("Ouverture du navigateur sur %s\n", url)
		openBrowser(url)
	}()

	// Handle graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("\nArrêt du serveur...")
		srv.Close()
		os.Exit(0)
	}()

	// Start server
	log.Printf("Serveur démarré sur http://localhost:%s\n", serverPort)
	log.Println("Appuyez sur Ctrl+C pour arrêter")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Erreur serveur: %v\n", err)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Impossible d'ouvrir le navigateur automatiquement.\n")
		log.Printf("Ouvrez manuellement : %s\n", url)
	}
}
