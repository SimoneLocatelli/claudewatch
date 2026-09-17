package statusline

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/SimoneLocatelli/claudewatch/internal/api"
	"github.com/SimoneLocatelli/claudewatch/internal/config"
	"github.com/SimoneLocatelli/claudewatch/internal/theme"
)

// ClaudeStatus represents the JSON piped from Claude Code.
type ClaudeStatus struct {
	SessionID     string        `json:"session_id"`
	Model         modelInfo     `json:"model"`
	ContextWindow contextWindow `json:"context_window"`
	Cost          costInfo      `json:"cost"`

	// Populated by main after parsing — not from JSON.
	Cwd    string `json:"-"`
	Branch string `json:"-"`
}

type modelInfo struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type contextWindow struct {
	UsedPercentage *float64 `json:"used_percentage"`
}

type costInfo struct {
	TotalCostUSD float64 `json:"total_cost_usd"`
}

// Parse decodes Claude Code's JSON status.
func Parse(data []byte) (ClaudeStatus, error) {
	var s ClaudeStatus
	if err := json.Unmarshal(data, &s); err != nil {
		return ClaudeStatus{}, fmt.Errorf("parsing status JSON: %w", err)
	}
	return s, nil
}

// ANSI helpers — raw escape codes so colors work in pipes.
const reset = "\x1b[0m"
const dim = "\x1b[2m"

func fg(hex string) string {
	r, g, b := hexToRGB(hex)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

func hexToRGB(hex string) (r, g, b int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 255, 255, 255
	}
	rv, _ := strconv.ParseInt(hex[0:2], 16, 64)
	gv, _ := strconv.ParseInt(hex[2:4], 16, 64)
	bv, _ := strconv.ParseInt(hex[4:6], 16, 64)
	return int(rv), int(gv), int(bv)
}

// Render produces the themed status line with raw ANSI codes.
func Render(s *ClaudeStatus, t *theme.Theme, plan string, usage *api.Usage, cfg *config.Config) string {
	pipe := dim + fg(t.Colors.Muted) + " | " + reset

	showPlan := plan != "" && config.Enabled(cfg.ShowPlan)
	parts := []string{
		renderModel(s, t, plan, showPlan),
		renderContext(s, t),
	}

	if usage != nil {
		if usage.FiveHour != nil && config.Enabled(cfg.Show5h) {
			parts = append(parts, renderQuota("5h", usage.FiveHour, t))
		}
		if usage.SevenDay != nil && config.Enabled(cfg.Show7d) {
			parts = append(parts, renderQuota("7d", usage.SevenDay, t))
		}
		if config.Enabled(cfg.ShowExtra) {
			if extra := renderExtra(usage.ExtraUsage, t); extra != "" {
				parts = append(parts, extra)
			}
		}
	}

	if config.Enabled(cfg.ShowCost) {
		if cost := renderCost(s, t); cost != "" {
			parts = append(parts, cost)
		}
	}

	if s.Cwd != "" && config.Enabled(cfg.ShowCwd) {
		parts = append(parts, renderCwd(s, t))
	}

	if s.Branch != "" && config.Enabled(cfg.ShowBranch) {
		parts = append(parts, renderBranch(s, t))
	}

	return strings.Join(parts, pipe)
}

func renderModel(s *ClaudeStatus, t *theme.Theme, plan string, showPlan bool) string {
	name := s.Model.DisplayName
	if name == "" {
		name = s.Model.ID
	}

	result := fg(t.Colors.Muted) + "[" + reset +
		fg(t.Colors.Accent) + name + reset
	if showPlan {
		result += fg(t.Colors.Muted) + " | " + reset +
			fg(t.Colors.Fg) + plan + reset
	}
	result += fg(t.Colors.Muted) + "]" + reset
	return result
}

func renderContext(s *ClaudeStatus, t *theme.Theme) string {
	var pct float64
	if s.ContextWindow.UsedPercentage != nil {
		pct = *s.ContextWindow.UsedPercentage
	}
	color := barColor(pct, t)
	return fg(t.Colors.Muted) + "ctx " + reset +
		progressBar(pct, 3, color, t.Colors.Muted) + " " +
		fg(color) + fmt.Sprintf("%.0f%%", pct) + reset
}

func renderQuota(label string, q *api.QuotaLimit, t *theme.Theme) string {
	pct := q.Utilization
	color := barColor(pct, t)
	result := fg(t.Colors.Muted) + label + " " + reset +
		progressBar(pct, 3, color, t.Colors.Muted) + " " +
		fg(color) + fmt.Sprintf("%.0f%%", pct) + reset

	if rs := formatReset(q.ResetsAt); rs != "" {
		result += " " + fg(t.Colors.Muted) + rs + reset
	}
	return result
}

func renderExtra(extra *api.ExtraUsage, t *theme.Theme) string {
	if extra == nil || !extra.IsEnabled || extra.MonthlyLimit == nil || extra.UsedCredits == nil {
		return ""
	}
	used := int(*extra.UsedCredits) / 100
	limit := int(*extra.MonthlyLimit) / 100
	if used == 0 {
		return ""
	}
	pct := float64(used) / float64(limit) * 100
	color := barColor(pct, t)
	return fg(color) + fmt.Sprintf("$%d", used) + reset +
		fg(t.Colors.Muted) + fmt.Sprintf("/$%d", limit) + reset
}

func renderCost(s *ClaudeStatus, t *theme.Theme) string {
	cost := s.Cost.TotalCostUSD
	if cost <= 0 {
		return ""
	}
	var formatted string
	switch {
	case cost < 0.01:
		formatted = fmt.Sprintf("$%.4f", cost)
	default:
		formatted = fmt.Sprintf("$%.2f", cost)
	}
	return fg(t.Colors.Muted) + "cost " + reset +
		fg(t.Colors.Fg) + formatted + reset
}

func renderCwd(s *ClaudeStatus, t *theme.Theme) string {
	return dim + fg(t.Colors.Muted) + "\uf07b " + s.Cwd + reset
}

func renderBranch(s *ClaudeStatus, t *theme.Theme) string {
	return dim + fg(t.Colors.Muted) + "\ue725 " + s.Branch + reset
}

// progressBar renders filled and empty blocks with foreground colors.
func progressBar(pct float64, width int, fillColor, emptyColor string) string {
	filled := int(math.Round(pct / 100.0 * float64(width)))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled

	return fg(fillColor) + strings.Repeat("\u2588", filled) + reset +
		dim + fg(emptyColor) + strings.Repeat("\u2591", empty) + reset
}

func barColor(pct float64, t *theme.Theme) string {
	switch {
	case pct >= 80:
		return t.Colors.Error
	case pct >= 50:
		return t.Colors.Warning
	default:
		return t.Colors.Info
	}
}

func formatReset(iso string) string {
	if iso == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return ""
	}
	remaining := time.Until(t)
	if remaining <= 0 {
		return ""
	}
	if remaining < 24*time.Hour {
		h := int(remaining.Hours())
		m := int(remaining.Minutes()) % 60
		if h > 0 {
			return fmt.Sprintf("%dh%02dm", h, m)
		}
		return fmt.Sprintf("%dm", m)
	}
	return t.Local().Weekday().String()[:3]
}
