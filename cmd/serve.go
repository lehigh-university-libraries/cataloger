package cmd

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/lehigh-university-libraries/cataloger/internal/handlers"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var port string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start web server for cataloging interface",
		Long: `Starts the Cataloger web interface on the specified port.

The web interface allows you to upload book title page images and
generate MARC records using vision-capable LLMs (Ollama or OpenAI).`,
		Example: `  # Start server on default port 8888
  cataloger serve

  # Start server on custom port
  cataloger serve --port 3000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			handler := handlers.New()

			// Set up routes
			mux := http.NewServeMux()
			mux.HandleFunc("/api/sessions", handler.HandleSessions)
			mux.HandleFunc("/api/sessions/", handler.HandleSessionDetail)
			mux.HandleFunc("/api/upload", handler.HandleUpload)
			mux.HandleFunc("/", handler.HandleStatic)
			mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
				if _, err := w.Write([]byte("OK")); err != nil {
					slog.Error("Unable to write healthcheck", "err", err)
				}
			})

			addr := ":" + port
			server := &http.Server{
				Addr:    addr,
				Handler: mux,
			}

			// Start server in goroutine
			serverErr := make(chan error, 1)
			go func() {
				slog.Info("Cataloger interface available", "addr", addr, "url", "http://localhost"+addr)
				if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					serverErr <- err
				}
			}()

			// Wait for context cancellation (Ctrl+C) or server error
			select {
			case <-cmd.Context().Done():
				slog.Info("Shutting down server...")
				// Give server 5 seconds to shut down gracefully
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := server.Shutdown(shutdownCtx); err != nil {
					slog.Error("Server shutdown failed", "err", err)
					return err
				}
				slog.Info("Server stopped")
				return nil
			case err := <-serverErr:
				return err
			}
		},
	}

	cmd.Flags().StringVarP(&port, "port", "p", "8888", "Port to listen on")

	return cmd
}
