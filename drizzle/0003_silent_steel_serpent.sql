ALTER TABLE `permission_group_resources` DROP INDEX `permission_group_resources_groupId_resourceType_resourceId_unique`;--> statement-breakpoint
ALTER TABLE `resource_ownership` DROP INDEX `resource_ownership_userId_resourceType_resourceId_unique`;--> statement-breakpoint
ALTER TABLE `role_permissions` DROP INDEX `role_permissions_roleId_permissionId_unique`;--> statement-breakpoint
ALTER TABLE `user_permission_groups` DROP INDEX `user_permission_groups_userId_groupId_unique`;--> statement-breakpoint
ALTER TABLE `user_roles` DROP INDEX `user_roles_userId_roleId_unique`;--> statement-breakpoint
ALTER TABLE `permission_group_resources` ADD CONSTRAINT `pgr_gid_rtype_rid` UNIQUE(`groupId`,`resourceType`,`resourceId`);--> statement-breakpoint
ALTER TABLE `resource_ownership` ADD CONSTRAINT `ro_uid_rtype_rid` UNIQUE(`userId`,`resourceType`,`resourceId`);--> statement-breakpoint
ALTER TABLE `role_permissions` ADD CONSTRAINT `rp_rid_pid` UNIQUE(`roleId`,`permissionId`);--> statement-breakpoint
ALTER TABLE `user_permission_groups` ADD CONSTRAINT `upg_uid_gid` UNIQUE(`userId`,`groupId`);--> statement-breakpoint
ALTER TABLE `user_roles` ADD CONSTRAINT `ur_uid_rid` UNIQUE(`userId`,`roleId`);