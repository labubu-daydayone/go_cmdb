# 任务 C0-08-03 交付报告

## 任务概述

任务编号：C0-08-03

任务名称：Node Group 去 domainId 设计债清理

任务类型：P0 架构债修复

目标：将 Node Group 完全改成只描述 IP 组，与域名的关系只能通过 domain_dns_records 表体现，移除所有 domainId 相关代码和数据库字段。

## 改动文件清单

### 数据库迁移文件

1. migrations/021_drop_node_groups_domain_id.sql

### 后端代码文件

1. internal/model/node_group.go
2. api/v1/node_groups/handler.go

## 实现内容

### 1. 数据库变更

执行迁移文件 021_drop_node_groups_domain_id.sql，移除 node_groups 表的 domain_id 字段。

迁移内容：

```sql
-- Drop foreign key constraint first
ALTER TABLE `node_groups` DROP FOREIGN KEY `fk_node_groups_domain`;

-- Drop index on domain_id
ALTER TABLE `node_groups` DROP INDEX `idx_node_groups_domain_id`;

-- Drop domain_id column
ALTER TABLE `node_groups` DROP COLUMN `domain_id`;
```

迁移后表结构：

```sql
CREATE TABLE `node_groups` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(128) NOT NULL,
  `description` varchar(255) DEFAULT NULL,
  `cname_prefix` varchar(128) NOT NULL,
  `cname` varchar(255) NOT NULL,
  `status` enum('active','inactive') NOT NULL DEFAULT 'active',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_node_groups_name` (`name`),
  UNIQUE KEY `idx_node_groups_cname_prefix` (`cname_prefix`),
  UNIQUE KEY `idx_node_groups_cname` (`cname`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
```

### 2. Model 修改

文件：internal/model/node_group.go

移除 DomainID 字段和 Domain 关联：

```go
type NodeGroup struct {
    BaseModel
    Name        string          `gorm:"type:varchar(128);uniqueIndex;not null" json:"name"`
    Description string          `gorm:"type:varchar(255)" json:"description"`
    CNAMEPrefix string          `gorm:"type:varchar(128);uniqueIndex;not null" json:"cnamePrefix"`
    CNAME       string          `gorm:"type:varchar(255);uniqueIndex;not null" json:"cname"`
    Status      NodeGroupStatus `gorm:"type:enum('active','inactive');default:'active'" json:"status"`
    
    // Relations
    IPs []NodeGroupIP `gorm:"foreignKey:NodeGroupID;constraint:OnDelete:CASCADE" json:"ips,omitempty"`
}
```

### 3. Handler 修改

文件：api/v1/node_groups/handler.go

#### 3.1 移除 ListRequest 中的 DomainID 过滤

```go
type ListRequest struct {
    Page     int    `form:"page"`
    PageSize int    `form:"pageSize"`
    Name     string `form:"name"`
    Status   string `form:"status"`
}
```

#### 3.2 修改 ListResponse 返回结构

使用自定义 NodeGroupItem 结构，确保所有字段使用 lowerCamelCase：

```go
type NodeGroupItem struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    CNAMEPrefix string `json:"cnamePrefix"`
    CNAME       string `json:"cname"`
    Status      string `json:"status"`
    IPCount     int    `json:"ipCount"`
    CreatedAt   string `json:"createdAt"`
    UpdatedAt   string `json:"updatedAt"`
}
```

#### 3.3 Create 方法修改

不再接收 domainId 参数，自动检测所有 CDN 域名并为每个域名创建 DNS 记录：

```go
func (h *Handler) Create(c *gin.Context) {
    var req CreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        httpx.FailErr(c, httpx.ErrParamMissing(err.Error()))
        return
    }

    // Fetch all CDN domains (purpose=cdn, status=active)
    var cdnDomains []model.Domain
    if err := h.db.Where("purpose = ? AND status = ?", "cdn", "active").Find(&cdnDomains).Error; err != nil {
        httpx.FailErr(c, httpx.ErrDatabaseError("failed to fetch CDN domains", err))
        return
    }

    if len(cdnDomains) == 0 {
        httpx.FailErr(c, httpx.ErrNotFound("no active CDN domains found"))
        return
    }

    // Create node group without domainId
    nodeGroup := model.NodeGroup{
        Name:        req.Name,
        Description: req.Description,
        CNAMEPrefix: cnamePrefix,
        CNAME:       cname,
        Status:      model.NodeGroupStatusActive,
    }

    // Create DNS records for all CDN domains × all available IPs
    if err := h.createDNSRecordsForAllCDNDomains(tx, &nodeGroup, cdnDomains, req.IPIDs); err != nil {
        tx.Rollback()
        httpx.FailErr(c, httpx.ErrDatabaseError("failed to create DNS records", err))
        return
    }

    // Return response with data.item structure
    httpx.OK(c, map[string]interface{}{
        "item": map[string]interface{}{
            "id":          nodeGroup.ID,
            "name":        nodeGroup.Name,
            "description": nodeGroup.Description,
            "cnamePrefix": nodeGroup.CNAMEPrefix,
            "cname":       nodeGroup.CNAME,
            "status":      nodeGroup.Status,
            "createdAt":   nodeGroup.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
            "updatedAt":   nodeGroup.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
        },
    })
}
```

#### 3.4 Update 方法修改

更新 IP 时自动为所有 CDN 域名创建新的 DNS 记录：

```go
func (h *Handler) Update(c *gin.Context) {
    // Handle IP updates
    if req.IPIDs != nil {
        // Mark old DNS records as absent
        if err := h.markDNSRecordsAsAbsent(tx, req.ID); err != nil {
            tx.Rollback()
            httpx.FailErr(c, httpx.ErrDatabaseError("failed to mark old DNS records as absent", err))
            return
        }

        // Fetch all CDN domains
        var cdnDomains []model.Domain
        if err := tx.Where("purpose = ? AND status = ?", "cdn", "active").Find(&cdnDomains).Error; err != nil {
            tx.Rollback()
            httpx.FailErr(c, httpx.ErrDatabaseError("failed to fetch CDN domains", err))
            return
        }

        // Create new DNS records for all CDN domains
        if err := h.createDNSRecordsForAllCDNDomains(tx, &nodeGroup, cdnDomains, req.IPIDs); err != nil {
            tx.Rollback()
            httpx.FailErr(c, httpx.ErrDatabaseError("failed to create DNS records", err))
            return
        }
    }
}
```

#### 3.5 Delete 方法修改

删除时标记 DNS 记录为 absent，由 DNS Worker 负责远端清理：

```go
func (h *Handler) Delete(c *gin.Context) {
    // Mark all DNS records as absent for each node group
    for _, id := range req.IDs {
        if err := h.markDNSRecordsAsAbsent(tx, id); err != nil {
            tx.Rollback()
            httpx.FailErr(c, httpx.ErrDatabaseError("failed to mark DNS records as absent", err))
            return
        }
    }

    // Delete node groups (cascade will delete node_group_ips)
    result := tx.Delete(&model.NodeGroup{}, req.IDs)

    // Return null data for delete operation
    httpx.OK(c, nil)
}
```

#### 3.6 新增 markDNSRecordsAsAbsent 方法

```go
func (h *Handler) markDNSRecordsAsAbsent(tx *gorm.DB, nodeGroupID int) error {
    updates := map[string]interface{}{
        "desired_state": model.DNSRecordDesiredStateAbsent,
    }

    if err := tx.Model(&model.DomainDNSRecord{}).
        Where("owner_type = ? AND owner_id = ?", "node_group", nodeGroupID).
        Updates(updates).Error; err != nil {
        return fmt.Errorf("failed to mark DNS records as absent: %w", err)
    }

    return nil
}
```

## 验收测试

### 验收用例 1：创建 Node Group 传递 domainId 应被忽略

请求：

```bash
curl -X POST http://20.2.140.226:8080/api/v1/node-groups/create \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"name":"test-with-domainid","domainId":9018,"ipIds":[3,4]}'
```

返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "item": {
      "cname": "ng-ac6c9fec46ba5412.4pxtech.com",
      "cnamePrefix": "ng-ac6c9fec46ba5412",
      "createdAt": "2026-01-28T16:25:19+08:00",
      "description": "",
      "id": 7,
      "name": "test-with-domainid",
      "status": "active",
      "updatedAt": "2026-01-28T16:25:19+08:00"
    }
  }
}
```

