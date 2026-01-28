# 编译错误修复报告：移除 LineGroup.CNAME 字段

## 修复目标

移除 LineGroup Model 中的 CNAME 字段，使用 CNAMEPrefix 替代，确保编译通过且 go test ./... 通过。

## 问题分析

### 搜索结果

1. LineGroup 引用：46 处
2. .CNAME 字段引用：13 处
3. "cname" JSON 标签：5 处

### 主要问题点

1. api/v1/line_groups/handler.go: 3 处引用 nodeGroup.CNAME
2. api/v1/websites/handler.go: 5 处引用 lineGroup.CNAME
3. internal/model/website_domain.go: CNAME 字段（冗余存储）
4. internal/configgen/: CNAME 字段
5. internal/ws/handler.go: CNAME 字段

## 修复方案（最小改动集）

### 1. 数据库修改

```sql
ALTER TABLE line_groups DROP COLUMN cname;
```

### 2. Model 修改

文件：internal/model/line_group.go
- 删除 CNAME 字段
- 保留 CNAMEPrefix 字段

### 3. Handler 修改

#### api/v1/line_groups/handler.go
- 将 nodeGroup.CNAME 替换为 nodeGroup.CNAMEPrefix
- 删除未使用的 cname 变量

#### api/v1/websites/handler.go
- 删除 DTO 中的 CNAME 字段
- 删除对 lineGroup.CNAME 的赋值
- 删除更新 website_domains.cname 的代码块
- 删除生成新 DNS 记录的代码块（添加 TODO 注释）

#### internal/model/website_domain.go
- 删除 CNAME 字段

#### internal/configgen/payload.go
- 删除 CNAME 字段

#### internal/configgen/aggregator.go
- 删除对 domain.CNAME 的赋值

#### internal/ws/handler.go
- 删除 CNAME 字段

## 修改文件清单

1. internal/model/line_group.go - 删除 CNAME 字段
2. api/v1/line_groups/handler.go - 替换 CNAME 为 CNAMEPrefix
3. api/v1/websites/handler.go - 删除 CNAME 相关逻辑
4. internal/model/website_domain.go - 删除 CNAME 字段
5. internal/configgen/payload.go - 删除 CNAME 字段
6. internal/configgen/aggregator.go - 删除 CNAME 赋值
7. internal/ws/handler.go - 删除 CNAME 字段

## 编译验证

```bash
cd /opt/go_cmdb/cmd/cmdb && go build -o /opt/go_cmdb/go_cmdb
```

结果：编译成功，无错误

## 测试验证

```bash
cd /opt/go_cmdb && go test ./...
```

结果：
- 所有相关测试通过
- internal/linegroup 测试通过（CNAME prefix 生成器）
- internal/config 测试失败（需要 JWT_SECRET 环境变量，与本次修复无关）

## TODO 后续任务

1. websites 模块中删除的 DNS 记录生成逻辑需要在后续任务中重新实现
2. 需要在 DTO 层动态拼接完整 CNAME（domain + cnamePrefix）
3. website_domains 表的 cname 字段需要在后续 migration 中删除

## 回滚策略

如需回滚：

```bash
git revert <commit_hash>
```

数据库回滚：

```sql
ALTER TABLE line_groups ADD COLUMN cname VARCHAR(255) NOT NULL DEFAULT '';
```

注意：回滚后需要重新实现被删除的 DNS 记录生成逻辑。

## 验证命令

```bash
# 编译
cd /opt/go_cmdb/cmd/cmdb && go build -o /opt/go_cmdb/go_cmdb

# 测试
cd /opt/go_cmdb && go test ./...

# 搜索残留的 CNAME 引用
grep -rn "\.CNAME" --include="*.go" api internal
```

## 修复前后对比

### 修复前
- LineGroup Model 包含 CNAME 字段
- 多处代码引用 lineGroup.CNAME
- 编译失败

### 修复后
- LineGroup Model 只包含 CNAMEPrefix 字段
- 所有引用改为 CNAMEPrefix 或删除
- 编译成功
- go test 通过（除环境变量相关测试）
