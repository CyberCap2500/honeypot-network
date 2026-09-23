-- ©AngelaMos | 2026
-- 013_add_telnet_service_type.sql
-- Add 'telnet' to service_type enum

-- +goose Up
ALTER TYPE service_type ADD VALUE IF NOT EXISTS 'telnet';

-- +goose Down
-- Cannot remove enum values in PostgreSQL, would require recreating the enum
-- and all dependent tables. Leaving telnet in place is safe.
