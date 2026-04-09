package config

import (
	"fmt"
	"sort"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Managers map[string]ManagerConfig
	Groups   map[string]GroupConfig
	Prompts  map[string]PromptConfig
}

type PromptConfig struct {
	Type     string `toml:"type"`     // "bool" or "string"
	Question string `toml:"question"` // displayed to user
}

type ManagerConfig struct {
	Check   string
	Install string
	Remove  string
	Enable  string
	Disable string
}

type GroupConfig struct {
	Packages []NamedEntry
	Services []NamedEntry
}

type NamedEntry struct {
	Name string
}

type PackageEntry struct {
	Name    string
	Manager string
}

type ServiceEntry struct {
	Name    string
	Manager string
}

func Load(path string) (*Config, error) {
	var raw map[string]any
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	cfg := &Config{
		Managers: make(map[string]ManagerConfig),
		Groups:   make(map[string]GroupConfig),
		Prompts:  make(map[string]PromptConfig),
	}

	// Parse managers
	if rawManagers, ok := raw["managers"]; ok {
		mm, ok := rawManagers.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("managers must be a table")
		}
		for name, val := range mm {
			m, ok := val.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("managers.%s must be a table", name)
			}
			mgr := ManagerConfig{}
			for field, ptr := range map[string]*string{
				"check":   &mgr.Check,
				"install": &mgr.Install,
				"remove":  &mgr.Remove,
				"enable":  &mgr.Enable,
				"disable": &mgr.Disable,
			} {
				if v, ok := m[field]; ok {
					s, ok := v.(string)
					if !ok {
						return nil, fmt.Errorf("managers.%s: %s must be a string", name, field)
					}
					*ptr = s
				}
			}
			cfg.Managers[name] = mgr
		}
	}

	// Parse prompts
	if rawPrompts, ok := raw["prompts"]; ok {
		pm, ok := rawPrompts.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("prompts must be a table")
		}
		for name, val := range pm {
			p, ok := val.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("prompts.%s must be a table", name)
			}
			prompt := PromptConfig{}
			if v, ok := p["type"].(string); ok {
				prompt.Type = v
			}
			if v, ok := p["question"].(string); ok {
				prompt.Question = v
			}
			cfg.Prompts[name] = prompt
		}
	}

	// Parse groups (everything except "managers" and "prompts")
	for key, val := range raw {
		if key == "managers" || key == "prompts" {
			continue
		}
		group, err := parseGroup(val)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		cfg.Groups[key] = *group
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

func parseGroup(val any) (*GroupConfig, error) {
	m, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected a table")
	}

	g := &GroupConfig{}

	// Parse packages
	if rawPkgs, ok := m["packages"]; ok {
		arr, ok := rawPkgs.([]any)
		if !ok {
			return nil, fmt.Errorf("packages must be an array")
		}
		for _, item := range arr {
			switch v := item.(type) {
			case string:
				g.Packages = append(g.Packages, NamedEntry{Name: v})
			case map[string]any:
				if name, ok := v["name"].(string); ok {
					g.Packages = append(g.Packages, NamedEntry{Name: name})
				}
			}
		}
	}

	// Parse services
	if rawSvcs, ok := m["services"]; ok {
		arr, ok := rawSvcs.([]any)
		if !ok {
			return nil, fmt.Errorf("services must be an array")
		}
		for _, item := range arr {
			switch v := item.(type) {
			case string:
				g.Services = append(g.Services, NamedEntry{Name: v})
			case map[string]any:
				if name, ok := v["name"].(string); ok {
					g.Services = append(g.Services, NamedEntry{Name: name})
				}
			}
		}
	}

	return g, nil
}

func (c *Config) validate() error {
	if len(c.Managers) == 0 {
		return fmt.Errorf("no managers defined")
	}
	for name, mgr := range c.Managers {
		if mgr.Check == "" {
			return fmt.Errorf("managers.%s: check command is required", name)
		}
	}
	for name := range c.Groups {
		if _, ok := c.Managers[name]; !ok {
			return fmt.Errorf("unknown manager %q", name)
		}
	}
	for name, p := range c.Prompts {
		switch p.Type {
		case "bool", "string":
		default:
			return fmt.Errorf("prompt %q: unknown type %q", name, p.Type)
		}
		if p.Question == "" {
			return fmt.Errorf("prompt %q: question is required", name)
		}
	}
	return nil
}

// sortedGroupNames returns group names in sorted order for deterministic iteration.
func (c *Config) sortedGroupNames() []string {
	names := make([]string, 0, len(c.Groups))
	for name := range c.Groups {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c *Config) Packages() []PackageEntry {
	var result []PackageEntry

	for _, mgrName := range c.sortedGroupNames() {
		group := c.Groups[mgrName]
		if len(group.Packages) == 0 {
			continue
		}
		for _, entry := range group.Packages {
			result = append(result, PackageEntry{
				Name:    entry.Name,
				Manager: mgrName,
			})
		}
	}
	return result
}

func (c *Config) Services() []ServiceEntry {
	var result []ServiceEntry

	for _, mgrName := range c.sortedGroupNames() {
		group := c.Groups[mgrName]
		if len(group.Services) == 0 {
			continue
		}
		for _, entry := range group.Services {
			result = append(result, ServiceEntry{
				Name:    entry.Name,
				Manager: mgrName,
			})
		}
	}
	return result
}
