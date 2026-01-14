import { int, mysqlEnum, mysqlTable, text, timestamp, varchar, unique } from "drizzle-orm/mysql-core";
import { relations } from "drizzle-orm";

/**
 * Core user table backing auth flow.
 * Extend this file with additional tables as your product grows.
 * Columns use camelCase to match both database fields and generated types.
 */
export const users = mysqlTable("users", {
  /**
   * Surrogate primary key. Auto-incremented numeric value managed by the database.
   * Use this for relations between tables.
   */
  id: int("id").autoincrement().primaryKey(),
  /** Manus OAuth identifier (openId) returned from the OAuth callback. Unique per user. */
  openId: varchar("openId", { length: 64 }).notNull().unique(),
  name: text("name"),
  email: varchar("email", { length: 320 }),
  loginMethod: varchar("loginMethod", { length: 64 }),
  role: mysqlEnum("role", ["user", "admin"]).default("user").notNull(),
  createdAt: timestamp("createdAt").defaultNow().notNull(),
  updatedAt: timestamp("updatedAt").defaultNow().onUpdateNow().notNull(),
  lastSignedIn: timestamp("lastSignedIn").defaultNow().notNull(),
});

export type User = typeof users.$inferSelect;
export type InsertUser = typeof users.$inferInsert;

/**
 * Roles table - defines system roles
 */
export const roles = mysqlTable("roles", {
  id: int("id").autoincrement().primaryKey(),
  name: varchar("name", { length: 100 }).notNull().unique(),
  description: text("description"),
  createdAt: timestamp("createdAt").defaultNow().notNull(),
  updatedAt: timestamp("updatedAt").defaultNow().onUpdateNow().notNull(),
});

export type Role = typeof roles.$inferSelect;
export type InsertRole = typeof roles.$inferInsert;

/**
 * Permissions table - defines system permissions
 */
export const permissions = mysqlTable("permissions", {
  id: int("id").autoincrement().primaryKey(),
  name: varchar("name", { length: 100 }).notNull().unique(),
  description: text("description"),
  action: varchar("action", { length: 50 }).notNull(), // read, write, delete, execute
  resource: varchar("resource", { length: 100 }).notNull(), // *, domain, nginx, script
  createdAt: timestamp("createdAt").defaultNow().notNull(),
  updatedAt: timestamp("updatedAt").defaultNow().onUpdateNow().notNull(),
});

export type Permission = typeof permissions.$inferSelect;
export type InsertPermission = typeof permissions.$inferInsert;

/**
 * Permission groups table - groups resources together for shared access
 */
export const permissionGroups = mysqlTable("permission_groups", {
  id: int("id").autoincrement().primaryKey(),
  name: varchar("name", { length: 100 }).notNull().unique(),
  description: text("description"),
  createdAt: timestamp("createdAt").defaultNow().notNull(),
  updatedAt: timestamp("updatedAt").defaultNow().onUpdateNow().notNull(),
});

export type PermissionGroup = typeof permissionGroups.$inferSelect;
export type InsertPermissionGroup = typeof permissionGroups.$inferInsert;

/**
 * User-Role association table
 */
export const userRoles = mysqlTable("user_roles", {
  id: int("id").autoincrement().primaryKey(),
  userId: int("userId").notNull(),
  roleId: int("roleId").notNull(),
  createdAt: timestamp("createdAt").defaultNow().notNull(),
}, (table) => ({
  uniqUserRole: unique("ur_uid_rid").on(table.userId, table.roleId),
}));

export type UserRole = typeof userRoles.$inferSelect;
export type InsertUserRole = typeof userRoles.$inferInsert;

/**
 * Role-Permission association table
 */
export const rolePermissions = mysqlTable("role_permissions", {
  id: int("id").autoincrement().primaryKey(),
  roleId: int("roleId").notNull(),
  permissionId: int("permissionId").notNull(),
  createdAt: timestamp("createdAt").defaultNow().notNull(),
}, (table) => ({
  uniqRolePerm: unique("rp_rid_pid").on(table.roleId, table.permissionId),
}));

