package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	version = "0.1.0"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "fetch":
		fetchCmd()
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

func printUsage() {
	fmt.Print(`cataloger-eval - MARC cataloging evaluation tool

Usage:
  eval fetch [options]    Fetch records from catalog to build evaluation dataset
  eval run [options]      Run evaluation on dataset
  eval report [options]   Generate detailed comparison report
  eval version            Print version
  eval help               Show this help

Examples:
  # Fetch 100 records from VuFind catalog
  eval fetch --catalog vufind --url https://catalog.example.edu --limit 100

  # Run evaluation with Ollama
  eval run --dataset ./eval_data --provider ollama --model mistral-small3.2:24b

  # Generate detailed report
  eval report --results ./eval_results
`)
}

func fetchCmd() {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	catalog := fs.String("catalog", "vufind", "Catalog type (vufind or folio)")
	url := fs.String("url", "", "Catalog URL (required)")
	limit := fs.Int("limit", 100, "Number of records to fetch")
	output := fs.String("output", "./eval_data", "Output directory for dataset")
	apiKey := fs.String("api-key", "", "API key for FOLIO (optional)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *url == "" {
		fmt.Println("Error: --url is required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	if err := executeFetch(*catalog, *url, *apiKey, *output, *limit); err != nil {
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
