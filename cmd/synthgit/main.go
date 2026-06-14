package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kolisko/synthetic-git-history/internal/config"
	"github.com/kolisko/synthetic-git-history/internal/gitops"
	"github.com/kolisko/synthetic-git-history/internal/schedule"
)

const usage = `synthgit generates synthetic Git commit histories for test repositories.

Usage:
  synthgit plan [--config <path>]
  synthgit generate [--config <path>] [--dry-run] [--push]
  synthgit init-config [--output <path>]
`

const (
	appName        = "synthgit"
	configFileName = "config.json"
)

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
	defaultPath, err := defaultConfigPath()
	if err != nil {
		return err
	}

	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", defaultPath, "Path to JSON config file.")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return err
	}
	specs := schedule.Build(cfg)
	printPlan(specs)
	return nil
}

func runGenerate(args []string) error {
	defaultPath, err := defaultConfigPath()
	if err != nil {
		return err
	}

	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", defaultPath, "Path to JSON config file.")
	dryRun := fs.Bool("dry-run", false, "Print the schedule without changing files.")
	pushRequested := fs.Bool("push", false, "Push after generation when config also allows it.")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := loadConfig(*configPath)
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
	defaultPath, err := defaultConfigPath()
	if err != nil {
		return err
	}

	fs := flag.NewFlagSet("init-config", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	output := fs.String("output", defaultPath, "Output config path.")
	if err := fs.Parse(args); err != nil {
		return err
	}

	outputPath, err := resolveOutputPath(*output)
	if err != nil {
		return err
	}
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("refusing to overwrite existing file: %s", outputPath)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, []byte(config.ExampleJSON), 0644); err != nil {
		return err
	}

	fmt.Printf("Created config: %s\n\n", outputPath)
	fmt.Print(config.ExampleJSON)
	fmt.Printf("\nEdit this file to change the target repository, date range, commit volume, identity, and push settings.\n")
	printNextSteps(outputPath)
	return nil
}

func printNextSteps(outputPath string) {
	defaultPath, err := defaultConfigPath()
	if err == nil && outputPath == defaultPath {
		fmt.Printf("Then run:\n  synthgit plan\n  synthgit generate\n")
		return
	}
	fmt.Printf("Then run:\n  synthgit plan --config %s\n  synthgit generate --config %s\n", outputPath, outputPath)
}

func loadConfig(path string) (config.Config, error) {
	configPath, err := resolveOutputPath(path)
	if err != nil {
		return config.Config{}, err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config.Config{}, fmt.Errorf("config file not found: %s\nRun `synthgit init-config` to create one, or pass --config with another path", configPath)
		}
		return config.Config{}, err
	}
	return cfg, nil
}

func defaultConfigPath() (string, error) {
	var configDir string
	if runtime.GOOS == "windows" {
		var err error
		configDir, err = os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("resolve user config directory: %w", err)
		}
	} else {
		configDir = os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("resolve home directory: %w", err)
			}
			configDir = filepath.Join(home, ".config")
		}
	}
	return filepath.Join(configDir, appName, configFileName), nil
}

func resolveOutputPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("output path cannot be empty")
	}

	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}

	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return absolute, nil
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
