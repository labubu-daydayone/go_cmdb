# 任务 C0-11 交付报告

## 任务信息

任务编号：C0-11
任务名称：P0 Node 列表补全 ips.items（包含主 IP）+ 字段命名统一
任务归属：控制端（go_cmdb）
优先级：P0
完成时间：2026-01-28

## 任务目标

Node 列表接口必须能让前端明确区分"节点启用/禁用"和"IP 启用/禁用"，并且主 IP 必须出现在 ips.items 中；同时对外 JSON 字段严格统一为 lowerCamelCase。

## 核心改动

### 1. 新增 DTO 结构体

新增文件：
- internal/dto/node_list.go
- internal/dto/node_detail.go

新增 DTO 结构体：
- NodeListItemDTO：Node 列表项，包含 ips.items 结构
- NodeDetailItemDTO：Node 详情项，包含 ips.items 结构
- NodeIPsContainerDTO：IP 容器，包含 items 数组
- NodeIPItemDTO：IP 项，包含 id, ip, isMain, ipEnabled 字段

所有 DTO 字段使用 lowerCamelCase 命名：
- nodeEnabled：节点启用/禁用状态（来自 nodes.enabled）
- ipEnabled：IP 启用/禁用状态（来自 node_ips.enabled）
- isMain：是否为主 IP（来自 node_ips.ip_type）
- ips.items：包含主 IP 和子 IP 的数组

### 2. 修改 Node 列表接口

修改文件：api/v1/nodes/handler.go

修改内容：
- ListResponse 使用 NodeListItemDTO 代替 NodeDTO
- List handler 中构建 ips.items 数组，包含所有 IP（主 IP 和子 IP）
- 主 IP 标记 isMain=true
- 所有 IP 包含 ipEnabled 字段
- 移除 subIps 字段输出
- enabled 字段重命名为 nodeEnabled

### 3. 修改 Node 详情接口

修改文件：api/v1/nodes/handler.go

修改内容：
- Get handler 使用 NodeDetailItemDTO 代替 NodeDetailDTO
- 构建 ips.items 数组，包含所有 IP
- 返回结构改为 data.item
- 移除 subIps 字段输出
- enabled 字段重命名为 nodeEnabled

## 字段统一保证

### 1. DTO 定义

所有 DTO 的 json 标签全部写死为 lowerCamelCase：
- nodeEnabled（不是 enabled 或 node_enabled）
- ipEnabled（不是 enabled 或 ip_enabled）
- isMain（不是 is_main 或 IsMain）
- mainIp（不是 main_ip 或 MainIP）

### 2. 禁止直接返回 GORM Model

所有接口都使用 DTO 构建返回结构，不直接返回 GORM Model，确保字段命名可控。

### 3. 语义明确

- nodeEnabled：明确表示节点级别的启用/禁用状态
- ipEnabled：明确表示 IP 级别的启用/禁用状态
- 前端可以独立控制节点和 IP 的启用状态

## 验收测试结果

测试环境：http://20.2.140.226:8080
测试用户：admin / admin123
测试节点：Node ID 12

### 验收 1：Node 列表字段与结构

请求：
```bash
curl -s "http://20.2.140.226:8080/api/v1/nodes?page=1&pageSize=15" \
  -H "Authorization: Bearer $TOKEN"
```

返回：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 12,
        "name": "Node-226",
        "mainIp": "20.2.140.226",
        "agentPort": 18080,
        "nodeEnabled": true,
        "agentStatus": "online",
        "lastSeenAt": "2026-01-28T18:15:02+08:00",
        "healthFailCount": 0,
        "ips": {
          "items": [
            {
              "id": 4,
              "ip": "104.208.76.193",
              "isMain": false,
              "ipEnabled": true
            },
            {
              "id": 3,
              "ip": "20.2.140.226",
              "isMain": true,
              "ipEnabled": true
            }
          ]
        },
        "createdAt": "2026-01-28T07:05:20.977+08:00",
        "updatedAt": "2026-01-28T18:15:01.939+08:00"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 15
  }
}
```

验收点：
- code=0：通过
- data.items 存在：通过
- items[0].nodeEnabled 存在且为 boolean：通过（true）
- items[0].ips.items 存在且为数组：通过（2 条记录）
- ips.items 至少包含 1 条 isMain=true 的记录：通过（id=3, ip=20.2.140.226）
- 返回体中不出现 subIps 字段：通过
- 返回体中不出现 main_ip/SubIPs/sub_ips 等非 lowerCamelCase 字段：通过

### 验收 2：主 IP 出现在 ips.items

从验收 1 返回中：
- mainIp 的值：20.2.140.226
- ips.items 中存在同样 ip 且 isMain=true：通过（id=3）

### 验收 3：IP 开关可见

从验收 1 返回中：
- ips.items 每一项都有 ipEnabled 字段：通过
- id=4 的 IP：ipEnabled=true
- id=3 的 IP：ipEnabled=true

### 验收 4：详情接口

请求：
```bash
curl -s "http://20.2.140.226:8080/api/v1/nodes/12" \
  -H "Authorization: Bearer $TOKEN"
