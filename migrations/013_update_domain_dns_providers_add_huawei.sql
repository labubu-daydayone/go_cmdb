-- Migration: 013_update_domain_dns_providers_add_huawei
-- Purpose: Add 'huawei' to domain_dns_providers.provider enum

ALTER TABLE domain_dns_providers
MODIFY COLUMN provider ENUM('cloudflare','aliyun','tencent','huawei','manual') NOT NULL;
