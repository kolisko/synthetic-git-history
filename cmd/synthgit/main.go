package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kolisko/synthetic-git-history/internal/config"
	"github.com/kolisko/synthetic-git-history/internal/gitops"
	"github.com/kolisko/synthetic-git-history/internal/schedule"
)

const usage = `synthgit generates synthetic Git commit histories for test repositories.

Usage:
  synthgit plan --config config.example.json
  synthgit generate --config config.example.json [--dry-run] [--push]
  synthgit init-config [--output synthgit.config.json]
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}

	switch args[0] {
	case "plan":
		return runPlan(args[1:])
	case "generate":
		return runGenerate(args[1:])
	case "init-config":
		return runInitConfig(args[1:])
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usage)
	}
}

func runPlan(args []string) error {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "", "Path to JSON config file.")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *configPath == "" {
		return fmt.Errorf("missing --config")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	specs := schedule.Build(cfg)
	printPlan(specs)
	return nil
}

func runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "", "Path to JSON config file.")
	dryRun := fs.Bool("dry-run", false, "Print the schedule without changing files.")
	pushRequested := fs.Bool("push", false, "Push after generation when config also allows it.")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *configPath == "" {
		return fmt.Errorf("missing --config")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	specs := schedule.Build(cfg)
	if *dryRun {
		printPlan(specs)
		fmt.Printf("\nDry run: no files changed. Planned commits: %d\n", len(specs))
		return nil
	}

	if err := gitops.EnsureRepository(cfg); err != nil {
		return err
	}
	for _, spec := range specs {
		if err := gitops.ApplyCommit(cfg, spec); err != nil {
			return err
		}
	}

	fmt.Printf("Created %d commits in %s\n", len(specs), cfg.Repository.Path)

	if *pushRequested {
		if !cfg.Repository.Push {
			return fmt.Errorf("refusing to push because repository.push is false in config")
		}
		if err := gitops.Push(cfg); err != nil {
			return err
		}
		fmt.Printf("Pushed branch %s to origin\n", cfg.Repository.Branch)
	} else if cfg.Repository.Push {
		fmt.Println("Config allows push, but CLI --push was not provided. Nothing was pushed.")
	}

	return nil
}

func runInitConfig(args []string) error {
	fs := flag.NewFlagSet("init-config", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	output := fs.String("output", "synthgit.config.json", "Output config path.")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if _, err := os.Stat(*output); err == nil {
		return fmt.Errorf("refusing to overwrite existing file: %s", *output)
	}
	return os.WriteFile(*output, []byte(config.ExampleJSON), 0644)
}

func printPlan(specs []schedule.CommitSpec) {
	if len(specs) == 0 {
		fmt.Println("No commits planned.")
		return
	}
	for _, spec := range specs {
		fmt.Printf("%s | %s\n", spec.GitDate(), spec.Message)
	}
	fmt.Printf("\nPlanned commits: %d\n", len(specs))
}
