CREATE TABLE `dns_provider_accounts` (
	`id` int AUTO_INCREMENT NOT NULL,
	`name` varchar(100) NOT NULL,
	`provider` enum('cloudflare','godaddy','namecheap','aliyun','dnspod') NOT NULL,
	`apiKey` text NOT NULL,
	`apiSecret` text,
	`email` varchar(320),
	`status` enum('active','inactive','error') NOT NULL DEFAULT 'active',
	`lastSyncAt` timestamp,
	`createdBy` int NOT NULL,
	`createdAt` timestamp NOT NULL DEFAULT (now()),
	`updatedAt` timestamp NOT NULL DEFAULT (now()) ON UPDATE CURRENT_TIMESTAMP,
	CONSTRAINT `dns_provider_accounts_id` PRIMARY KEY(`id`)
);
--> statement-breakpoint
CREATE TABLE `domains` (
	`id` int AUTO_INCREMENT NOT NULL,
	`domainName` varchar(255) NOT NULL,
	`registrarAccountId` int,
	`dnsAccountId` int,
	`source` enum('auto_sync','manual') NOT NULL,
	`zoneId` varchar(100),
	`nsRecords` text,
	`nsStatus` enum('pending','active','failed','unknown') NOT NULL DEFAULT 'unknown',
	`lastNsCheckAt` timestamp,
	`expireDate` timestamp,
	`autoRenew` enum('yes','no','unknown') DEFAULT 'unknown',
	`createdBy` int NOT NULL,
	`createdAt` timestamp NOT NULL DEFAULT (now()),
	`updatedAt` timestamp NOT NULL DEFAULT (now()) ON UPDATE CURRENT_TIMESTAMP,
	CONSTRAINT `domains_id` PRIMARY KEY(`id`),
	CONSTRAINT `domains_domainName_unique` UNIQUE(`domainName`)
);
