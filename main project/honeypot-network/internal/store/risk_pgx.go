/*
©AngelaMos | 2026
risk_pgx.go

Risk score repository implementation for PostgreSQL

Implements storage and retrieval of risk scores, attacker risk profiles,
and historical tracking. Part of the NEW risk scoring feature.
*/

package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/CarterPerez-dev/miragenet/pkg/types"
)

// SaveRiskScore stores a computed risk score with full factor breakdown
func (s *PgxStore) SaveRiskScore(
	ctx context.Context,
	attackerIP string,
	score *types.RiskScore,
	profile *types.AttackerRiskProfile,
) error {
	factorsJSON, err := json.Marshal(score.Factors)
	if err != nil {
		return fmt.Errorf("marshal factors: %w", err)
	}

	query := `
		INSERT INTO attacker_risk_scores (
			attacker_ip, score, level, factors, computed_at,
			session_count, failed_auth_attempts, successful_auth_count,
			command_count, unique_commands, mitre_techniques,
			exploit_attempts, suspicious_commands, escalation_speed
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err = s.pool.Exec(ctx, query,
		attackerIP,
		score.Score,
		score.Level,
		factorsJSON,
		score.ComputedAt,
		profile.SessionCount,
		profile.FailedAuthAttempts,
		profile.SuccessfulAuthCount,
		profile.CommandCount,
		profile.UniqueCommands,
		profile.MitreTechniques,
		profile.ExploitAttempts,
		profile.SuspiciousCommands,
		profile.EscalationSpeed,
	)

	if err != nil {
		return fmt.Errorf("insert risk score: %w", err)
	}

	return nil
}

// UpdateAttackerRiskScore updates the current risk score on the attacker record
func (s *PgxStore) UpdateAttackerRiskScore(
	ctx context.Context,
	attackerIP string,
	score int,
	level string,
) error {
	query := `
		UPDATE attackers
		SET risk_score = $1,
		    risk_level = $2,
		    risk_computed_at = $3
		WHERE ip = $4
	`

	_, err := s.pool.Exec(ctx, query, score, level, time.Now().UTC(), attackerIP)
	if err != nil {
		return fmt.Errorf("update attacker risk score: %w", err)
	}

	return nil
}

// GetAttackerRiskProfile builds a risk profile for an attacker from their activity
func (s *PgxStore) GetAttackerRiskProfile(
	ctx context.Context,
	attackerIP string,
) (*types.AttackerRiskProfile, error) {
	profile := &types.AttackerRiskProfile{
		SourceIP: attackerIP,
	}

	// Get attacker basic info
	query := `
		SELECT first_seen, last_seen, total_sessions
		FROM attackers
		WHERE ip = $1
	`
	err := s.pool.QueryRow(ctx, query, attackerIP).Scan(
		&profile.FirstSeenAt,
		&profile.LastSeenAt,
		&profile.SessionCount,
	)
	if err != nil {
		return nil, fmt.Errorf("query attacker: %w", err)
	}

	// Get auth attempts
	authQuery := `
		SELECT 
			COUNT(*) FILTER (WHERE event_type = 'login.failed') as failed_count,
			COUNT(*) FILTER (WHERE event_type = 'login.success') as success_count
		FROM events
		WHERE source_ip = $1
	`
	err = s.pool.QueryRow(ctx, authQuery, attackerIP).Scan(
		&profile.FailedAuthAttempts,
		&profile.SuccessfulAuthCount,
	)
	if err != nil {
		return nil, fmt.Errorf("query auth attempts: %w", err)
	}

	// Get command stats
	cmdQuery := `
		SELECT 
			COUNT(*) as total_commands,
			COUNT(DISTINCT service_data->>'command') as unique_commands
		FROM events
		WHERE source_ip = $1
		  AND event_type = 'command.input'
		  AND service_data IS NOT NULL
	`
	err = s.pool.QueryRow(ctx, cmdQuery, attackerIP).Scan(
		&profile.CommandCount,
		&profile.UniqueCommands,
	)
	if err != nil {
		// Commands might not exist for all attackers
		profile.CommandCount = 0
		profile.UniqueCommands = 0
	}

	// Get MITRE techniques (from sessions)
	mitreQuery := `
		SELECT DISTINCT unnest(mitre_techniques) as technique
		FROM sessions
		WHERE source_ip = $1
		  AND mitre_techniques != '{}'
	`
	rows, err := s.pool.Query(ctx, mitreQuery, attackerIP)
	if err == nil {
		defer rows.Close()
		techniques := []string{}
		for rows.Next() {
			var technique string
			if err := rows.Scan(&technique); err == nil {
				techniques = append(techniques, technique)
			}
		}
		profile.MitreTechniques = techniques
	}

	// Calculate suspicious commands (simplified - checking for common attack patterns)
	suspiciousQuery := `
		SELECT COUNT(*)
		FROM events
		WHERE source_ip = $1
		  AND event_type = 'command.input'
		  AND (
		    service_data->>'command' LIKE '%wget%' OR
		    service_data->>'command' LIKE '%curl%' OR
		    service_data->>'command' LIKE '%nc %' OR
		    service_data->>'command' LIKE '%bash%' OR
		    service_data->>'command' LIKE '%python%' OR
		    service_data->>'command' LIKE '%chmod%' OR
		    service_data->>'command' LIKE '%sudo%' OR
		    service_data->>'command' LIKE '%rm -rf%'
		  )
	`
	err = s.pool.QueryRow(ctx, suspiciousQuery, attackerIP).Scan(&profile.SuspiciousCommands)
	if err != nil {
		profile.SuspiciousCommands = 0
	}

	// Exploit attempts (checking for known exploit events)
	exploitQuery := `
		SELECT COUNT(*)
		FROM events
		WHERE source_ip = $1
		  AND event_type = 'exploit.attempt'
	`
	err = s.pool.QueryRow(ctx, exploitQuery, attackerIP).Scan(&profile.ExploitAttempts)
	if err != nil {
		profile.ExploitAttempts = 0
	}

	// Calculate average session length and escalation speed
	sessionStatsQuery := `
		SELECT 
			AVG(EXTRACT(EPOCH FROM (COALESCE(ended_at, NOW()) - started_at))) as avg_duration,
			AVG(command_count / NULLIF(EXTRACT(EPOCH FROM (COALESCE(ended_at, NOW()) - started_at)) / 60.0, 0)) as avg_speed
		FROM sessions
		WHERE source_ip = $1
	`
	var avgDuration, avgSpeed *float64
	err = s.pool.QueryRow(ctx, sessionStatsQuery, attackerIP).Scan(&avgDuration, &avgSpeed)
	if err == nil && avgDuration != nil {
		profile.AverageSessionLength = *avgDuration
		if avgSpeed != nil {
			profile.EscalationSpeed = *avgSpeed
		}
	}

	return profile, nil
}

// GetRiskScoreHistory retrieves historical risk scores for an attacker
func (s *PgxStore) GetRiskScoreHistory(
	ctx context.Context,
	attackerIP string,
	limit int,
) ([]*types.RiskScore, error) {
	query := `
		SELECT score, level, factors, computed_at
		FROM attacker_risk_scores
		WHERE attacker_ip = $1
		ORDER BY computed_at DESC
		LIMIT $2
	`

	rows, err := s.pool.Query(ctx, query, attackerIP, limit)
	if err != nil {
		return nil, fmt.Errorf("query risk history: %w", err)
	}
	defer rows.Close()

	scores := []*types.RiskScore{}
	for rows.Next() {
		var score types.RiskScore
		var factorsJSON []byte

		err := rows.Scan(&score.Score, &score.Level, &factorsJSON, &score.ComputedAt)
		if err != nil {
			return nil, fmt.Errorf("scan risk score: %w", err)
		}

		if err := json.Unmarshal(factorsJSON, &score.Factors); err != nil {
			return nil, fmt.Errorf("unmarshal factors: %w", err)
		}

		scores = append(scores, &score)
	}

	return scores, nil
}

// GetHighRiskAttackers returns attackers with risk scores above a threshold
func (s *PgxStore) GetHighRiskAttackers(
	ctx context.Context,
	minScore int,
	limit, offset int,
) ([]*types.Attacker, error) {
	query := `
		SELECT 
			id, ip, first_seen, last_seen, total_events, total_sessions,
			country_code, country, city, latitude, longitude, asn, org,
			threat_score, tool_family, tags,
			risk_score, risk_level, risk_computed_at
		FROM attackers
		WHERE risk_score >= $1
		ORDER BY risk_score DESC, last_seen DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.pool.Query(ctx, query, minScore, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query high risk attackers: %w", err)
	}
	defer rows.Close()

	attackers := []*types.Attacker{}
	for rows.Next() {
		var a types.Attacker
		var riskScore, riskLevel, riskComputedAt interface{}

		err := rows.Scan(
			&a.ID, &a.IP, &a.FirstSeen, &a.LastSeen,
			&a.TotalEvents, &a.TotalSessions,
			&a.Geo.CountryCode, &a.Geo.Country, &a.Geo.City,
			&a.Geo.Latitude, &a.Geo.Longitude,
			&a.Geo.ASN, &a.Geo.Org,
			&a.ThreatScore, &a.ToolFamily, &a.Tags,
			&riskScore, &riskLevel, &riskComputedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan attacker: %w", err)
		}

		attackers = append(attackers, &a)
	}

	return attackers, nil
}
