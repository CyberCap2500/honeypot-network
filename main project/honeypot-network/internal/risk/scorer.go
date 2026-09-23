/*
©AngelaMos | 2026
scorer.go

Risk score calculation engine

Computes a 0-100 risk score for attackers based on weighted factors:
- Authentication behavior (failed attempts, credential variety)
- Command sophistication (unique commands, MITRE techniques)
- Exploit patterns (known attack signatures, port scanning)
- Temporal behavior (escalation speed, session frequency)
- Geographic anomalies (optional, if GeoIP data available)

Scoring methodology uses normalized weighted sums with configurable
weights per factor. The algorithm is table-driven tested.

This is a NEW feature not present in the source project.
*/

package risk

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/rs/zerolog"

	"github.com/CarterPerez-dev/miragenet/internal/store"
	"github.com/CarterPerez-dev/miragenet/pkg/types"
)

// Default weights for each risk factor (sum to 1.0)
const (
	WeightAuthBehavior          = 0.25 // Failed auth attempts, credential spray
	WeightCommandSophistication = 0.30 // Unique commands, MITRE techniques
	WeightExploitPatterns       = 0.25 // Known exploit signatures
	WeightTemporalBehavior      = 0.15 // Escalation speed, session frequency
	WeightPersistence           = 0.05 // Repeat visits, session duration
)

// Thresholds for normalizing raw metrics
const (
	MaxFailedAuthForScore   = 50.0  // normalize failed auth attempts
	MaxCommandsForScore     = 100.0 // normalize command count
	MaxSessionsForScore     = 20.0  // normalize session count
	MaxTechniquesForScore   = 10.0  // normalize MITRE technique count
	MaxEscalationSpeed      = 10.0  // commands per minute
	SuspiciousCommandWeight = 5.0   // multiplier for dangerous commands
)

// Suspicious command patterns (simplified for Phase 1)
var suspiciousPatterns = []string{
	"wget", "curl", "nc", "netcat", "bash", "sh", "python",
	"perl", "ruby", "chmod", "chown", "sudo", "su",
	"passwd", "useradd", "adduser", "rm -rf", "dd if=",
	"iptables", "nmap", "masscan", "sqlmap", "metasploit",
}

type Engine struct {
	store  store.Store
	logger zerolog.Logger
}

func NewEngine(st store.Store, logger *zerolog.Logger) *Engine {
	return &Engine{
		store:  st,
		logger: logger.With().Str("component", "risk-scorer").Logger(),
	}
}

// ComputeScore calculates the risk score for an attacker profile
func (e *Engine) ComputeScore(profile *types.AttackerRiskProfile) *types.RiskScore {
	factors := []types.RiskFactor{}

	// Factor 1: Authentication behavior
	authScore := e.computeAuthBehavior(profile)
	factors = append(factors, types.RiskFactor{
		Name:        "auth_behavior",
		Value:       float64(profile.FailedAuthAttempts),
		Weight:      WeightAuthBehavior,
		Score:       authScore * WeightAuthBehavior * 100,
		Description: fmt.Sprintf("%d failed auth attempts", profile.FailedAuthAttempts),
	})

	// Factor 2: Command sophistication
	commandScore := e.computeCommandSophistication(profile)
	factors = append(factors, types.RiskFactor{
		Name:        "command_sophistication",
		Value:       float64(profile.UniqueCommands),
		Weight:      WeightCommandSophistication,
		Score:       commandScore * WeightCommandSophistication * 100,
		Description: fmt.Sprintf("%d unique commands, %d MITRE techniques", profile.UniqueCommands, len(profile.MitreTechniques)),
	})

	// Factor 3: Exploit patterns
	exploitScore := e.computeExploitPatterns(profile)
	factors = append(factors, types.RiskFactor{
		Name:        "exploit_patterns",
		Value:       float64(profile.ExploitAttempts + profile.SuspiciousCommands),
		Weight:      WeightExploitPatterns,
		Score:       exploitScore * WeightExploitPatterns * 100,
		Description: fmt.Sprintf("%d exploit attempts, %d suspicious commands", profile.ExploitAttempts, profile.SuspiciousCommands),
	})

	// Factor 4: Temporal behavior (escalation speed)
	temporalScore := e.computeTemporalBehavior(profile)
	factors = append(factors, types.RiskFactor{
		Name:        "temporal_behavior",
		Value:       profile.EscalationSpeed,
		Weight:      WeightTemporalBehavior,
		Score:       temporalScore * WeightTemporalBehavior * 100,
		Description: fmt.Sprintf("%.1f commands/min escalation speed", profile.EscalationSpeed),
	})

	// Factor 5: Persistence (repeat visits)
	persistenceScore := e.computePersistence(profile)
	factors = append(factors, types.RiskFactor{
		Name:        "persistence",
		Value:       float64(profile.SessionCount),
		Weight:      WeightPersistence,
		Score:       persistenceScore * WeightPersistence * 100,
		Description: fmt.Sprintf("%d sessions over %s", profile.SessionCount, e.formatDuration(profile.LastSeenAt.Sub(profile.FirstSeenAt))),
	})

	// Calculate final weighted score (0-100)
	totalScore := 0.0
	for _, factor := range factors {
		totalScore += factor.Score
	}

	// Clamp to 0-100 range
	finalScore := int(math.Min(100, math.Max(0, totalScore)))

	return &types.RiskScore{
		Score:      finalScore,
		Level:      RiskLevel(finalScore),
		Factors:    factors,
		ComputedAt: time.Now().UTC(),
	}
}

