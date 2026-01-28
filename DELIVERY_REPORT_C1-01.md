# 任务 C1-01 交付报告：线路分组 LineGroup 数据模型 + Migration

## 任务信息

- 任务编号：C1-01
- 任务名称：线路分组 LineGroup 数据模型 + Migration
- 任务级别：P0

## 任务目标

创建 LineGroup 数据模型，包括数据库 migration、Model 定义和 CNAME prefix 生成器。

## 核心实现

### 1. 数据库 Migration

创建文件：migrations/022_create_line_groups.sql

表结构：
- id: BIGINT 主键自增
- name: VARCHAR(128) 线路分组名称
- description: VARCHAR(255) 描述信息
- domain_id: BIGINT 关联域名 ID
- node_group_id: BIGINT 关联节点组 ID
- cname_prefix: VARCHAR(64) CNAME 前缀（唯一）
- cname: VARCHAR(255) 完整 CNAME（为保持向后兼容）
- status: VARCHAR(32) 状态（active/disabled）
- created_at: DATETIME(3) 创建时间
- updated_at: DATETIME(3) 更新时间

约束：
- 主键：id
- 唯一索引：uk_cname_prefix (cname_prefix)
- 普通索引：idx_domain_id (domain_id)
- 普通索引：idx_node_group_id (node_group_id)
- 外键：fk_line_groups_domain (domain_id -> domains.id)
- 外键：fk_line_groups_node_group (node_group_id -> node_groups.id)

### 2. LineGroup Model

创建文件：internal/model/line_group.go

字段定义：
- ID: int64
- Name: string
- Description: string
- DomainID: int64
- NodeGroupID: int64
- CNAMEPrefix: string
- CNAME: string（为保持向后兼容）
- Status: string
- CreatedAt: time.Time
- UpdatedAt: time.Time

关联关系：
- Domain: 关联到 domains 表
- NodeGroup: 关联到 node_groups 表

### 3. CNAME Prefix 生成器

创建文件：internal/linegroup/prefix.go

函数：GenerateCNAMEPrefix() string

实现逻辑：
- 生成 8 个随机字节
- 转换为 16 位十六进制字符串
- 添加 "lg-" 前缀
- 格式：lg-<16 hex characters>
- 示例：lg-a0b719f2b1d6f6aa

### 4. 单元测试

创建文件：internal/linegroup/prefix_test.go

测试用例：
1. 格式验证
   - 必须以 "lg-" 开头
   - 总长度为 19 个字符
   - 十六进制部分为 16 个字符
   - 所有字符必须是有效的十六进制数字

2. 唯一性检查
   - 生成 100 个前缀
   - 验证没有重复

测试结果：
```
=== RUN   TestGenerateCNAMEPrefix
=== RUN   TestGenerateCNAMEPrefix/format_validation
=== RUN   TestGenerateCNAMEPrefix/uniqueness_check
--- PASS: TestGenerateCNAMEPrefix (0.00s)
    --- PASS: TestGenerateCNAMEPrefix/format_validation (0.00s)
    --- PASS: TestGenerateCNAMEPrefix/uniqueness_check (0.00s)
PASS
ok  	go_cmdb/internal/linegroup	0.002s
```

## 数据库验证

表结构验证：

```sql
SHOW CREATE TABLE line_groups;
```

结果：
```
CREATE TABLE `line_groups` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL,
  `description` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `domain_id` bigint NOT NULL,
  `node_group_id` bigint NOT NULL,
  `cname_prefix` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `cname` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `status` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'active',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cname_prefix` (`cname_prefix`),
  KEY `idx_line_groups_domain_id` (`domain_id`),
  KEY `idx_line_groups_node_group_id` (`node_group_id`),
  CONSTRAINT `fk_line_groups_domain` FOREIGN KEY (`domain_id`) REFERENCES `domains` (`id`),
  CONSTRAINT `fk_line_groups_node_group` FOREIGN KEY (`node_group_id`) REFERENCES `node_groups` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
```

## 交付物

1. 数据库 Migration 文件：
   - migrations/022_create_line_groups.sql

2. Model 定义文件：
   - internal/model/line_group.go

3. CNAME Prefix 生成器：
   - internal/linegroup/prefix.go
   - internal/linegroup/prefix_test.go

4. 交付报告：
   - DELIVERY_REPORT_C1-01.md

5. 代码已推送到 GitHub 仓库：labubu-daydayone/go_cmdb

6. 数据库已部署到测试服务器：20.2.140.226

## 注意事项

1. 为保持向后兼容，保留了 cname 字段
2. status 字段使用 VARCHAR(32) 而非 ENUM，以便灵活扩展
3. CNAME prefix 格式固定为 lg-<16 hex>，与 Node Group 的 ng-<16 hex> 保持一致
4. 单元测试验证了生成器的格式正确性和唯一性

## 回滚策略

如需回滚：

```sql
DROP TABLE IF EXISTS line_groups;
```

注意：回滚前需确保没有其他表依赖此表。