结果：虽然传递了 domainId 参数，但后端自动忽略，创建成功且返回结果不包含 domainId。

### 验收用例 2：创建 Node Group 不传 domainId 必须成功

请求：

```bash
curl -X POST http://20.2.140.226:8080/api/v1/node-groups/create \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"name":"ng-c008-03-test","ipIds":[3,4]}'
```

返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "item": {
      "cname": "ng-325bd7a49159a821.4pxtech.com",
      "cnamePrefix": "ng-325bd7a49159a821",
      "createdAt": "2026-01-28T16:25:32+08:00",
      "description": "",
      "id": 8,
      "name": "ng-c008-03-test",
      "status": "active",
      "updatedAt": "2026-01-28T16:25:32+08:00"
    }
  }
}
```

结果：创建成功，data.item 结构符合规范，所有字段使用 lowerCamelCase，不包含 domainId。

### 验收用例 3：创建后自动生成 DNS Records

数据库查询：

```sql
SELECT r.id, d.domain, r.type, r.name, r.value, r.owner_type, r.owner_id, r.status, r.desired_state 
FROM domain_dns_records r 
JOIN domains d ON r.domain_id = d.id 
WHERE r.owner_type = "node_group" AND r.owner_id = 8 
ORDER BY d.domain, r.value;
```

查询结果：

```
id    domain         type  name                    value            owner_type   owner_id  status   desired_state
167   4pxtech.com    A     ng-325bd7a49159a821     104.208.76.193   node_group   8         active   present
166   4pxtech.com    A     ng-325bd7a49159a821     20.2.140.226     node_group   8         active   present
169   wx40.xyz       A     ng-325bd7a49159a821     104.208.76.193   node_group   8         active   present
168   wx40.xyz       A     ng-325bd7a49159a821     20.2.140.226     node_group   8         active   present
```

结果验证：
- CDN 域名数量：2（4pxtech.com 和 wx40.xyz）
- IP 数量：2（20.2.140.226 和 104.208.76.193）
- 生成的记录数量：4 = 2 domains × 2 IPs
- 所有记录 type=A，status=active，desired_state=present
- name 字段为相对名称（ng-325bd7a49159a821）

### 验收用例 4：List Node Groups 不再出现 domainId

请求：

```bash
curl "http://20.2.140.226:8080/api/v1/node-groups?page=1&pageSize=20" \
  -H "Authorization: Bearer <TOKEN>"
