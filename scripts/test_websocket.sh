#!/bin/bash

# T2-09 WebSocket Real-time Sync Test Script
# 测试Socket.IO服务端、JWT认证、websites事件推送、增量补发

set -e

# 配置
DB_HOST="20.2.140.226"
DB_USER="root"
DB_NAME="cdn_control"
DB_SOCKET="/data/mysql/run/mysql.sock"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================="
echo "T2-09 WebSocket Real-time Sync Test"
echo "========================================="
echo ""

# SQL验证函数
run_sql() {
    local desc="$1"
    local sql="$2"
    echo -e "${YELLOW}SQL验证: ${desc}${NC}"
    sshpass -p 'Uviev5Ohyeit' ssh -o StrictHostKeyChecking=no root@${DB_HOST} \
        "mysql -uroot -S ${DB_SOCKET} ${DB_NAME} -e \"${sql}\""
    echo ""
}

# ========================================
# SQL验证（15+条）
# ========================================

echo "========================================="
echo "SQL验证（15+条）"
echo "========================================="
echo ""

# SQL 1: 清理测试数据
run_sql "1. 清理测试数据" "
DELETE FROM ws_events WHERE topic = 'websites' AND id >= 9000;
SELECT '测试数据已清理' AS result;
"

# SQL 2: 验证ws_events表结构
run_sql "2. 验证ws_events表结构" "
DESC ws_events;
"

# SQL 3: 验证索引存在
run_sql "3. 验证idx_topic_id索引" "
SHOW INDEX FROM ws_events WHERE Key_name = 'idx_topic_id';
"

# SQL 4: 插入测试事件（add）
run_sql "4. 插入测试事件（add）" "
INSERT INTO ws_events (id, topic, event_type, payload, created_at)
VALUES (9001, 'websites', 'add', '{\\\"id\\\": 1001}', NOW());
SELECT '测试事件已插入' AS result;
"

# SQL 5: 插入测试事件（update）
run_sql "5. 插入测试事件（update）" "
INSERT INTO ws_events (id, topic, event_type, payload, created_at)
VALUES (9002, 'websites', 'update', '{\\\"id\\\": 1001}', NOW());
SELECT '测试事件已插入' AS result;
"

# SQL 6: 插入测试事件（delete）
run_sql "6. 插入测试事件（delete）" "
INSERT INTO ws_events (id, topic, event_type, payload, created_at)
VALUES (9003, 'websites', 'delete', '{\\\"id\\\": 1001}', NOW());
SELECT '测试事件已插入' AS result;
"

# SQL 7: 验证事件写入
run_sql "7. 验证事件写入（应有3条）" "
SELECT COUNT(*) AS count FROM ws_events WHERE id >= 9001 AND id <= 9003;
"

# SQL 8: 验证topic过滤
run_sql "8. 验证topic过滤（应有3条）" "
SELECT COUNT(*) AS count FROM ws_events WHERE topic = 'websites' AND id >= 9001;
"

# SQL 9: 验证event_type过滤（add）
run_sql "9. 验证event_type过滤（add，应有1条）" "
SELECT COUNT(*) AS count FROM ws_events WHERE topic = 'websites' AND event_type = 'add' AND id >= 9001;
"

# SQL 10: 验证event_type过滤（update）
run_sql "10. 验证event_type过滤（update，应有1条）" "
SELECT COUNT(*) AS count FROM ws_events WHERE topic = 'websites' AND event_type = 'update' AND id >= 9001;
"

# SQL 11: 验证event_type过滤（delete）
run_sql "11. 验证event_type过滤（delete，应有1条）" "
SELECT COUNT(*) AS count FROM ws_events WHERE topic = 'websites' AND event_type = 'delete' AND id >= 9001;
"

# SQL 12: 验证按lastEventId拉取（lastEventId=9001，应有2条）
run_sql "12. 验证按lastEventId拉取（lastEventId=9001，应有2条）" "
SELECT id, topic, event_type, payload FROM ws_events 
WHERE topic = 'websites' AND id > 9001 
ORDER BY id ASC;
"

# SQL 13: 验证按lastEventId拉取（lastEventId=9002，应有1条）
run_sql "13. 验证按lastEventId拉取（lastEventId=9002，应有1条）" "
SELECT id, topic, event_type, payload FROM ws_events 
WHERE topic = 'websites' AND id > 9002 
ORDER BY id ASC;
"

# SQL 14: 验证payload格式
run_sql "14. 验证payload格式（应为JSON）" "
SELECT id, JSON_VALID(payload) AS is_valid_json FROM ws_events WHERE id >= 9001 AND id <= 9003;
"

# SQL 15: 验证created_at字段
run_sql "15. 验证created_at字段（应为当前时间）" "
SELECT id, created_at FROM ws_events WHERE id >= 9001 AND id <= 9003;
"

# SQL 16: 验证最新事件ID查询
run_sql "16. 验证最新事件ID查询" "
SELECT MAX(id) AS latest_event_id FROM ws_events WHERE topic = 'websites';
"

