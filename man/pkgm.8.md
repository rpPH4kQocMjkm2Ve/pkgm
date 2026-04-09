---
title: PKGM
section: 8
header: System Administration
footer: pkgm
---

# NAME

pkgm — declarative package and service manager

# SYNOPSIS

**pkgm** **init**

**pkgm** **apply** [**-n**|**--dry-run**]

**pkgm** **status**

**pkgm** **reset** *NAME*...

**pkgm** **reset** **--all**

**pkgm** **version**

**pkgm** [**-h**|**--help**]

**pkgm** [**-V**|**--version**]

# DESCRIPTION

**pkgm** is a declarative package and service manager driven by a single
**pkgm.toml** manifest. It installs desired packages, removes obsolete ones,
and manages services — with zero hardcoded package managers or init systems.

Configuration is read from **pkgm.toml**, searched from the current
directory upward. The desired state is compared against a previously saved
manifest stored in **~/.local/state/pkgm/**.

# COMMANDS

**init**
:   Run interactive prompts defined in **[prompts]**, save answers to
    state cache. Must be run at least once before **apply** if prompts are
    defined. Re-run to re-answer prompts.

**apply**
:   Install desired packages and enable services defined in **pkgm.toml**.
    Remove obsolete packages and disable services that were previously
    installed but are no longer in the config.

    **-n**, **--dry-run**
    :   Preview what would happen without making any changes.

**status**
:   Show the current system state compared to the desired configuration.
    Reports packages as **OK**, **MISSING**, **SKIP**, or **OBSOLETE**, and
    services as **ENABLED**, **DISABLED**, **SKIP**, or **OBSOLETE**.

**reset** *NAME*...
:   Reset cached prompt values by name. Removes the specified entries from
    the state cache so they will be prompted again on next **init**.

    **--all**
    :   Reset all cached prompt values at once. Cannot be mixed with names.

**version**
:   Print program name and version, then exit. Same as **-V**/**--version**.

# OPTIONS

**-h**, **--help**
:   Show usage information and exit.

**-V**, **--version**
:   Print program name and version, then exit.

# CONFIGURATION

Create **pkgm.toml** in your project or dotfiles repository. Define managers
under **[managers]**, then declare desired packages and services under group
sections matching the manager name.

## Manager definition

Each manager specifies shell commands for checking, installing, removing,
enabling, and disabling:

```toml
[managers.pacman]
check   = "pacman -Q {{.Name}}"
install = "sudo pacman -S --needed {{.Name}}"
remove  = "sudo pacman -Rns {{.Name}}"

[managers.systemd]
check   = "systemctl is-enabled {{.Name}}"
enable  = "systemctl enable {{.Name}}"
disable = "systemctl disable {{.Name}}"
```

The **{{.Name}}** template variable is substituted with the package or
service name.

## Prompts

Interactive prompts store user preferences in state. Values are available
as template variables (e.g., **.laptop**, **.nvidia**) for conditions
on packages and services.

```toml
[prompts]
laptop = { type = "bool", question = "Is this a laptop?" }
tablet = { type = "bool", question = "Do you use a tablet?" }
git_email = { type = "string", question = "Git email address" }
```

Types: **bool** (y/n), **string** (free text). Values are cached in
**~/.local/state/pkgm/** and reused across runs. Run **pkgm init** to
re-answer prompts.

## Package declaration

```toml
[pacman]
packages = ["git", "zsh", "neovim"]
```

Per-entry conditions are supported:

```toml
[pacman]
packages = [
    "git",
    "{{ if .nvidia }}nvidia-utils{{ end }}",
]
```

## Service declaration

```toml
[systemd]
services = ["firewalld", "sshd"]
```

## Conditions

Package and service names can contain Go template expressions. If a name
renders to an empty string, the entry is skipped. All prompt values are
available as template variables:

```toml
[prompts]
nvidia = { type = "bool", question = "Use NVIDIA config?" }

[pacman]
packages = [
    "base",
    "{{ if .nvidia }}nvidia-open{{ end }}",
]
```

# EXAMPLES

Initialize prompts interactively:

    pkgm init

Declarative system upgrade:

    cd ~/projects/dotfiles
    pkgm apply

Preview changes:

    pkgm apply --dry-run

Check current state:

    pkgm status

# EXIT STATUS

**0**
:   Success.

**1**
:   Error. Common causes: unknown command, config parse failure, template
    error, missing manager definition.

# FILES

**~/.local/state/pkgm/\*.toml**
:   Per-directory manifest state files. The filename is the first 16 hex
    characters (first 8 bytes) of the SHA-256 hash of the config directory
    path.

# SEE ALSO

**gitpkg**(1)
