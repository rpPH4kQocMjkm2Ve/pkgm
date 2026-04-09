package engine

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"pkgm/internal/config"
	"pkgm/internal/manifest"
)

type Engine struct {
	cfg       *config.Config
	configDir string
	data      map[string]any
	tmplCache map[string]*template.Template
}

func New(cfg *config.Config, configDir string, data map[string]any) *Engine {
	return &Engine{cfg: cfg, configDir: configDir, data: data}
}

// shellQuote returns a shell-safe single-quoted string.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// rawData returns template data with raw (unquoted) values — used for name rendering.
func (e *Engine) rawData(name string) map[string]any {
	data := make(map[string]any, len(e.data)+1)
	data["Name"] = name
	for k, v := range e.data {
		data[k] = v
	}
	return data
}

// shellData returns template data with shell-quoted string values — used for command rendering.
func (e *Engine) shellData(name string) map[string]any {
	data := make(map[string]any, len(e.data)+1)
	data["Name"] = shellQuote(name)
	for k, v := range e.data {
		if s, ok := v.(string); ok {
			data[k] = shellQuote(s)
		} else {
			data[k] = v
		}
	}
	return data
}

// renderCached parses and caches compiled templates to avoid redundant parsing.
func (e *Engine) renderCached(cmdTemplate string, data map[string]any) (string, error) {
	tmpl, ok := e.tmplCache[cmdTemplate]
	if !ok {
		var err error
		tmpl, err = template.New("cmd").Option("missingkey=error").Parse(cmdTemplate)
		if err != nil {
			return "", fmt.Errorf("parse template: %w", err)
		}
		if e.tmplCache == nil {
			e.tmplCache = make(map[string]*template.Template)
		}
		e.tmplCache[cmdTemplate] = tmpl
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

func (e *Engine) Apply(dryRun bool) error {
	prev, err := manifest.Load(e.configDir)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	desiredPkgs := make(map[string]string) // name -> manager
	pkgEntries := make([]manifest.PackageEntry, 0)

	// Install desired packages
	for _, pkg := range e.cfg.Packages() {
		name, ok, err := e.renderName(pkg.Name)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		desiredPkgs[name] = pkg.Manager
		pkgEntries = append(pkgEntries, manifest.PackageEntry{
			Name:    name,
			Manager: pkg.Manager,
		})

		mgr := e.cfg.Managers[pkg.Manager]

		if dryRun {
			fmt.Printf("[DRY RUN] Would check and potentially install: %s (%s)\n", name, pkg.Manager)
			continue
		}

		installed, err := e.check(mgr.Check, name)
		if err != nil {
			return fmt.Errorf("check %s: %w", name, err)
		}

		if !installed {
			if err := e.run(mgr.Install, name); err != nil {
				return fmt.Errorf("install %s: %w", name, err)
			}
			fmt.Printf("Installed: %s (%s)\n", name, pkg.Manager)
		}
	}

	// Remove obsolete packages
	var removeErrs []error
	removedOk := make(map[string]bool)

	for _, entry := range prev.Packages {
		if _, ok := desiredPkgs[entry.Name]; ok {
			continue
		}
		mgr, ok := e.cfg.Managers[entry.Manager]
		if !ok {
			fmt.Fprintf(os.Stderr, "WARN: manager %q for %s not found, skipping\n", entry.Manager, entry.Name)
			continue
		}

		if dryRun {
			fmt.Printf("[DRY RUN] Would check and potentially remove: %s (%s)\n", entry.Name, entry.Manager)
			continue
		}

		installed, err := e.check(mgr.Check, entry.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: check %s: %v\n", entry.Name, err)
			continue
		}
		if installed {
			if err := e.run(mgr.Remove, entry.Name); err != nil {
				fmt.Fprintf(os.Stderr, "WARN: remove %s: %v\n", entry.Name, err)
				removeErrs = append(removeErrs, fmt.Errorf("remove %s (%s): %w", entry.Name, entry.Manager, err))
				continue
			}
			removedOk[entry.Name] = true
			fmt.Printf("Removed: %s (%s)\n", entry.Name, entry.Manager)
		}
	}

	// Collect desired service names
	desiredSvcs := make(map[string]string) // name -> manager
	svcEntries := make([]manifest.ServiceEntry, 0)

	// Enable desired services
	for _, svc := range e.cfg.Services() {
		name, ok, err := e.renderName(svc.Name)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		desiredSvcs[name] = svc.Manager
		svcEntries = append(svcEntries, manifest.ServiceEntry{
			Name:    name,
			Manager: svc.Manager,
		})

		mgr := e.cfg.Managers[svc.Manager]

		if dryRun {
			fmt.Printf("[DRY RUN] Would check and potentially enable: %s (%s)\n", name, svc.Manager)
			continue
		}

		enabled, err := e.check(mgr.Check, name)
		if err != nil {
			return fmt.Errorf("check service %s: %w", name, err)
		}

		if !enabled {
			if err := e.run(mgr.Enable, name); err != nil {
				return fmt.Errorf("enable %s: %w", name, err)
			}
			fmt.Printf("Enabled: %s (%s)\n", name, svc.Manager)
		}
	}

	// Disable obsolete services
	for _, entry := range prev.Services {
		if _, ok := desiredSvcs[entry.Name]; ok {
			continue
		}
		mgr, ok := e.cfg.Managers[entry.Manager]
		if !ok {
			fmt.Fprintf(os.Stderr, "WARN: manager %q for service %s not found, skipping\n", entry.Manager, entry.Name)
			continue
		}

		if dryRun {
			fmt.Printf("[DRY RUN] Would check and potentially disable: %s (%s)\n", entry.Name, entry.Manager)
			continue
		}

		enabled, err := e.check(mgr.Check, entry.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: check service %s: %v\n", entry.Name, err)
			continue
		}
		if enabled {
			if err := e.run(mgr.Disable, entry.Name); err != nil {
				fmt.Fprintf(os.Stderr, "WARN: disable %s: %v\n", entry.Name, err)
				removeErrs = append(removeErrs, fmt.Errorf("disable %s (%s): %w", entry.Name, entry.Manager, err))
				continue
			}
			fmt.Printf("Disabled: %s (%s)\n", entry.Name, entry.Manager)
		}
	}

	// Block manifest save if any removals failed
	if len(removeErrs) > 0 {
		return fmt.Errorf("failed to remove/disable %d item(s), manifest not saved: fix the errors and re-run apply", len(removeErrs))
	}

	// Keep obsolete items that were NOT removed (failed or not installed) in manifest
	for _, entry := range prev.Packages {
		if _, desired := desiredPkgs[entry.Name]; desired {
			continue
		}
		if removedOk[entry.Name] {
			continue
		}
		pkgEntries = append(pkgEntries, manifest.PackageEntry{
			Name:    entry.Name,
			Manager: entry.Manager,
		})
	}
	for _, entry := range prev.Services {
		if _, desired := desiredSvcs[entry.Name]; desired {
			continue
		}
		svcEntries = append(svcEntries, manifest.ServiceEntry{
			Name:    entry.Name,
			Manager: entry.Manager,
		})
	}

	newManifest := &manifest.Manifest{
		Packages: pkgEntries,
		Services: svcEntries,
	}
	if err := manifest.Save(e.configDir, newManifest); err != nil {
		return fmt.Errorf("save manifest: %w", err)
	}

	return nil
}

func (e *Engine) Status() error {
	desiredPkgs := make(map[string]bool)

	fmt.Println("Packages:")
	for _, pkg := range e.cfg.Packages() {
		name, ok, err := e.renderName(pkg.Name)
		if err != nil {
			fmt.Printf("  ?        %s (%s) — %v\n", pkg.Name, pkg.Manager, err)
			continue
		}
		if !ok {
			fmt.Printf("  SKIP     %s (%s) — template rendered empty\n", pkg.Name, pkg.Manager)
			continue
		}
		desiredPkgs[name] = true

		mgr := e.cfg.Managers[pkg.Manager]
		installed, err := e.check(mgr.Check, name)
		if err != nil {
			fmt.Printf("  ?        %s (%s) — check error: %v\n", name, pkg.Manager, err)
			continue
		}
		if installed {
			fmt.Printf("  OK       %s (%s)\n", name, pkg.Manager)
		} else {
			fmt.Printf("  MISSING  %s (%s)\n", name, pkg.Manager)
		}
	}

	// Obsolete packages
	prev, err := manifest.Load(e.configDir)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}
	for _, entry := range prev.Packages {
		if desiredPkgs[entry.Name] {
			continue
		}
		mgr, ok := e.cfg.Managers[entry.Manager]
		if !ok {
			fmt.Fprintf(os.Stderr, "WARN: manager %q for %s not found, skipping\n", entry.Manager, entry.Name)
			continue
		}
		installed, err := e.check(mgr.Check, entry.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: check %s: %v\n", entry.Name, err)
			continue
		}
		if installed {
			fmt.Printf("  OBSOLETE %s (%s) — still installed\n", entry.Name, entry.Manager)
		}
	}

	// Services
	desiredSvcs := make(map[string]bool)
	fmt.Println("\nServices:")
	for _, svc := range e.cfg.Services() {
		name, ok, err := e.renderName(svc.Name)
		if err != nil {
			fmt.Printf("  ?        %s (%s) — %v\n", svc.Name, svc.Manager, err)
			continue
		}
		if !ok {
			fmt.Printf("  SKIP     %s (%s) — template rendered empty\n", svc.Name, svc.Manager)
			continue
		}
		desiredSvcs[name] = true

		mgr, ok := e.cfg.Managers[svc.Manager]
		if !ok {
			fmt.Printf("  ?        %s (manager %q not found)\n", name, svc.Manager)
			continue
		}
		enabled, err := e.check(mgr.Check, name)
		if err != nil {
			fmt.Printf("  ?        %s (%s) — check error: %v\n", name, svc.Manager, err)
			continue
		}
		if enabled {
			fmt.Printf("  ENABLED  %s (%s)\n", name, svc.Manager)
		} else {
			fmt.Printf("  DISABLED %s (%s)\n", name, svc.Manager)
		}
	}

	// Obsolete services
	for _, entry := range prev.Services {
		if desiredSvcs[entry.Name] {
			continue
		}
		mgr, ok := e.cfg.Managers[entry.Manager]
		if !ok {
			fmt.Fprintf(os.Stderr, "WARN: manager %q for service %s not found, skipping\n", entry.Manager, entry.Name)
			continue
		}
		enabled, err := e.check(mgr.Check, entry.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: check service %s: %v\n", entry.Name, err)
			continue
		}
		if enabled {
			fmt.Printf("  OBSOLETE %s (%s) — still enabled\n", entry.Name, entry.Manager)
		}
	}

	return nil
}

func (e *Engine) check(cmdTemplate string, name string) (bool, error) {
	data := e.shellData(name)
	cmd, err := e.renderCached(cmdTemplate, data)
	if err != nil {
		return false, fmt.Errorf("render template: %w", err)
	}
	c := exec.Command("bash", "-c", cmd)
	err = c.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return false, nil // non-zero exit = not installed/enabled
		}
		return false, fmt.Errorf("check command failed to execute: %w", err)
	}
	return true, nil
}

func (e *Engine) run(cmdTemplate string, name string) error {
	data := e.shellData(name)
	cmd, err := e.renderCached(cmdTemplate, data)
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}
	c := exec.Command("bash", "-c", cmd)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}

func renderTemplate(cmdTemplate string, name string, data map[string]any) (string, error) {
	tmpl, err := template.New("cmd").Option("missingkey=error").Parse(cmdTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

// renderName renders a name template and returns the result.
// If the name contains no template expressions, it is returned as-is.
// If the rendered result is empty, returns ("", false, nil).
// If the template fails to parse/execute, returns ("", false, error).
func (e *Engine) renderName(name string) (string, bool, error) {
	if !strings.Contains(name, "{{") {
		return name, name != "", nil
	}

	data := e.rawData("name")
	result, err := e.renderCached(name, data)
	if err != nil {
		return "", false, fmt.Errorf("render %q: %w", name, err)
	}
	result = strings.TrimSpace(result)
	return result, result != "", nil
}