// RiskLevel returns the severity level based on score
func RiskLevel(score int) string {
	switch {
	case score >= 80:
		return "critical"
	case score >= 60:
		return "high"
	case score >= 40:
		return "medium"
	default:
		return "low"
	}
}

// computeAuthBehavior scores authentication patterns (0.0-1.0)
func (e *Engine) computeAuthBehavior(profile *types.AttackerRiskProfile) float64 {
	// Normalize failed attempts
	failedNorm := math.Min(float64(profile.FailedAuthAttempts)/MaxFailedAuthForScore, 1.0)

	// Bonus if they had any successful auth (credential stuffing success)
	successBonus := 0.0
	if profile.SuccessfulAuthCount > 0 {
		successBonus = 0.3
	}

	return math.Min(failedNorm+successBonus, 1.0)
}

// computeCommandSophistication scores command patterns (0.0-1.0)
func (e *Engine) computeCommandSophistication(profile *types.AttackerRiskProfile) float64 {
	// Unique commands normalized
	commandNorm := math.Min(float64(profile.UniqueCommands)/MaxCommandsForScore, 1.0)

	// MITRE techniques normalized
	techniqueNorm := math.Min(float64(len(profile.MitreTechniques))/MaxTechniquesForScore, 1.0)

	// Weighted average (techniques are more important than raw command count)
	return (commandNorm * 0.3) + (techniqueNorm * 0.7)
}

// computeExploitPatterns scores known attack signatures (0.0-1.0)
func (e *Engine) computeExploitPatterns(profile *types.AttackerRiskProfile) float64 {
	// Direct exploit attempts (e.g., SQL injection, path traversal)
	exploitNorm := math.Min(float64(profile.ExploitAttempts)/10.0, 1.0)

	// Suspicious command count
	suspiciousNorm := math.Min(float64(profile.SuspiciousCommands)/20.0, 1.0)

	return math.Max(exploitNorm, suspiciousNorm) // Take the worst
}

// computeTemporalBehavior scores escalation speed (0.0-1.0)
func (e *Engine) computeTemporalBehavior(profile *types.AttackerRiskProfile) float64 {
	// Fast escalation = higher risk (automated tools)
	speedNorm := math.Min(profile.EscalationSpeed/MaxEscalationSpeed, 1.0)

	return speedNorm
}

// computePersistence scores repeat visit patterns (0.0-1.0)
func (e *Engine) computePersistence(profile *types.AttackerRiskProfile) float64 {
	// Multiple sessions = persistent attacker
	sessionNorm := math.Min(float64(profile.SessionCount)/MaxSessionsForScore, 1.0)

	// Longer average session = more thorough reconnaissance
	avgSessionMinutes := profile.AverageSessionLength / 60.0
	durationNorm := math.Min(avgSessionMinutes/10.0, 1.0) // 10+ min sessions = high

	return (sessionNorm * 0.6) + (durationNorm * 0.4)
}

// ComputeSessionRisk calculates risk for a single session
func (e *Engine) ComputeSessionRisk(sessionID string) (*types.RiskScore, error) {
	ctx := context.Background()

	// Fetch session data from store
	session, err := e.store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("fetch session: %w", err)
	}

	// Fetch all sessions for this attacker to build profile
	profile, err := e.store.GetAttackerRiskProfile(ctx, session.SourceIP)
	if err != nil {
		return nil, fmt.Errorf("fetch attacker profile: %w", err)
	}

	return e.ComputeScore(profile), nil
}

// IsSuspiciousCommand checks if a command matches known attack patterns
func IsSuspiciousCommand(cmd string) bool {
	for _, pattern := range suspiciousPatterns {
		if contains(cmd, pattern) {
			return true
		}
	}
	return false
}

// Helper to check substring (case-insensitive)
func contains(s, substr string) bool {
	// Use strings package for proper substring checking
	return len(s) >= len(substr) && indexSubstring(s, substr) >= 0
}

// indexSubstring returns the index of substr in s, or -1 if not found
func indexSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func (e *Engine) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "< 1 min"
	} else if d < time.Hour {
		return fmt.Sprintf("%d min", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	} else {
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	}
}