```

返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 8,
        "name": "ng-c008-03-test",
        "description": "",
        "cnamePrefix": "ng-325bd7a49159a821",
        "cname": "ng-325bd7a49159a821.4pxtech.com",
        "status": "active",
        "ipCount": 2,
        "createdAt": "2026-01-28T16:25:32+08:00",
        "updatedAt": "2026-01-28T16:25:32+08:00"
      },
      {
        "id": 7,
        "name": "test-with-domainid",
        "description": "",
        "cnamePrefix": "ng-ac6c9fec46ba5412",
        "cname": "ng-ac6c9fec46ba5412.4pxtech.com",
        "status": "active",
        "ipCount": 2,
        "createdAt": "2026-01-28T16:25:19+08:00",
        "updatedAt": "2026-01-28T16:25:19+08:00"
      }
    ],
    "total": 4,
    "page": 1,
    "pageSize": 20
  }
}
```

结果：列表接口返回结构符合 data.items 规范，所有字段使用 lowerCamelCase，不包含 domainId。

### 验收用例 5：删除 Node Group 触发 DNS Records absent

请求：

```bash
curl -X POST http://20.2.140.226:8080/api/v1/node-groups/delete \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"ids":[9]}'
```

返回：

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

数据库查询：

```sql
SELECT id, domain_id, type, name, value, owner_type, owner_id, status, desired_state 
FROM domain_dns_records 
WHERE owner_type = "node_group" AND owner_id = 9;
```

查询结果：

```
id    domain_id  type  name                    value            owner_type   owner_id  status   desired_state
170   9018       A     ng-ccc94f2f3b2f6056     20.2.140.226     node_group   9         active   absent
171   9018       A     ng-ccc94f2f3b2f6056     104.208.76.193   node_group   9         active   absent
172   9039       A     ng-ccc94f2f3b2f6056     20.2.140.226     node_group   9         active   absent
173   9039       A     ng-ccc94f2f3b2f6056     104.208.76.193   node_group   9         active   absent
```

结果：
- Node Group 已被删除（id=9 不存在于 node_groups 表）
- DNS 记录保留但 desired_state 已标记为 absent
- 等待 DNS Worker 清理这些记录
- 删除接口返回 data=null 符合规范

## 回滚策略

选择 A：不回滚字段删除，只能回滚代码

原因：domain_id 字段语义错误，Node Group 应该只描述 IP 组，与域名的关系应该只通过 domain_dns_records 表体现。不建议恢复此字段。

代码回滚命令：

```bash
git revert <commit_hash>
```

如果必须紧急回滚数据库（不推荐）：

```sql
ALTER TABLE `node_groups` ADD COLUMN `domain_id` bigint NOT NULL AFTER `description`;
ALTER TABLE `node_groups` ADD INDEX `idx_node_groups_domain_id` (`domain_id`);
ALTER TABLE `node_groups` ADD CONSTRAINT `fk_node_groups_domain` FOREIGN KEY (`domain_id`) REFERENCES `domains` (`id`);
```

## 部署信息

服务器：20.2.140.226

服务端口：8080

配置文件：/opt/go_cmdb/config.ini

服务状态：运行中

## 任务完成情况

任务 C0-08-03 已完成所有要求：

1. 从数据库表中移除 node_groups.domain_id 字段
2. 从 Model 中移除 DomainID 字段和 Domain 关联
3. 从所有 API 中移除 domainId 参数和返回值
4. Node Group 与域名的关系仅通过 domain_dns_records 表体现
5. 所有接口返回符合项目规范（lowerCamelCase、data.items/data.item 结构）
6. 删除操作正确标记 DNS 记录为 absent，由 DNS Worker 负责远端清理
7. 通过完整验收测试（5 个用例全部通过）
8. 交付报告不包含任何 emoji 或装饰符号
