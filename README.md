# pkgm

Declarative package and service manager. Repository is the source of truth — no local state beyond a manifest for diffing. Zero hardcoded package managers or init systems.

## Usage

```bash
cd ~/projects/dotfiles   # where pkgm.toml lives
pkgm init                # answer prompts (if any)
pkgm apply               # install desired, remove obsolete
pkgm apply --dry-run     # preview changes
pkgm status              # current vs desired state
pkgm version             # print version
```

## Configuration

Create `pkgm.toml` in your dotfiles repository:

```toml
# ─── Managers ───

[managers.pacman]
check   = "pacman -Q {{.Name}}"
install = "sudo pacman -S --needed {{.Name}}"
remove  = "sudo pacman -Rns {{.Name}}"

[managers.aur]
check   = "pacman -Q {{.Name}}"
install = "aur sync --no-view -n {{.Name}} && sudo pacman -S --needed {{.Name}}"
remove  = "sudo pacman -Rns {{.Name}}"

[managers.systemd]
check   = "systemctl is-enabled {{.Name}}"
enable  = "systemctl enable {{.Name}}"
disable = "systemctl disable {{.Name}}"

[managers.systemd-user]
check   = "systemctl --user is-enabled {{.Name}}"
enable  = "systemctl --user enable {{.Name}}"
disable = "systemctl --user disable {{.Name}}"

# ─── Packages ───

[pacman]
packages = [
    "hyprland", "neovim", "zsh", "pipewire", "pipewire-pulse",
]

[aur]
packages = ["kopia-bin", "coolercontrol-bin"]

# ─── Services ───

[systemd]
services = [
    "firewalld",
    "systemd-oomd",
]

[systemd-user]
services = [
    "hypridle",
    "waybar",
    "mpd",
]
```

## How it works

1. `pkgm apply` reads `pkgm.toml` and loads the previous manifest
2. For each package: runs `check` → if not installed, runs `install`
3. For each package in manifest but not in config: if still installed, runs `remove`
4. For each service: runs `check` → if not enabled, runs `enable`
5. For each service in manifest but not in config: if still enabled, runs `disable`
6. Saves new manifest to `~/.local/state/pkgm/<hash>.toml`

## Adding a manager

Add a section to `[managers]` — no code changes needed:

```toml
# Package manager
[managers.pip]
check   = "pip list 2>/dev/null | grep -q '{{.Name}}'"
install = "pip install --user {{.Name}}"
remove  = "pip uninstall -y {{.Name}}"

# Service manager (openrc example)
[managers.openrc]
check   = "rc-status | grep -q '{{.Name}}'"
enable  = "rc-update add {{.Name}} default"
disable = "rc-update del {{.Name}}"

# Then use it:
[openrc]
services = ["sshd", "NetworkManager"]
```

## Conditions

Package and service names can contain Go template expressions. If a name
renders to an empty string, the entry is skipped:

```toml
[pacman]
packages = [
    "git",
    "{{ if .nvidia }}nvidia-utils{{ end }}",
    "{{ if .tablet }}opentabletdriver{{ end }}",
]
```

All prompt values are available as template variables:

```toml
[prompts]
nvidia = { type = "bool", question = "Use NVIDIA config?" }
tablet = { type = "bool", question = "Do you use a tablet?" }

[pacman]
packages = [
    "base",
    "{{ if .nvidia }}nvidia-open{{ end }}",
]
```

## Prompts

Interactive prompts store user preferences in state. Run `pkgm init` to
answer them.

```toml
[prompts]
laptop = { type = "bool", question = "Is this a laptop?" }
git_email = { type = "string", question = "Git email address" }
```

Types: `bool` (y/n), `string` (free text). Values are cached in
`~/.local/state/pkgm/` and reused across runs.

## Install

```bash
git clone https://gitlab.com/fkzys/pkgm.git
cd pkgm
sudo make install
```

## License

AGPL-3.0-or-later
