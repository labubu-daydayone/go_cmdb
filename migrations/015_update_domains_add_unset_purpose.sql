-- Migration: 015_update_domains_add_unset_purpose
-- Purpose: Add 'unset' to domains.purpose enum and change default to 'unset'

ALTER TABLE domains
MODIFY COLUMN purpose ENUM('unset','cdn','general') NOT NULL DEFAULT 'unset';
