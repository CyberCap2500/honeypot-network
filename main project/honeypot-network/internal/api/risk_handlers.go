/*
©AngelaMos | 2026
risk_handlers.go

Risk score API endpoints

Provides REST endpoints for querying attacker risk scores, risk history,
and high-risk attacker lists. Part of the NEW risk scoring feature.
*/

package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// handleAttackerRisk returns the current risk score and breakdown for an attacker
// GET /api/attackers/{ip}/risk
func (s *Server) handleAttackerRisk(w http.ResponseWriter, r *http.Request) {
	attackerIP := chi.URLParam(r, "ip")
	if attackerIP == "" {
		s.writeError(w, http.StatusBadRequest, "missing attacker IP")
		return
	}

	// Get attacker profile
	profile, err := s.store.GetAttackerRiskProfile(r.Context(), attackerIP)
	if err != nil {
		s.logger.Error().Err(err).Str("ip", attackerIP).Msg("failed to get risk profile")
		s.writeError(w, http.StatusInternalServerError, "failed to retrieve risk profile")
		return
	}

	// Get latest risk score from history
	history, err := s.store.GetRiskScoreHistory(r.Context(), attackerIP, 1)
	if err != nil {
		s.logger.Error().Err(err).Str("ip", attackerIP).Msg("failed to get risk score")
		s.writeError(w, http.StatusInternalServerError, "failed to retrieve risk score")
		return
	}

	var latestScore interface{}
	if len(history) > 0 {
		latestScore = history[0]
	}

	response := map[string]interface{}{
		"attacker_ip": attackerIP,
		"profile":     profile,
		"risk_score":  latestScore,
	}

	s.writeJSON(w, http.StatusOK, apiResponse{Data: response})
}

// handleAttackerRiskHistory returns historical risk scores for an attacker
// GET /api/attackers/{ip}/risk/history?limit=10
func (s *Server) handleAttackerRiskHistory(w http.ResponseWriter, r *http.Request) {
	attackerIP := chi.URLParam(r, "ip")
	if attackerIP == "" {
		s.writeError(w, http.StatusBadRequest, "missing attacker IP")
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
			if limit > 100 {
				limit = 100
			}
		}
	}

	history, err := s.store.GetRiskScoreHistory(r.Context(), attackerIP, limit)
	if err != nil {
		s.logger.Error().Err(err).Str("ip", attackerIP).Msg("failed to get risk history")
		s.writeError(w, http.StatusInternalServerError, "failed to retrieve risk history")
		return
	}

	response := map[string]interface{}{
		"attacker_ip": attackerIP,
		"history":     history,
		"count":       len(history),
	}

	s.writeJSON(w, http.StatusOK, apiResponse{Data: response})
}

// handleHighRiskAttackers returns attackers with risk scores above a threshold
// GET /api/risk/high?min_score=60&limit=20&offset=0
func (s *Server) handleHighRiskAttackers(w http.ResponseWriter, r *http.Request) {
	minScore := 60 // default: high risk (>=60)
	if scoreStr := r.URL.Query().Get("min_score"); scoreStr != "" {
		if parsed, err := strconv.Atoi(scoreStr); err == nil && parsed >= 0 && parsed <= 100 {
			minScore = parsed
		}
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
			if limit > 100 {
				limit = 100
			}
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	attackers, err := s.store.GetHighRiskAttackers(r.Context(), minScore, limit, offset)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get high risk attackers")
		s.writeError(w, http.StatusInternalServerError, "failed to retrieve high risk attackers")
		return
	}

	response := map[string]interface{}{
		"min_score": minScore,
		"attackers": attackers,
		"count":     len(attackers),
		"limit":     limit,
		"offset":    offset,
	}

	s.writeJSON(w, http.StatusOK, apiResponse{Data: response})
}

// handleRiskStats returns aggregate statistics about risk scores
// GET /api/risk/stats
func (s *Server) handleRiskStats(w http.ResponseWriter, r *http.Request) {
	// For now, return basic stats from high risk attackers query
	// Can be enhanced with dedicated stats query later

	attackers, err := s.store.GetHighRiskAttackers(r.Context(), 0, 1000, 0)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get attackers for stats")
		s.writeError(w, http.StatusInternalServerError, "failed to retrieve risk statistics")
		return
	}

	// Calculate stats from attackers
	levelCounts := map[string]int{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
	}

	totalScore := 0
	highestIP := ""
	highestScore := 0

	for _, attacker := range attackers {
		// Get risk level from attacker (would need to add to Attacker type)
		// For now, infer from score
		level := "low"
		// This is a placeholder - actual level should come from attacker.RiskLevel
		if attacker.ThreatScore >= 80 {
			level = "critical"
		} else if attacker.ThreatScore >= 60 {
			level = "high"
		} else if attacker.ThreatScore >= 40 {
			level = "medium"
		}
		levelCounts[level]++

		totalScore += attacker.ThreatScore
		if attacker.ThreatScore > highestScore {
			highestScore = attacker.ThreatScore
			highestIP = attacker.IP
		}
	}

	avgScore := 0.0
	if len(attackers) > 0 {
		avgScore = float64(totalScore) / float64(len(attackers))
	}

	response := map[string]interface{}{
		"by_level":      levelCounts,
		"average_score": avgScore,
		"total_count":   len(attackers),
		"highest_risk": map[string]interface{}{
			"ip":    highestIP,
			"score": highestScore,
		},
	}

	s.writeJSON(w, http.StatusOK, apiResponse{Data: response})
}
