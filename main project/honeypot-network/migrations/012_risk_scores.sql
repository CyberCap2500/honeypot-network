-- ©AngelaMos | 2026
-- 012_risk_scores.sql
-- Risk scoring system for attacker behavior analysis

-- +goose Up

-- Add risk_score column to attackers table (current/latest score)
ALTER TABLE attackers ADD COLUMN risk_score INTEGER DEFAULT 0;
ALTER TABLE attackers ADD COLUMN risk_level TEXT DEFAULT 'low';
ALTER TABLE attackers ADD COLUMN risk_computed_at TIMESTAMPTZ;

-- Add risk_score column to sessions table (score at session time)
ALTER TABLE sessions ADD COLUMN risk_score INTEGER;

-- Create attacker_risk_scores table for historical risk score tracking
CREATE TABLE attacker_risk_scores (
    id                BIGSERIAL PRIMARY KEY,
    attacker_ip       TEXT NOT NULL,
    score             INTEGER NOT NULL CHECK (score >= 0 AND score <= 100),
    level             TEXT NOT NULL CHECK (level IN ('low', 'medium', 'high', 'critical')),
    factors           JSONB NOT NULL,  -- Array of RiskFactor objects
    computed_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Profile snapshot at computation time
    session_count           INTEGER NOT NULL DEFAULT 0,
    failed_auth_attempts    INTEGER NOT NULL DEFAULT 0,
    successful_auth_count   INTEGER NOT NULL DEFAULT 0,
    command_count           INTEGER NOT NULL DEFAULT 0,
    unique_commands         INTEGER NOT NULL DEFAULT 0,
    mitre_techniques        TEXT[] NOT NULL DEFAULT '{}',
    exploit_attempts        INTEGER NOT NULL DEFAULT 0,
    suspicious_commands     INTEGER NOT NULL DEFAULT 0,
    escalation_speed        DOUBLE PRECISION NOT NULL DEFAULT 0,
    
    FOREIGN KEY (attacker_ip) REFERENCES attackers(ip) ON DELETE CASCADE
);

-- Indexes for risk score queries
CREATE INDEX idx_attacker_risk_scores_ip ON attacker_risk_scores (attacker_ip);
CREATE INDEX idx_attacker_risk_scores_computed ON attacker_risk_scores (computed_at DESC);
CREATE INDEX idx_attacker_risk_scores_score ON attacker_risk_scores (score DESC);
CREATE INDEX idx_attacker_risk_scores_level ON attacker_risk_scores (level);

-- Index on attackers.risk_score for filtering/sorting
CREATE INDEX idx_attackers_risk_score ON attackers (risk_score DESC);
CREATE INDEX idx_attackers_risk_level ON attackers (risk_level);

-- +goose Down

-- Drop indexes
DROP INDEX IF EXISTS idx_attackers_risk_level;
DROP INDEX IF EXISTS idx_attackers_risk_score;
DROP INDEX IF EXISTS idx_attacker_risk_scores_level;
DROP INDEX IF EXISTS idx_attacker_risk_scores_score;
DROP INDEX IF EXISTS idx_attacker_risk_scores_computed;
DROP INDEX IF EXISTS idx_attacker_risk_scores_ip;

-- Drop table
DROP TABLE IF EXISTS attacker_risk_scores;

-- Remove columns from sessions
ALTER TABLE sessions DROP COLUMN IF EXISTS risk_score;

-- Remove columns from attackers
ALTER TABLE attackers DROP COLUMN IF EXISTS risk_computed_at;
ALTER TABLE attackers DROP COLUMN IF EXISTS risk_level;
ALTER TABLE attackers DROP COLUMN IF EXISTS risk_score;
