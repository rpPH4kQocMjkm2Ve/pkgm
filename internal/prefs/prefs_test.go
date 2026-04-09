package prefs

import (
	"path/filepath"
	"strings"
	"testing"

	"pkgm/internal/config"
)

func TestStateFile(t *testing.T) {
	tests := []struct {
		name      string
		configDir string
	}{
		{"absolute", "/home/user/dotfiles"},
		{"different", "/home/user/root_m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := stateFile(tt.configDir)
			if err != nil {
				t.Fatalf("stateFile: %v", err)
			}
			if !filepath.IsAbs(path) {
				t.Errorf("expected absolute path, got %s", path)
			}
			if !strings.HasSuffix(path, ".toml") {
				t.Errorf("expected .toml suffix, got %s", path)
			}
		})
	}
}

func TestLoadSaveState(t *testing.T) {
	dir := t.TempDir()

	orig := &State{
		Data: map[string]any{
			"laptop": true,
			"email":  "user@example.com",
		},
	}

	if err := orig.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadState(dir)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	if v, ok := loaded.Data["laptop"].(bool); !ok || !v {
		t.Errorf("expected laptop=true, got %v", loaded.Data["laptop"])
	}
	if v, ok := loaded.Data["email"].(string); !ok || v != "user@example.com" {
		t.Errorf("expected email=user@example.com, got %v", loaded.Data["email"])
	}
}

func TestLoadEmptyState(t *testing.T) {
	dir := t.TempDir()

	s, err := LoadState(dir)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if len(s.Data) != 0 {
		t.Errorf("expected 0 data entries, got %d", len(s.Data))
	}
}

func TestResolveBoolPrompt(t *testing.T) {
	cfg := &config.Config{
		Prompts: map[string]config.PromptConfig{
			"laptop": {Type: "bool", Question: "Is this a laptop?"},
		},
	}

	state := &State{Data: map[string]any{}}
	input := strings.NewReader("y\n")
	var output strings.Builder

	changed, err := Resolve(cfg, state, input, &output)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if v, ok := state.Data["laptop"].(bool); !ok || !v {
		t.Errorf("expected laptop=true, got %v", state.Data["laptop"])
	}
	if !strings.Contains(output.String(), "Is this a laptop?") {
		t.Errorf("expected prompt question, got: %s", output.String())
	}
}

func TestResolveStringPrompt(t *testing.T) {
	cfg := &config.Config{
		Prompts: map[string]config.PromptConfig{
			"email": {Type: "string", Question: "Email address"},
		},
	}

	state := &State{Data: map[string]any{}}
	input := strings.NewReader("user@example.com\n")
	var output strings.Builder

	changed, err := Resolve(cfg, state, input, &output)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if v, ok := state.Data["email"].(string); !ok || v != "user@example.com" {
		t.Errorf("expected email=user@example.com, got %v", state.Data["email"])
	}
}

func TestResolveCachedSkip(t *testing.T) {
	cfg := &config.Config{
		Prompts: map[string]config.PromptConfig{
			"laptop": {Type: "bool", Question: "Is this a laptop?"},
		},
	}

	state := &State{Data: map[string]any{"laptop": true}}
	input := strings.NewReader("")
	var output strings.Builder

	changed, err := Resolve(cfg, state, input, &output)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if changed {
		t.Error("expected changed=false (cached)")
	}
	if output.Len() > 0 {
		t.Errorf("expected no output, got: %s", output.String())
	}
}

func TestResolveNoPrompts(t *testing.T) {
	cfg := &config.Config{}
	state := &State{Data: map[string]any{}}

	changed, err := Resolve(cfg, state, nil, nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if changed {
		t.Error("expected changed=false")
	}
}

func TestCoerceValue(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want any
	}{
		{"int64 stays", int64(42), int64(42)},
		{"float whole to int64", float64(3.0), int64(3)},
		{"float fractional stays", float64(3.14), float64(3.14)},
		{"string true", "true", true},
		{"string false", "FALSE", false},
		{"string regular", "hello", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coerceValue(tt.in)
			if got != tt.want {
				t.Errorf("coerceValue(%v) = %v (%T), want %v (%T)", tt.in, got, got, tt.want, tt.want)
			}
		})
	}
}

func TestBuildData(t *testing.T) {
	state := &State{
		Data: map[string]any{
			"laptop": true,
			"email":  "user@example.com",
		},
	}

	data := BuildData(state)

	if data["laptop"] != true {
		t.Errorf("expected laptop=true in data")
	}
	if data["email"] != "user@example.com" {
		t.Errorf("expected email=user@example.com in data")
	}
	if _, ok := data["homeDir"]; !ok {
		t.Errorf("expected homeDir in data")
	}
	if _, ok := data["hostname"]; !ok {
		t.Errorf("expected hostname in data")
	}
}

func TestFormatPromptValue(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"string", "hello", "hello"},
		{"int", int64(42), "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPromptValue(tt.in)
			if got != tt.want {
				t.Errorf("FormatPromptValue(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestAskBoolInvalid(t *testing.T) {
	cfg := &config.Config{
		Prompts: map[string]config.PromptConfig{
			"laptop": {Type: "bool", Question: "Is this a laptop?"},
		},
	}

	// Test with invalid input followed by valid
	state := &State{Data: map[string]any{}}
	input := strings.NewReader("maybe\nyes\n")
	var output strings.Builder

	changed, err := Resolve(cfg, state, input, &output)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if v, ok := state.Data["laptop"].(bool); !ok || !v {
		t.Errorf("expected laptop=true, got %v", state.Data["laptop"])
	}
	if !strings.Contains(output.String(), "please answer y or n") {
		t.Errorf("expected retry message, got: %s", output.String())
	}
}
