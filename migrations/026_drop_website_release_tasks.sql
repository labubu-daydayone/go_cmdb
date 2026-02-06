-- Migration: 026_drop_website_release_tasks
-- Purpose: 删除 website_release_tasks 表，统一使用 release_tasks 作为唯一发布体系
-- Date: 2026-02-07

DROP TABLE IF EXISTS `website_release_tasks`;