```

返回：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "item": {
      "id": 12,
      "name": "Node-226",
      "mainIp": "20.2.140.226",
      "agentPort": 18080,
      "nodeEnabled": true,
      "agentStatus": "online",
      "lastSeenAt": "2026-01-28T18:15:22+08:00",
      "healthFailCount": 0,
      "identity": {
        "id": 11,
        "fingerprint": "d823c847d0cf6bf3a6331c85f93a1f98218e23d737e7bdc328f9472881296d38"
      },
      "ips": {
        "items": [
          {
            "id": 4,
            "ip": "104.208.76.193",
            "isMain": false,
            "ipEnabled": true
          },
          {
            "id": 3,
            "ip": "20.2.140.226",
            "isMain": true,
            "ipEnabled": true
          }
        ]
      },
      "createdAt": "2026-01-28T07:05:20.977+08:00",
      "updatedAt": "2026-01-28T18:15:21.938+08:00"
    }
  }
}
```

验收点：
- code=0：通过
- data.item.ips.items 存在：通过
- 不出现 subIps：通过

## 改动文件清单

### 新增文件

1. internal/dto/node_list.go
   - NodeListItemDTO
   - NodeIPsContainerDTO
   - NodeIPItemDTO

2. internal/dto/node_detail.go
   - NodeDetailItemDTO

### 修改文件

1. api/v1/nodes/handler.go
   - ListResponse 结构体
   - List 方法：构建 ips.items，移除 subIps
   - Get 方法：构建 ips.items，移除 subIps，返回 data.item

## 回滚策略

### 代码回滚

```bash
git revert <commit_hash>
cd /opt/go_cmdb/cmd/cmdb
go build -o /opt/go_cmdb/go_cmdb
killall -9 go_cmdb
cd /opt/go_cmdb
nohup ./go_cmdb --config /opt/go_cmdb/config.ini > /tmp/go_cmdb_start.log 2>&1 &
```

### 兼容回滚

如果前端已依赖 subIps 字段，回滚后需要临时恢复 subIps 输出：

在 List 和 Get 方法中添加：
```go
// 临时兼容：同时输出 subIps 和 ips.items
var subIps []dto.SubIpDTO
for _, ip := range node.IPs {
    if ip.IPType != model.NodeIPTypeMain {
        subIps = append(subIps, dto.SubIpDTO{
            ID:      ip.ID,
            IP:      ip.IP,
            Enabled: ip.Enabled,
        })
    }
}
// 在返回结构中添加 SubIps 字段
```

注意：这只是临时兼容方案，不在本任务实现范围内。

## 提交信息

仓库：labubu-daydayone/go_cmdb
分支：main
提交信息：Fix C0-11: Node list includes main IP in ips.items and unifies field naming to lowerCamelCase
提交日期：2026-01-28

## 部署信息

服务地址：http://20.2.140.226:8080
二进制文件：/opt/go_cmdb/go_cmdb
配置文件：/opt/go_cmdb/config.ini
数据库：20.2.140.226:3306/cdn_control

## 总结

任务 C0-11 已成功完成，所有 P0 要求均已满足：

1. Node 列表接口返回 ips.items，包含主 IP 和子 IP
2. 主 IP 标记 isMain=true
3. 所有字段使用 lowerCamelCase
4. nodeEnabled 表示节点开关，ipEnabled 表示 IP 开关
5. 移除 subIps 字段
6. 详情接口同步修改
7. 所有验收用例通过
8. 使用 DTO 确保字段命名统一
9. 交付报告不包含任何 emoji 或装饰符号