export type RolePermission = typeof rolePermissions.$inferSelect;
export type InsertRolePermission = typeof rolePermissions.$inferInsert;

/**
 * User-PermissionGroup association table
 */
export const userPermissionGroups = mysqlTable("user_permission_groups", {
  id: int("id").autoincrement().primaryKey(),
  userId: int("userId").notNull(),
  groupId: int("groupId").notNull(),
  createdAt: timestamp("createdAt").defaultNow().notNull(),
}, (table) => ({
  uniqUserGrp: unique("upg_uid_gid").on(table.userId, table.groupId),
}));

export type UserPermissionGroup = typeof userPermissionGroups.$inferSelect;
export type InsertUserPermissionGroup = typeof userPermissionGroups.$inferInsert;

/**
 * Permission Group Resources table - tracks which resources belong to which groups
 */
export const permissionGroupResources = mysqlTable("permission_group_resources", {
  id: int("id").autoincrement().primaryKey(),
  groupId: int("groupId").notNull(),
  resourceType: varchar("resourceType", { length: 50 }).notNull(), // domain, nginx, script
  resourceId: int("resourceId").notNull(), // ID of the actual resource
  createdAt: timestamp("createdAt").defaultNow().notNull(),
}, (table) => ({
  uniqGrpRes: unique("pgr_gid_rtype_rid").on(table.groupId, table.resourceType, table.resourceId),
}));

export type PermissionGroupResource = typeof permissionGroupResources.$inferSelect;
export type InsertPermissionGroupResource = typeof permissionGroupResources.$inferInsert;

/**
 * Resource ownership table - tracks who created/owns each resource
 */
export const resourceOwnership = mysqlTable("resource_ownership", {
  id: int("id").autoincrement().primaryKey(),
  userId: int("userId").notNull(),
  resourceType: varchar("resourceType", { length: 50 }).notNull(), // domain, nginx, script
  resourceId: int("resourceId").notNull(), // ID of the actual resource
  createdAt: timestamp("createdAt").defaultNow().notNull(),
}, (table) => ({
  uniqOwner: unique("ro_uid_rtype_rid").on(table.userId, table.resourceType, table.resourceId),
}));

export type ResourceOwnership = typeof resourceOwnership.$inferSelect;
export type InsertResourceOwnership = typeof resourceOwnership.$inferInsert;

// Relations
export const usersRelations = relations(users, ({ many }) => ({
  userRoles: many(userRoles),
  userPermissionGroups: many(userPermissionGroups),
  ownedResources: many(resourceOwnership),
}));

export const rolesRelations = relations(roles, ({ many }) => ({
  userRoles: many(userRoles),
  rolePermissions: many(rolePermissions),
}));

export const permissionsRelations = relations(permissions, ({ many }) => ({
  rolePermissions: many(rolePermissions),
}));

export const permissionGroupsRelations = relations(permissionGroups, ({ many }) => ({
  userPermissionGroups: many(userPermissionGroups),
  groupResources: many(permissionGroupResources),
}));

export const userRolesRelations = relations(userRoles, ({ one }) => ({
  user: one(users, {
    fields: [userRoles.userId],
    references: [users.id],
  }),
  role: one(roles, {
    fields: [userRoles.roleId],
    references: [roles.id],
  }),
}));

export const rolePermissionsRelations = relations(rolePermissions, ({ one }) => ({
  role: one(roles, {
    fields: [rolePermissions.roleId],
    references: [roles.id],
  }),
  permission: one(permissions, {
    fields: [rolePermissions.permissionId],
    references: [permissions.id],
  }),
}));

export const userPermissionGroupsRelations = relations(userPermissionGroups, ({ one }) => ({
  user: one(users, {
    fields: [userPermissionGroups.userId],
    references: [users.id],
  }),
  group: one(permissionGroups, {
    fields: [userPermissionGroups.groupId],
    references: [permissionGroups.id],
  }),
}));

