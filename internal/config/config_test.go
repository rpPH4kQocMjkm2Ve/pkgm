package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	content := `[managers.pacman]
check = "pacman -Q {{.Name}}"
install = "pacman -S --needed {{.Name}}"
remove = "pacman -Rns {{.Name}}"

[managers.systemd]
check = "systemctl is-enabled {{.Name}}"
enable = "systemctl enable {{.Name}}"
disable = "systemctl disable {{.Name}}"

[managers.systemd-user]
check = "systemctl --user is-enabled {{.Name}}"
enable = "systemctl --user enable {{.Name}}"
disable = "systemctl --user disable {{.Name}}"

[pacman]
packages = ["git", "zsh"]

[systemd]
services = ["firewalld", "sshd"]

[systemd-user]
services = ["hypridle", "waybar"]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "pkgm.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.Managers) != 3 {
		t.Errorf("expected 3 managers, got %d", len(cfg.Managers))
	}

	pkgs := cfg.Packages()
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}

	names := make(map[string]string)
	for _, p := range pkgs {
		names[p.Name] = p.Manager
	}
	if names["git"] != "pacman" {
		t.Errorf("expected git->pacman, got %s", names["git"])
	}

	svcs := cfg.Services()
	if len(svcs) != 4 {
		t.Fatalf("expected 4 services, got %d", len(svcs))
	}

	svcNames := make(map[string]string)
	for _, s := range svcs {
		svcNames[s.Name] = s.Manager
	}
	if svcNames["firewalld"] != "systemd" {
		t.Errorf("expected firewalld->systemd, got %s", svcNames["firewalld"])
	}
	if svcNames["hypridle"] != "systemd-user" {
		t.Errorf("expected hypridle->systemd-user, got %s", svcNames["hypridle"])
	}
}

func TestLoadCompact(t *testing.T) {
	content := `[managers.pacman]
check = "pacman -Q {{.Name}}"

[pacman]
packages = ["git"]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "pkgm.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	pkgs := cfg.Packages()
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	if pkgs[0].Name != "git" {
		t.Errorf("expected git, got %s", pkgs[0].Name)
	}
	if pkgs[0].Manager != "pacman" {
		t.Errorf("expected pacman, got %s", pkgs[0].Manager)
	}
}

func TestLoadUnknownManager(t *testing.T) {
	content := `[managers.pacman]
check = "pacman -Q {{.Name}}"

[unknown]
packages = ["git"]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "pkgm.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unknown manager group")
	}
}

func TestLoadNoManagers(t *testing.T) {
	content := `[pacman]
packages = ["git"]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "pkgm.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for no managers")
	}
}

func TestLoadMixedPackageFormats(t *testing.T) {
	content := `[managers.pacman]
check = "pacman -Q {{.Name}}"

[pacman]
packages = ["git", { name = "nvidia-utils" }]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "pkgm.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	pkgs := cfg.Packages()
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}

	names := make(map[string]bool)
	for _, p := range pkgs {
		names[p.Name] = true
	}
	if !names["git"] {
		t.Errorf("expected git in packages")
	}
	if !names["nvidia-utils"] {
		t.Errorf("expected nvidia-utils in packages")
	}
}

func TestEmptyServiceGroup(t *testing.T) {
	content := `[managers.pacman]
check = "pacman -Q {{.Name}}"

[pacman]
packages = ["git"]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "pkgm.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	svcs := cfg.Services()
	if len(svcs) != 0 {
		t.Fatalf("expected 0 services, got %d", len(svcs))
	}
}

func TestLoadPrompts(t *testing.T) {
	content := `[managers.pacman]
check = "pacman -Q {{.Name}}"

[prompts]
laptop = { type = "bool", question = "Is this a laptop?" }
git_email = { type = "string", question = "Git email" }

[pacman]
packages = ["git"]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "pkgm.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.Prompts) != 2 {
		t.Fatalf("expected 2 prompts, got %d", len(cfg.Prompts))
	}
	if cfg.Prompts["laptop"].Type != "bool" {
		t.Errorf("expected laptop type bool, got %q", cfg.Prompts["laptop"].Type)
	}
	if cfg.Prompts["laptop"].Question != "Is this a laptop?" {
		t.Errorf("expected laptop question, got %q", cfg.Prompts["laptop"].Question)
	}
	if cfg.Prompts["git_email"].Type != "string" {
		t.Errorf("expected git_email type string, got %q", cfg.Prompts["git_email"].Type)
	}
}

func TestLoadPromptsValidation(t *testing.T) {
	content := `[managers.pacman]
check = "pacman -Q {{.Name}}"

[prompts]
bad = { type = "unknown", question = "Bad type" }

[pacman]
packages = ["git"]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "pkgm.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unknown prompt type")
	}
}

func TestLoadPromptsQuestionRequired(t *testing.T) {
	content := `[managers.pacman]
check = "pacman -Q {{.Name}}"

[prompts]
bad = { type = "bool" }

[pacman]
packages = ["git"]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "pkgm.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing prompt question")
	}
}
