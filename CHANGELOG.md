# Changelog

## v0.4.1 (2026-09-17)

- Fix: `claudewatch install` now writes the `statusLine` command path with forward slashes. On Windows, Claude Code runs the command through Git Bash, which stripped the backslashes and the status line never appeared.
- Bump pinned install target to `v0.4.1` (`go install`, `update` command, setup skill)

## v0.4.0

- `update` command — fetch latest version and re-register (`claudewatch update`)
- `version` command — print installed version (`claudewatch version`)
- `/claudewatch:update` skill for updating via Claude Code
- Install now creates config file with all keys explicitly set
- `show_cost` defaults to `false` (was `true`)
- Uninstall preserves config file for future reinstalls
- `/claudewatch:config` skill now covers all settings — theme + all segment toggles

## v0.3.0

- Session cost display (`cost $1.23`) from Claude Code status JSON
- Working directory and git branch segments (off by default)
- Configurable segments via `config.toml` — toggle plan, 5h, 7d, extra, cost, cwd, branch
- Claude Code plugin marketplace support (`marketplace.json`)
- Cross-platform releases: Windows (amd64, arm64) and Linux ARM64
- Release script with patch/minor/major version bumping and changelog

## v0.2.0

- Real-time usage data from Anthropic API (5h window, 7d limits)
- Auto-detect plan type (Pro, Max, Team, Enterprise) from credentials
- Labels on bars (ctx, 5h, 7d) for clarity
- Extra usage (pay-as-you-go) display
- Color thresholds: blue < 50%, orange 50-80%, red > 80%
- Uninstall command
- Claude Code plugin with setup, config, and uninstall skills
- 10 built-in themes with true-color support

## v0.1.0

- Initial release
- Themed status line for Claude Code
- Context window, 5-hour, and weekly usage bars
- 10 built-in themes
- Install/uninstall commands
