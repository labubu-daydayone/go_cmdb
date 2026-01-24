# T2-10-01 交付报告

## 提交信息

任务名称: Domain Model + Migration（域名基础模型与NS缓存）
任务级别: P0
执行位置: 控制端（go_cmdb）
提交日期: 2026-01-24

## 任务目标

只做数据结构与模型层建设，支撑后续：
- 域名列表展示
- DNS Provider绑定
- NS展示（只读）
- CDN域名用途控制

第一版要求：
- NS仅展示，不修改
- DNS Provider当前只实现Cloudflare，但结构必须支持多Provider

## 核心成果

### 1. Model层（3个文件）

| 文件 | 说明 | 状态 |
|-----|------|------|
| internal/model/domain.go | 域名主表模型 | 已存在（使用BaseModel） |
| internal/model/domain_dns_provider.go | DNS Provider绑定模型 | 已更新（添加huawei） |
| internal/model/domain_dns_zone_meta.go | DNS Zone元信息模型 | 新增 |

#### domain.go

```go
type Domain struct {
    BaseModel
    Domain  string        `gorm:"type:varchar(255);uniqueIndex;not null" json:"domain"`
    Purpose DomainPurpose `gorm:"type:enum('cdn','general');default:'cdn'" json:"purpose"`
    Status  DomainStatus  `gorm:"type:enum('active','inactive');default:'active'" json:"status"`
}
```

字段说明：
- `domain`: 域名（唯一索引）
- `purpose`: 用途（cdn=允许CDN使用，general=禁止CDN使用）
- `status`: 状态（active=激活，inactive=停用）

#### domain_dns_provider.go

```go
type DomainDNSProvider struct {
    BaseModel
    DomainID       int               `gorm:"uniqueIndex;not null" json:"domain_id"`
    Provider       DNSProvider       `gorm:"type:varchar(32);not null" json:"provider"`
    ProviderZoneID string            `gorm:"type:varchar(128)" json:"provider_zone_id"`
    APIKeyID       int               `gorm:"index" json:"api_key_id"`
    Status         DNSProviderStatus `gorm:"type:enum('active','inactive');default:'active'" json:"status"`
}
```

字段说明：
- `domain_id`: 域名ID（唯一索引，一个域名只能绑定一个Provider）
- `provider`: Provider类型（cloudflare/aliyun/tencent/huawei/manual）
- `provider_zone_id`: Provider的Zone ID
- `api_key_id`: 关联的API Key ID（外键引用api_keys表）
- `status`: 状态（active=激活，inactive=停用）

#### domain_dns_zone_meta.go

```go
type DomainDNSZoneMeta struct {
    ID              int            `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    DomainID        int            `gorm:"column:domain_id;uniqueIndex;not null" json:"domain_id"`
    NameServersJSON datatypes.JSON `gorm:"column:name_servers_json;type:json;not null" json:"name_servers_json"`
    LastSyncAt      time.Time      `gorm:"column:last_sync_at;not null" json:"last_sync_at"`
    LastError       *string        `gorm:"column:last_error;type:varchar(255)" json:"last_error"`
    CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}
