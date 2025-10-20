package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/lehigh-university-libraries/cataloger/internal/evalcmd"
	"github.com/lehigh-university-libraries/cataloger/internal/handlers"
	"github.com/lehigh-university-libraries/cataloger/internal/utils"
)

func main() {
	// Load .env file if present
	_ = godotenv.Load()

	// If no args or first arg is "serve", run web server
	if len(os.Args) == 1 || os.Args[1] == "serve" {
		runServer()
		return
	}

	// If first arg is "eval", handle evaluation commands
	if os.Args[1] == "eval" {
		runEvalCommands()
		return
	}

	// Handle other top-level commands
	switch os.Args[1] {
	case "help", "-h", "--help":
		printUsage()
	case "version":
		fmt.Println("cataloger version 0.1.0")
	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`cataloger - MARC cataloging tool

Usage:
  cataloger [serve]       Start web server for cataloging interface (default)
  cataloger eval <cmd>    Run evaluation commands
  cataloger version       Print version
  cataloger help          Show this help

Commands:
  serve                   Start web server on port 8888 (default)
  eval                    Evaluation tools (see 'cataloger eval help')

Examples:
  # Start web server (default behavior)
  cataloger
  cataloger serve

  # Run evaluation with Institutional Books dataset
  cataloger eval eval-ib --sample 10

  # Fetch evaluation data from OAI-PMH
  cataloger eval fetch --url https://folio.example.edu/oai

For more information on eval commands:
  cataloger eval help
`)
}

func runServer() {
	handler := handlers.New()

	// Set up routes
	http.HandleFunc("/api/sessions", handler.HandleSessions)
	http.HandleFunc("/api/sessions/", handler.HandleSessionDetail)
	http.HandleFunc("/api/upload", handler.HandleUpload)
	http.HandleFunc("/", handler.HandleStatic)
	http.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("OK"))
		if err != nil {
			slog.Error("Unable to write healthcheck", "err", err)
			os.Exit(1)
		}
	})

	addr := ":8888"
	slog.Info("Cataloger interface available", "addr", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		utils.ExitOnError("Server failed to start", err)
	}
}

func runEvalCommands() {
	// Import and run eval main logic
	// We need to adjust os.Args to remove "eval" and pass to evalcmd.Run
	os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	evalcmd.Run()
}
