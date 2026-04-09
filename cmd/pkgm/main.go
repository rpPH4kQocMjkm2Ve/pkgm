package main

import (
	"fmt"
	"os"
	"path/filepath"

	"pkgm/internal/config"
	"pkgm/internal/engine"
	"pkgm/internal/prefs"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "pkgm: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		return nil
	}

	cmd := args[0]
	flags := args[1:]

	switch cmd {
	case "init":
		return cmdInit()
	case "apply":
		return cmdApply(flags)
	case "status":
		return cmdStatus(flags)
	case "reset":
		return cmdReset(flags)
	case "help", "--help", "-h":
		printUsage()
		return nil
	case "version", "--version", "-V":
		cmdVersion()
		return nil
	default:
		return fmt.Errorf("unknown command %q\nrun 'pkgm help' for usage", cmd)
	}
}

func findConfig() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		path := filepath.Join(dir, "pkgm.toml")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("pkgm.toml not found in current directory or parents")
		}
		dir = parent
	}
}

func cmdInit() error {
	configPath, err := findConfig()
	if err != nil {
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	state, err := prefs.LoadState(configDir)
	if err != nil {
		return err
	}

	changed, err := prefs.Resolve(cfg, state, os.Stdin, os.Stdout)
	if err != nil {
		return err
	}

	if changed {
		if err := state.Save(configDir); err != nil {
			return fmt.Errorf("save state: %w", err)
		}
	}

	fmt.Printf("\npkgm initialized\n")
	fmt.Printf("  state: %s\n", prefs.FormatStateFile(configDir))

	if len(state.Data) > 0 {
		fmt.Printf("\n  data:\n")
		for k, v := range state.Data {
			fmt.Printf("    %s = %s\n", k, prefs.FormatPromptValue(v))
		}
	}

	return nil
}

func cmdApply(flags []string) error {
	dryRun := false
	for _, f := range flags {
		switch f {
		case "-n", "--dry-run":
			dryRun = true
		case "-h", "--help":
			fmt.Println(`Usage: pkgm apply [-n|--dry-run]

Install desired packages, remove obsolete ones.

Options:
  -n, --dry-run   Show what would happen without making changes
  -h, --help      Show this help`)
			return nil
		default:
			return fmt.Errorf("unknown flag %q\nrun 'pkgm apply --help' for usage", f)
		}
	}

	configPath, err := findConfig()
	if err != nil {
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	state, err := prefs.LoadState(configDir)
	if err != nil {
		return err
	}

	if changed, err := prefs.Resolve(cfg, state, os.Stdin, os.Stdout); err != nil {
		return err
	} else if changed {
		if err := state.Save(configDir); err != nil {
			return fmt.Errorf("save state: %w", err)
		}
	}

	prefsData := prefs.BuildData(state)
	eng := engine.New(cfg, configDir, prefsData)
	return eng.Apply(dryRun)
}

func cmdStatus(flags []string) error {
	for _, f := range flags {
		switch f {
		case "-h", "--help":
			fmt.Println(`Usage: pkgm status

Show current system state vs desired configuration.`)
			return nil
		default:
			return fmt.Errorf("unknown flag %q\nrun 'pkgm status --help' for usage", f)
		}
	}

	configPath, err := findConfig()
	if err != nil {
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	state, err := prefs.LoadState(configDir)
	if err != nil {
		return err
	}

	prefsData := prefs.BuildData(state)
	eng := engine.New(cfg, configDir, prefsData)
	return eng.Status()
}

func cmdReset(flags []string) error {
	if len(flags) == 0 {
		return fmt.Errorf("reset requires prompt names or --all\nrun 'pkgm help reset' for usage")
	}

	all := false
	var names []string
	for _, f := range flags {
		switch f {
		case "--all":
			all = true
		default:
			names = append(names, f)
		}
	}

	if all && len(names) > 0 {
		return fmt.Errorf("cannot mix --all with prompt names")
	}

	configPath, err := findConfig()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	state, err := prefs.LoadState(configDir)
	if err != nil {
		return err
	}

	if all {
		state.Data = make(map[string]any)
	} else {
		for _, name := range names {
			if _, ok := state.Data[name]; !ok {
				return fmt.Errorf("prompt %q not found in state", name)
			}
			delete(state.Data, name)
		}
	}

	if err := state.Save(configDir); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	if all {
		fmt.Println("All prompts reset.")
	} else {
		for _, name := range names {
			fmt.Printf("Prompt %q reset.\n", name)
		}
	}

	return nil
}

func printUsage() {
	fmt.Println(`Usage: pkgm <command> [options]

Commands:
  init                    Run prompts, create state cache
  apply [-n|--dry-run]    Install desired, remove obsolete
  status                  Show current vs desired state
  reset NAME              Reset cached prompt value(s)
  reset --all             Reset all cached values
  help                    Show this help
  version                 Show version`)
}
