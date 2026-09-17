package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/SimoneLocatelli/claudewatch/internal/api"
	"github.com/SimoneLocatelli/claudewatch/internal/auth"
	"github.com/SimoneLocatelli/claudewatch/internal/config"
	"github.com/SimoneLocatelli/claudewatch/internal/statusline"
	"github.com/SimoneLocatelli/claudewatch/internal/theme"
)

func main() {
	if len(os.Args) > 1 {
		var err error
		switch os.Args[1] {
		case "install":
			err = install()
		case "uninstall":
			err = uninstall()
		case "update":
			err = update()
		case "version":
			printVersion()
			return
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if os.Args[1] == "install" || os.Args[1] == "uninstall" || os.Args[1] == "update" || os.Args[1] == "version" {
			return
		}
	}

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("no input on stdin — pipe Claude Code JSON or run: claudewatch install")
	}

	status, err := statusline.Parse(data)
	if err != nil {
		return err
	}

	// Populate environment info.
	if wd, wdErr := os.Getwd(); wdErr == nil {
		status.Cwd = filepath.Base(wd)
	}
	if branch, brErr := gitBranch(); brErr == nil {
		status.Branch = branch
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	t, err := theme.Load(cfg.Theme)
	if err != nil {
		t, _ = theme.LoadBuiltin("dracula")
	}

	// Read credentials for plan detection and API access.
	var plan string
	var usage *api.Usage
	creds, credErr := auth.Read()
	if credErr == nil {
		plan = auth.PlanName(creds.SubscriptionType)
		if creds.AccessToken != "" {
			usage, _ = api.FetchUsage(creds.AccessToken)
		}
	}

	output := statusline.Render(&status, &t, plan, usage, &cfg)
	// Leading reset clears stale ANSI state from previous renders.
	// Non-breaking spaces prevent the terminal from collapsing whitespace.
	output = "\x1b[0m" + strings.ReplaceAll(output, " ", "\u00A0")
	// Extra newline adds visual padding below the status line.
	fmt.Println(output + "\n")
	return nil
}

func install() error {
	binaryPath, err := exec.LookPath("claudewatch")
	if err != nil {
		binaryPath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("could not find claudewatch binary: %w", err)
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")

	settings := make(map[string]interface{})
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		// Refuse to continue on a parse error: silently proceeding would
		// overwrite the whole file with only the statusLine key.
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parsing %s: %w (fix the file, then retry)", settingsPath, err)
		}
		if err := backupSettings(settingsPath, data); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", settingsPath, err)
	}

	settings["statusLine"] = map[string]interface{}{
		"type":    "command",
		// Forward slashes: Claude Code runs the command through Git Bash on
		// Windows, which eats unescaped backslashes and fails silently.
		"command": filepath.ToSlash(binaryPath),
	}

	_ = os.MkdirAll(filepath.Dir(settingsPath), 0o755)

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(settingsPath, out, 0o644); err != nil {
		return err
	}

	// Create default config file if it doesn't exist.
	cfgPath, err := config.Path()
	if err == nil {
		if _, statErr := os.Stat(cfgPath); os.IsNotExist(statErr) {
			if saveErr := config.Save(config.DefaultConfig()); saveErr == nil {
				fmt.Printf("  config:   %s\n", cfgPath)
			}
		}
	}

	fmt.Printf("Installed claudewatch as Claude Code status line\n")
	fmt.Printf("  binary:   %s\n", binaryPath)
	fmt.Printf("  settings: %s\n", settingsPath)
	fmt.Printf("\nRestart Claude Code to see your status line.\n")
	return nil
}

func uninstall() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Remove statusLine from Claude Code settings.
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		settings := make(map[string]interface{})
		if err := json.Unmarshal(data, &settings); err == nil {
			delete(settings, "statusLine")
			out, err := json.MarshalIndent(settings, "", "  ")
			if err == nil {
				_ = backupSettings(settingsPath, data)
				_ = os.WriteFile(settingsPath, out, 0o644)
			}
		}
	}

	// Remove usage cache.
	_ = os.Remove(filepath.Join(os.TempDir(), "claudewatch-usage.json"))

	// Remove the binary itself.
	binPath, _ := os.Executable()
	if binPath != "" {
		_ = os.Remove(binPath)
	}

	fmt.Printf("Uninstalled claudewatch\n")
	fmt.Printf("  removed statusLine from %s\n", settingsPath)
	fmt.Printf("  kept config at ~/.config/claudewatch/config.toml\n")
	if binPath != "" {
		fmt.Printf("  removed binary %s\n", binPath)
	}
	fmt.Printf("\nRestart Claude Code to apply.\n")
	return nil
}

// backupSettings copies the current settings.json to settings.json.bak
// before it is rewritten, so a bad write can be reverted by hand.
func backupSettings(path string, data []byte) error {
	if err := os.WriteFile(path+".bak", data, 0o600); err != nil {
		return fmt.Errorf("backing up %s: %w", path, err)
	}
	return nil
}

func update() error {
	// Show current version.
	currentVersion := version()
	fmt.Printf("Current version: %s\n", currentVersion)

	// Fetch and install latest.
	fmt.Printf("Fetching latest version...\n")
	cmd := exec.Command("go", "install", "github.com/SimoneLocatelli/claudewatch@v0.4.1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go install failed: %w", err)
	}

	// Show new version.
	out, err := exec.Command("claudewatch", "version").Output()
	if err == nil {
		fmt.Printf("Updated to: %s", string(out))
	}

	// Re-register to update binary path in settings.
	if err := install(); err != nil {
		return fmt.Errorf("re-registering: %w", err)
	}

	return nil
}

func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "dev"
	}
	return info.Main.Version
}

func printVersion() {
	fmt.Println(version())
}

func gitBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
