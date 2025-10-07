package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const (
	version = "0.1.0"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	err := godotenv.Load()
	if err != nil {
		slog.Warn("Error loading .env file", "err", err)
	}
	setupLogger()

	command := os.Args[1]

	switch command {
	case "fetch":
		fetchCmd()
	case "enrich":
		enrichCmd()
	case "run":
		runCmd()
	case "report":
		reportCmd()
	case "version":
		fmt.Printf("cataloger-eval version %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func setupLogger() {
	logLevel := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	if logLevel == "" {
		logLevel = "INFO"
	}

	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		slog.Info("Unknown log level", "logLevel", logLevel)
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	logger := slog.New(handler)

	slog.SetDefault(logger)
}

func printUsage() {
	fmt.Print(`cataloger-eval - MARC cataloging evaluation tool

Usage:
  eval fetch [options]    Fetch records from OAI-PMH endpoint to build evaluation dataset
  eval enrich [options]   Enrich dataset with images and MARCXML for each ISBN
  eval run [options]      Run evaluation on dataset
  eval report [options]   Generate detailed comparison report
  eval version            Print version
  eval help               Show this help

Examples:
  # Fetch 100 books with ISBN from FOLIO OAI-PMH endpoint
  eval fetch --url https://folio.example.edu/oai --limit 100

  # Enrich dataset with images (creates ISBN directories with marc.xml + images)
  eval enrich --dataset ./eval_data --output ./enriched_data

  # Run evaluation with Ollama
  eval run --dataset ./eval_data --provider ollama --model mistral-small3.2:24b

  # Generate detailed report
  eval report --results ./eval_results
`)
}

type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprint(*s)
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func fetchCmd() {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	url := fs.String("url", os.Getenv("OAI_PMH_URL"), "OAI-PMH base URL (required)")
	prefix := fs.String("prefix", "marc21", "OAI-PMH metadataPrefix")
	limit := fs.Int("limit", 100, "Maximum number of records to fetch")
	output := fs.String("output", "./eval_data", "Output directory for dataset")
	sleep := fs.Int("sleep", 0, "Seconds to sleep between resumption token requests (0 = no sleep)")

	var excludeTags stringSlice
	fs.Var(&excludeTags, "exclude", "MARC tag to exclude (can be specified multiple times)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *url == "" {
		fmt.Println("Error: --url is required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	if err := executeFetch(*url, *prefix, *output, *limit, excludeTags, *sleep); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func enrichCmd() {
	fs := flag.NewFlagSet("enrich", flag.ExitOnError)
	dataset := fs.String("dataset", "./eval_data", "Dataset directory containing dataset.json")
	output := fs.String("output", "./enriched_data", "Output directory for enriched data")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if err := executeEnrich(*dataset, *output); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func runCmd() {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	dataset := fs.String("dataset", "./eval_data", "Dataset directory")
	provider := fs.String("provider", "ollama", "LLM provider (ollama or openai)")
	model := fs.String("model", "", "Model name (uses defaults if not specified)")
	output := fs.String("output", "./eval_results", "Output directory for results")
	concurrency := fs.Int("concurrency", 1, "Number of concurrent evaluations")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if err := executeRun(*dataset, *provider, *model, *output, *concurrency); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func reportCmd() {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	results := fs.String("results", "./eval_results", "Results directory")
	format := fs.String("format", "text", "Output format (text, json, csv)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if err := executeReport(*results, *format); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
