---
name: setup
description: Install and configure claudewatch status line for Claude Code
allowed-tools: Bash, Read, Write, AskUserQuestion
---

Install claudewatch — a themed status line for Claude Code.

## Steps

1. **Check Go**: Verify Go is installed (`go version`). If not installed, tell the user to install Go first from https://go.dev/dl/ and stop.

2. **Install binary**: Run `go install github.com/SimoneLocatelli/claudewatch@v0.4.0`

3. **Register with Claude Code**: Run `claudewatch install` — this also creates the default config file with all keys.

4. **Verify**: Read `~/.claude/settings.json` and confirm statusLine is set.

5. **Choose theme**: Ask which theme they want:
   Options: dracula (default), catppuccin-mocha, catppuccin-latte, nord, tokyo-night, gruvbox, solarized-dark, solarized-light, one-dark, rosepine

6. **Write config**: Read the existing config from `~/.config/claudewatch/config.toml`, update only the theme value, and write it back preserving all other keys.

7. **Done**: Tell the user to restart Claude Code. Plan type and usage limits are auto-detected from credentials. Mention they can run `/claudewatch:config` to toggle individual segments on/off.
