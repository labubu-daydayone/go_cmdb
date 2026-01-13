# CMDB运维管理系统 - React前端

基于React的CMDB运维管理系统前端，提供用户管理、权限管理、WebSocket实时更新等功能。

## 功能特性

### 🔐 用户管理
- 用户列表展示（实时WebSocket更新）
- 创建新用户
- 删除用户
- 修改密码

### 👥 权限管理
- 角色管理（创建、查看）
- 权限管理（创建、查看）
- 权限组管理（创建、查看）
- 权限分配

### ⚡ 实时功能
- WebSocket实时连接
- 用户列表实时更新
- 权限变化实时推送

### 🎨 界面设计
- 现代化UI设计
- 响应式布局
- 深色/浅色主题支持
- 侧边栏导航

## 技术栈

- **框架**: React 19 + TypeScript
- **路由**: Wouter
- **UI组件**: shadcn/ui
- **样式**: Tailwind CSS 4
- **HTTP客户端**: Axios
- **WebSocket**: 原生WebSocket API
- **构建工具**: Vite

## 快速开始

### 前置条件
- Node.js 18+
- npm 或 pnpm

### 安装依赖

```bash
pnpm install
```

### 开发模式

```bash
pnpm dev
```

服务器将在 `http://localhost:3000` 启动

### 构建生产版本

```bash
pnpm build
```

### 启动生产服务器

```bash
pnpm start
```

## 环境配置

### 开发环境

编辑 `package.json` 中的 `dev` 脚本，修改API地址：

```json
"dev": "VITE_API_URL=http://localhost:8080/api/v1 VITE_WS_URL=ws://localhost:8080/api/v1/ws/connect vite --host"
```

### 生产环境

编辑 `package.json` 中的 `build` 脚本，修改API地址：

```json
"build": "VITE_API_URL=/api/v1 VITE_WS_URL=ws://api/v1/ws/connect vite build && ..."
```

## 项目结构

```
client/
├── public/              # 静态资源
├── src/
│   ├── components/      # 可复用组件
│   │   ├── Layout.tsx   # 主布局
│   │   └── ...
│   ├── contexts/        # React Context
│   │   └── AuthContext.tsx  # 认证上下文
│   ├── hooks/           # 自定义Hook
│   │   └── useWebSocket.ts  # WebSocket Hook
│   ├── lib/             # 工具函数
│   │   └── api.ts       # API调用
│   ├── pages/           # 页面组件
│   │   ├── Login.tsx    # 登录页
│   │   ├── Dashboard.tsx # 仪表板
│   │   ├── Users.tsx    # 用户管理
│   │   └── Permissions.tsx # 权限管理
│   ├── App.tsx          # 根组件
│   ├── main.tsx         # 入口文件
│   └── index.css        # 全局样式
├── index.html           # HTML模板
└── package.json         # 项目配置
```

## API集成

### 认证

```typescript
import { authAPI } from '@/lib/api';

// 登录
await authAPI.login(username, password);

// 注册
await authAPI.register(username, email, password);
```

### 用户管理

```typescript
import { userAPI } from '@/lib/api';

// 获取用户列表
await userAPI.list(page, pageSize);

// 创建用户
await userAPI.create(username, email, password);

// 删除用户
await userAPI.delete(userId);

// 修改密码
await userAPI.changePassword(oldPassword, newPassword);
```

### 权限管理

```typescript
import { permissionAPI } from '@/lib/api';

// 创建角色
await permissionAPI.createRole(name, description);

// 创建权限
await permissionAPI.create(name, description, action, resource);

// 创建权限组
await permissionAPI.createGroup(name, description);
```

## WebSocket实时更新

### 使用Hook

```typescript
import { useWebSocket } from '@/hooks/useWebSocket';

const { isConnected, subscribe, unsubscribe } = useWebSocket(token, {
  onMessage: (message) => {
    console.log('Received:', message);
  },
});

// 订阅频道
subscribe('user:list');

// 取消订阅
unsubscribe('user:list');
```

### 消息格式

```typescript
interface WebSocketMessage {
  type: string;           // 消息类型
  action: string;         // 操作类型 (create, update, delete)
  resource: string;       // 资源类型 (user, domain, etc)
  resource_id?: string;   // 资源ID
  data: any;              // 数据
  timestamp: number;      // 时间戳
  user_id: string;        // 用户ID
}
```

## 认证流程

### 登录

1. 用户输入用户名和密码
2. 调用 `authAPI.login()`
3. 获取Token和用户信息
4. 保存到localStorage
5. 重定向到仪表板

### 路由保护

所有需要认证的路由都通过 `AuthProvider` 和 `useAuth` Hook进行保护。

```typescript
const { isAuthenticated, user, token } = useAuth();

if (!isAuthenticated) {
  // 重定向到登录页
}
```

## 常见问题

### Q: 如何修改API地址？

A: 修改 `package.json` 中的 `dev` 或 `build` 脚本中的 `VITE_API_URL` 和 `VITE_WS_URL` 环境变量。

### Q: WebSocket连接失败怎么办？

A: 检查以下几点：
1. 后端服务器是否正在运行
2. WebSocket URL是否正确
3. Token是否有效
4. 浏览器控制台是否有错误信息

### Q: 如何添加新的页面？

A: 
1. 在 `client/src/pages/` 创建新的页面组件
2. 在 `App.tsx` 中添加路由
3. 在 `Layout.tsx` 中添加导航项

### Q: 如何自定义主题？

A: 编辑 `client/src/index.css` 中的CSS变量：

```css
:root {
  --primary: #your-color;
  --background: #your-color;
  /* ... 其他变量 */
}
```

## 后端API文档

详见后端仓库：https://github.com/labubu-daydayone/go_cmdb

## 部署

### Docker部署

```bash
docker build -t cmdb-web:latest .
docker run -p 3000:3000 cmdb-web:latest
```

### Docker Compose

```bash
docker-compose up -d
```

## 贡献指南

欢迎提交Issue和Pull Request！

## 许可证

MIT License