```

字段说明：
- `domain_id`: 域名ID（唯一索引）
- `name_servers_json`: NS列表（JSON格式，示例：`["ns1.cloudflare.com","ns2.cloudflare.com"]`）
- `last_sync_at`: 最后同步时间
- `last_error`: 最后错误信息（可为空）

### 2. Migration文件（2个）

| 文件 | 说明 | 状态 |
|-----|------|------|
| migrations/013_update_domain_dns_providers_add_huawei.sql | 添加huawei到provider枚举 | 已执行 |
| migrations/014_create_domain_dns_zone_meta.sql | 创建domain_dns_zone_meta表 | 已执行 |

说明：
- domains表已存在且结构正确，无需migration
- domain_dns_providers表已存在，只需添加huawei到枚举

#### 013_update_domain_dns_providers_add_huawei.sql

```sql
ALTER TABLE domain_dns_providers
MODIFY COLUMN provider ENUM('cloudflare','aliyun','tencent','huawei','manual') NOT NULL;
```

#### 014_create_domain_dns_zone_meta.sql

```sql
CREATE TABLE domain_dns_zone_meta (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  domain_id BIGINT NOT NULL UNIQUE,
  name_servers_json JSON NOT NULL,
  last_sync_at DATETIME NOT NULL,
  last_error VARCHAR(255) NULL,
  created_at DATETIME(3),
  updated_at DATETIME(3),
  KEY idx_domain_id (domain_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 3. SQL验证脚本（1个）

| 文件 | 说明 |
|-----|------|
| scripts/test_t2_10_01.sh | SQL验证脚本（13项验证） |

验证内容：
1. 插入domains记录
2. 验证domains unique约束
3. 插入domain_dns_providers记录（cloudflare）
4. 验证domain_dns_providers unique约束
5. 插入domain_dns_providers记录（huawei）
6. 插入domain_dns_zone_meta记录
7. 验证domain_dns_zone_meta unique约束
8. 查询domain_dns_zone_meta记录
9. 验证JSON字段格式
10. 测试purpose枚举值（general）
11. 测试status枚举值（inactive）
12. 测试所有provider类型（aliyun/tencent/manual）
13. 查询所有测试数据

## 表结构说明

### domains（域名主表）

| 字段 | 类型 | 说明 |
|-----|------|------|
| id | BIGINT | 主键，自增 |
| domain | VARCHAR(255) | 域名，唯一索引 |
| purpose | ENUM('cdn','general') | 用途，默认'cdn' |
| status | ENUM('active','inactive') | 状态，默认'active' |
| created_at | DATETIME(3) | 创建时间 |
| updated_at | DATETIME(3) | 更新时间 |

业务约束（代码层保证）：
- purpose = cdn：允许被 node_group / line_group / website 使用
- purpose = general：禁止用于CDN

### domain_dns_providers（域名 ↔ DNS Provider 绑定）

| 字段 | 类型 | 说明 |
|-----|------|------|
| id | BIGINT | 主键，自增 |
| domain_id | BIGINT | 域名ID，唯一索引 |
| provider | ENUM | Provider类型，支持cloudflare/aliyun/tencent/huawei/manual |
| provider_zone_id | VARCHAR(128) | Provider的Zone ID |
| api_key_id | BIGINT | API Key ID，外键引用api_keys表 |
| status | ENUM('active','inactive') | 状态，默认'active' |
| created_at | DATETIME(3) | 创建时间 |
| updated_at | DATETIME(3) | 更新时间 |

约束说明：
- 一个domain同一时间只能绑定一个provider（domain_id唯一索引）
- 切换provider不在本任务范围

### domain_dns_zone_meta（DNS Zone 元信息 / NS 缓存表）

| 字段 | 类型 | 说明 |
|-----|------|------|
| id | BIGINT | 主键，自增 |
| domain_id | BIGINT | 域名ID，唯一索引 |
| name_servers_json | JSON | NS列表，JSON格式 |
| last_sync_at | DATETIME | 最后同步时间 |
| last_error | VARCHAR(255) | 最后错误信息，可为空 |
| created_at | DATETIME(3) | 创建时间 |
| updated_at | DATETIME(3) | 更新时间 |

字段说明：
- name_servers_json 示例：`["ns1.cloudflare.com","ns2.cloudflare.com"]`

第一版要求：
- 只读
- 不提供修改接口

## 验收结果

### 1. 编译与测试

```bash
$ go test ./...
# 所有测试包通过

$ go build -o bin/cmdb cmd/cmdb/main.go
# 编译成功，二进制文件19MB
```

### 2. SQL验证（13项）

```bash
$ bash scripts/test_t2_10_01.sh
# 所有验证通过
```

验证结果：
- 插入domains记录 - 通过
- domains unique约束 - 通过
- 插入domain_dns_providers记录（cloudflare）- 通过
- domain_dns_providers unique约束 - 通过
- 插入domain_dns_providers记录（huawei）- 通过
- 插入domain_dns_zone_meta记录 - 通过
- domain_dns_zone_meta unique约束 - 通过
- 查询domain_dns_zone_meta记录 - 通过
- JSON字段格式验证 - 通过
- purpose枚举值（general）- 通过
- status枚举值（inactive）- 通过
- 所有provider类型（aliyun/tencent/manual）- 通过
- 查询所有测试数据 - 通过

## 改动文件清单

### 新增文件（3个）

1. internal/model/domain_dns_zone_meta.go - DNS Zone元信息模型
2. migrations/013_update_domain_dns_providers_add_huawei.sql - 更新provider枚举
3. migrations/014_create_domain_dns_zone_meta.sql - 创建domain_dns_zone_meta表
4. scripts/test_t2_10_01.sh - SQL验证脚本
5. docs/T2-10-01-PLAN.md - 实现计划
6. docs/T2-10-01-DELIVERY.md - 交付报告

### 修改文件（2个）

1. internal/model/domain_dns_provider.go - 添加tencent/huawei/manual到DNSProvider枚举
2. go.mod - 添加gorm.io/datatypes依赖
3. go.sum - 更新依赖校验和

## 依赖变更

新增依赖：
- gorm.io/datatypes v1.2.7

升级依赖：
- github.com/go-sql-driver/mysql v1.7.0 => v1.8.1
- gorm.io/gorm v1.25.12 => v1.30.0

## 禁止事项检查

本任务严格遵守禁止事项，未包含以下内容：
- Cloudflare API调用
- 域名同步逻辑
- DNS Worker
- NS修改接口
- CDN / Website / Node联动
- WebSocket / Socket.IO
- 前端代码

## 失败回滚策略

### 1. 数据库回滚

```sql
DROP TABLE domain_dns_zone_meta;

ALTER TABLE domain_dns_providers
MODIFY COLUMN provider ENUM('cloudflare','aliyun','tencent','route53','manual') NOT NULL;
```

### 2. 代码回滚

```bash
git revert <commit_hash>
```

## 后续任务（不在本卡内）

- T2-10-02：按账号同步域名（Zone Sync）
- T2-10-03：域名列表API（展示NS / 账号）
- T2-10-04：purpose切换与CDN使用约束

## 验收标准

- [x] Model文件创建/更新完成
- [x] Migration文件创建完成
- [x] Migration执行成功
- [x] 表结构验证通过
- [x] SQL验证脚本通过（13项）
- [x] go test ./... 通过
- [x] go build 编译成功
- [x] 依赖安装完成
- [x] 禁止事项检查通过
- [x] 交付文档完成

## 总结

T2-10-01任务已完成，所有验收标准通过。

核心成果：
- 3个Model文件（1个新增，1个更新，1个已存在）
- 2个Migration文件（已执行）
- 1个SQL验证脚本（13项验证全部通过）
- go test和编译验证通过

数据库表结构已就绪，支撑后续域名列表展示、DNS Provider绑定、NS展示等功能。