export const permissionGroupResourcesRelations = relations(permissionGroupResources, ({ one }) => ({
  group: one(permissionGroups, {
    fields: [permissionGroupResources.groupId],
    references: [permissionGroups.id],
  }),
}));

export const resourceOwnershipRelations = relations(resourceOwnership, ({ one }) => ({
  user: one(users, {
    fields: [resourceOwnership.userId],
    references: [users.id],
  }),
}));

/**
 * DNS Provider Accounts table - stores API credentials for DNS providers
 */
export const dnsProviderAccounts = mysqlTable("dns_provider_accounts", {
  id: int("id").autoincrement().primaryKey(),
  name: varchar("name", { length: 100 }).notNull(), // Account nickname
  provider: mysqlEnum("provider", ["cloudflare", "godaddy", "namecheap", "aliyun", "dnspod"]).notNull(),
  apiKey: text("apiKey").notNull(), // Encrypted API key
  apiSecret: text("apiSecret"), // Encrypted API secret (if needed)
  email: varchar("email", { length: 320 }), // Account email (for Cloudflare)
  status: mysqlEnum("status", ["active", "inactive", "error"]).default("active").notNull(),
  lastSyncAt: timestamp("lastSyncAt"),
  createdBy: int("createdBy").notNull(),
  createdAt: timestamp("createdAt").defaultNow().notNull(),
  updatedAt: timestamp("updatedAt").defaultNow().onUpdateNow().notNull(),
});

export type DnsProviderAccount = typeof dnsProviderAccounts.$inferSelect;
export type InsertDnsProviderAccount = typeof dnsProviderAccounts.$inferInsert;

/**
 * Domains table - stores domain assets
 */
export const domains = mysqlTable("domains", {
  id: int("id").autoincrement().primaryKey(),
  domainName: varchar("domainName", { length: 255 }).notNull().unique(),
  registrarAccountId: int("registrarAccountId"), // Reference to registrar account (if auto-synced)
  dnsAccountId: int("dnsAccountId"), // Reference to DNS provider account
  source: mysqlEnum("source", ["auto_sync", "manual"]).notNull(), // How domain was added
  zoneId: varchar("zoneId", { length: 100 }), // External zone ID (e.g., Cloudflare zone ID)
  nsRecords: text("nsRecords"), // JSON array of NS records
  nsStatus: mysqlEnum("nsStatus", ["pending", "active", "failed", "unknown"]).default("unknown").notNull(),
  lastNsCheckAt: timestamp("lastNsCheckAt"),
  expireDate: timestamp("expireDate"),
  autoRenew: mysqlEnum("autoRenew", ["yes", "no", "unknown"]).default("unknown"),
  createdBy: int("createdBy").notNull(),
  createdAt: timestamp("createdAt").defaultNow().notNull(),
  updatedAt: timestamp("updatedAt").defaultNow().onUpdateNow().notNull(),
});

export type Domain = typeof domains.$inferSelect;
export type InsertDomain = typeof domains.$inferInsert;

// Relations for DNS provider accounts
export const dnsProviderAccountsRelations = relations(dnsProviderAccounts, ({ one, many }) => ({
  creator: one(users, {
    fields: [dnsProviderAccounts.createdBy],
    references: [users.id],
  }),
  domains: many(domains),
}));

// Relations for domains
export const domainsRelations = relations(domains, ({ one }) => ({
  dnsAccount: one(dnsProviderAccounts, {
    fields: [domains.dnsAccountId],
    references: [dnsProviderAccounts.id],
  }),
  registrarAccount: one(dnsProviderAccounts, {
    fields: [domains.registrarAccountId],
    references: [dnsProviderAccounts.id],
  }),
  creator: one(users, {
    fields: [domains.createdBy],
    references: [users.id],
  }),
}));
