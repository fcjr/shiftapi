package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"

	"github.com/joho/godotenv"

	"{{modulePath}}/internal/server"
)

func main() {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		slog.Warn("error loading .env file, skipping", "error", err)
	}

	if err := run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	port := getenv("PORT")
	if port == "" {
		port = "{{port}}"
	}

	slog.Info("starting server", "port", port)
	// docs at http://localhost:{{port}}/docs
	return server.ListenAndServe(ctx, ":"+port)
}
