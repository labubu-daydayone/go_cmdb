-- Migration: Drop cname column from node_groups table
-- Description: Node Groups are pure backend carriers and should not have domain-related fields
-- Only Line Groups should expose CNAME to external systems

-- Drop cname column from node_groups
ALTER TABLE node_groups DROP COLUMN cname;
