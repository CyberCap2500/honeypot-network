// ===================
// ©AngelaMos | 2026
// attacker.types.ts
// ===================

import { z } from 'zod'
import { geoInfoSchema } from './common.types'

export const attackerSchema = z.object({
  id: z.number(),
  ip: z.string(),
  first_seen: z.string(),
  last_seen: z.string(),
  total_events: z.number(),
  total_sessions: z.number(),
  geo: geoInfoSchema,
  threat_score: z.number(),
  tool_family: z.string().optional(),
  tags: z.array(z.string()).optional(),
  risk_score: z.number().optional(),
  risk_level: z.enum(['low', 'medium', 'high', 'critical']).optional(),
  risk_computed_at: z.string().optional(),
})

export type Attacker = z.infer<typeof attackerSchema>

export const isValidAttacker = (data: unknown): data is Attacker => {
  return attackerSchema.safeParse(data).success
}

// ===================
// Risk Score Types (NEW - Phase 1)
// ===================

export const riskFactorSchema = z.object({
  name: z.string(),
  value: z.number(),
  weight: z.number(),
  score: z.number(),
  description: z.string(),
})

export type RiskFactor = z.infer<typeof riskFactorSchema>

export const riskScoreSchema = z.object({
  score: z.number(),
  level: z.enum(['low', 'medium', 'high', 'critical']),
  factors: z.array(riskFactorSchema),
  computed_at: z.string(),
})

export type RiskScore = z.infer<typeof riskScoreSchema>

export const attackerRiskProfileSchema = z.object({
  source_ip: z.string(),
  session_count: z.number(),
  failed_auth_attempts: z.number(),
  successful_auth_count: z.number(),
  command_count: z.number(),
  unique_commands: z.number(),
  mitre_techniques: z.array(z.string()),
  exploit_attempts: z.number(),
  suspicious_commands: z.number(),
  first_seen_at: z.string(),
  last_seen_at: z.string(),
  average_session_length: z.number(),
  escalation_speed: z.number(),
})

export type AttackerRiskProfile = z.infer<typeof attackerRiskProfileSchema>