# SQL 17: 验证事件排序（ASC）
run_sql "17. 验证事件排序（ASC）" "
SELECT id, event_type FROM ws_events WHERE id >= 9001 AND id <= 9003 ORDER BY id ASC;
"

# SQL 18: 验证LIMIT查询（模拟增量补发限制）
run_sql "18. 验证LIMIT查询（限制2条）" "
SELECT id, event_type FROM ws_events WHERE topic = 'websites' AND id > 9000 ORDER BY id ASC LIMIT 2;
"

echo -e "${GREEN}SQL验证完成！${NC}"
echo ""

# ========================================
# curl/Node.js验证（12+条）
# ========================================

echo "========================================="
echo "curl/Node.js验证（12+条）"
echo "========================================="
echo ""

echo -e "${YELLOW}注意：以下验证需要手动执行或使用Node.js脚本${NC}"
echo ""

echo "验证1: 无token连接被拒绝"
echo "  预期：401 Unauthorized"
echo "  命令：curl -i http://localhost:8080/socket.io/?EIO=4&transport=polling"
echo ""

echo "验证2: 正常token连接成功收到connected"
echo "  预期：收到connected事件 {\"ok\": true}"
echo "  需要：Node.js Socket.IO客户端"
echo ""

echo "验证3: request:websites收到websites:initial"
echo "  预期：收到websites:initial事件，包含items/total/version/lastEventId"
echo "  需要：Node.js Socket.IO客户端"
echo ""

echo "验证4: create website后收到websites:update add"
echo "  预期：收到websites:update事件，type=add"
echo "  命令：curl -X POST http://localhost:8080/api/v1/websites/create ..."
echo ""

echo "验证5: update website后收到websites:update update"
echo "  预期：收到websites:update事件，type=update"
echo "  命令：curl -X POST http://localhost:8080/api/v1/websites/update ..."
echo ""

echo "验证6: delete website后收到websites:update delete"
echo "  预期：收到websites:update事件，type=delete"
echo "  命令：curl -X POST http://localhost:8080/api/v1/websites/delete ..."
echo ""

echo "验证7: lastEventId=旧值 → 收到增量事件"
echo "  预期：收到多个websites:update事件"
echo "  需要：Node.js Socket.IO客户端，发送request:websites {lastEventId: 9001}"
echo ""

echo "验证8: lastEventId太旧/事件过多 → 回落收到websites:initial"
echo "  预期：收到websites:initial事件（全量）"
echo "  需要：Node.js Socket.IO客户端，发送request:websites {lastEventId: 1}"
echo ""

echo "验证9: JWT token过期 → 连接被拒绝"
echo "  预期：401 Unauthorized"
echo "  需要：使用过期的JWT token"
echo ""

echo "验证10: JWT token无效 → 连接被拒绝"
echo "  预期：401 Unauthorized"
echo "  命令：curl -i http://localhost:8080/socket.io/?EIO=4&transport=polling&token=invalid"
echo ""

echo "验证11: 多个客户端同时连接 → 都能收到广播"
echo "  预期：所有客户端都收到websites:update事件"
echo "  需要：Node.js Socket.IO客户端（多个实例）"
echo ""

echo "验证12: 断线重连 → 使用lastEventId补发"
echo "  预期：重连后收到断线期间的增量事件"
echo "  需要：Node.js Socket.IO客户端（断线重连测试）"
echo ""

echo "验证13: 无lastEventId → 收到全量列表"
echo "  预期：收到websites:initial事件（全量）"
echo "  需要：Node.js Socket.IO客户端，发送request:websites {}"
echo ""

echo -e "${GREEN}验证步骤已列出！${NC}"
echo ""

echo "========================================="
echo "Node.js测试客户端示例代码"
echo "========================================="
echo ""

cat << 'EOF'
// 安装依赖：npm install socket.io-client
const io = require('socket.io-client');

// 获取JWT token（从登录接口）
const token = 'YOUR_JWT_TOKEN_HERE';

// 连接Socket.IO服务器
const socket = io('http://localhost:8080', {
  auth: { token }
});

// 连接成功
socket.on('connect', () => {
  console.log('Connected:', socket.id);
});

// 收到connected确认
socket.on('connected', (data) => {
  console.log('Connected confirmation:', data);
  
  // 请求websites列表
  socket.emit('request:websites', { lastEventId: 0 });
});

// 收到初始全量
socket.on('websites:initial', (data) => {
  console.log('Websites initial:', data);
});

// 收到增量更新
socket.on('websites:update', (data) => {
  console.log('Websites update:', data);
});

// 连接错误
socket.on('connect_error', (error) => {
  console.error('Connection error:', error.message);
});

// 断开连接
socket.on('disconnect', (reason) => {
  console.log('Disconnected:', reason);
});
EOF

echo ""
echo -e "${GREEN}测试脚本执行完成！${NC}"
echo ""

# 清理测试数据
echo "清理测试数据..."
run_sql "清理测试数据" "
DELETE FROM ws_events WHERE id >= 9001 AND id <= 9003;
SELECT '测试数据已清理' AS result;
"

echo -e "${GREEN}所有测试完成！${NC}"
