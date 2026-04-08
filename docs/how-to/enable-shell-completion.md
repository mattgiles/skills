# Enable Shell Completion

Use `skills completion` to generate shell completion scripts for supported shells.

## Supported Shells

```text
skills completion bash
skills completion fish
skills completion powershell
skills completion zsh
```

## Load Completion In The Current Shell

Bash:

```bash
source <(skills completion bash)
```

Zsh:

```bash
source <(skills completion zsh)
```

Fish:

```fish
skills completion fish | source
```

PowerShell:

```powershell
skills completion powershell | Out-String | Invoke-Expression
```

## Install Persistent Zsh Completion On macOS

```bash
mkdir -p "$(brew --prefix)/share/zsh/site-functions"
skills completion zsh > "$(brew --prefix)/share/zsh/site-functions/_skills"
```

If shell completion is not already enabled in your zsh environment, initialize it once in `~/.zshrc`:

```bash
autoload -U compinit
compinit
```

## Get Shell-Specific Setup Help

Each shell subcommand includes its own help text:

```bash
skills completion zsh --help
skills completion bash --help
```

Use that help output as the source of truth for shell-specific install details.
