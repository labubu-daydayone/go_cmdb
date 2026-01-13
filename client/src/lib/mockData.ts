/**
 * Mock数据服务
 * 用于前端开发时模拟后端API响应
 */

// Mock用户数据
let mockUsers = [
  {
    id: '1',
    username: 'admin',
    email: 'admin@example.com',
    password: 'admin123',
    is_admin: true,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: '2',
    username: 'user1',
    email: 'user1@example.com',
    password: 'user123',
    is_admin: false,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
];

// Mock角色权限关联
let mockRolePermissions: { [roleId: string]: string[] } = {
  '1': ['1', '2', '3', '4', '5', '6', '7', '8', '9', '10', '11', '12'], // Admin角色拥有所有权限
  '2': ['1', '5', '9'], // User角色只有读取权限
};

// Mock角色数据
let mockRoles = [
  {
    id: '1',
    name: 'Admin',
    description: '系统管理员',
    created_at: new Date().toISOString(),
  },
  {
    id: '2',
    name: 'User',
    description: '普通用户',
    created_at: new Date().toISOString(),
  },
];

// Mock权限数据
let mockPermissions = [
  // 用户管理权限
  {
    id: '1',
    name: '用户管理-读取',
    description: '查看用户列表和详情',
    action: 'read',
    resource: 'user',
    created_at: new Date().toISOString(),
  },
  {
    id: '2',
    name: '用户管理-创建',
    description: '创建新用户',
    action: 'create',
    resource: 'user',
    created_at: new Date().toISOString(),
  },
  {
    id: '3',
    name: '用户管理-更新',
    description: '编辑用户信息',
    action: 'update',
    resource: 'user',
    created_at: new Date().toISOString(),
  },
  {
    id: '4',
    name: '用户管理-删除',
    description: '删除用户',
    action: 'delete',
    resource: 'user',
    created_at: new Date().toISOString(),
  },
  // 权限管理权限
  {
    id: '5',
    name: '权限管理-读取',
    description: '查看权限、角色、权限组列表',
    action: 'read',
    resource: 'permission',
    created_at: new Date().toISOString(),
  },
  {
    id: '6',
    name: '权限管理-创建',
    description: '创建权限、角色、权限组',
    action: 'create',
    resource: 'permission',
    created_at: new Date().toISOString(),
  },
  {
    id: '7',
    name: '权限管理-更新',
    description: '编辑权限、角色、权限组',
    action: 'update',
    resource: 'permission',
    created_at: new Date().toISOString(),
  },
  {
    id: '8',
    name: '权限管理-删除',
    description: '删除权限、角色、权限组',
    action: 'delete',
    resource: 'permission',
    created_at: new Date().toISOString(),
  },
  // 域名管理权限
  {
    id: '9',
    name: '域名管理-读取',
    description: '查看域名列表和详情',
    action: 'read',
    resource: 'domain',
    created_at: new Date().toISOString(),
  },
  {
    id: '10',
    name: '域名管理-创建',
    description: '添加新域名',
    action: 'create',
    resource: 'domain',
    created_at: new Date().toISOString(),
  },
  {
    id: '11',
    name: '域名管理-更新',
    description: '编辑域名配置',
    action: 'update',
    resource: 'domain',
    created_at: new Date().toISOString(),
  },
  {
    id: '12',
    name: '域名管理-删除',
    description: '删除域名',
    action: 'delete',
    resource: 'domain',
    created_at: new Date().toISOString(),
  },
];

// Mock权限组数据
let mockGroups = [
  {
    id: '1',
    name: 'Group A',
    description: '权限组A',
    created_at: new Date().toISOString(),
  },
];

// 模拟延迟
const delay = (ms: number = 300) => new Promise(resolve => setTimeout(resolve, ms));

// 统一响应格式
const response = (code: number = 200, message: string = 'success', data: any = null) => ({
  code,
  message,
  data,
  timestamp: new Date().toISOString(),
});

// Mock API
export const mockAPI = {
  // ===== 认证相关 =====
  auth: {
    login: async (username: string, password: string) => {
      await delay();
      const user = mockUsers.find(u => u.username === username && u.password === password);
      if (!user) {
        throw { code: 401, message: 'Invalid username or password' };
      }
      const { password: _, ...userWithoutPassword } = user;
      return response(200, 'Login successful', {
        token: 'mock-token-' + user.id,
        ...userWithoutPassword,
      });
    },
    register: async (username: string, email: string, password: string) => {
      await delay();
      const newUser = {
        id: String(mockUsers.length + 1),
        username,
        email,
        password,
        is_admin: false,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };
      mockUsers.push(newUser);
      const { password: _, ...userWithoutPassword } = newUser;
      return response(200, 'Registration successful', {
        token: 'mock-token-' + newUser.id,
        ...userWithoutPassword,
      });
    },
    refreshToken: async (token: string) => {
      await delay();
      return response(200, 'Token refreshed', { token: token + '-refreshed' });
    },
  },

  // ===== 用户管理 =====
  user: {
    create: async (username: string, email: string, password: string) => {
      await delay();
      const newUser = {
        id: String(mockUsers.length + 1),
        username,
        email,
        password,
        is_admin: false,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };
      mockUsers.push(newUser);
      const { password: _, ...userWithoutPassword } = newUser;
      return response(200, 'User created', userWithoutPassword);
    },
    list: async (page: number = 1, pageSize: number = 10) => {
      await delay();
      const start = (page - 1) * pageSize;
      const end = start + pageSize;
      const items = mockUsers.slice(start, end).map(({ password, ...user }) => user);
      return response(200, 'Users fetched', {
        items,
        total: mockUsers.length,
        page,
        page_size: pageSize,
      });
    },
    get: async (id: string) => {
      await delay();
      const user = mockUsers.find(u => u.id === id);
      if (!user) {
        throw { code: 404, message: 'User not found' };
      }
      const { password, ...userWithoutPassword } = user;
      return response(200, 'User fetched', userWithoutPassword);
    },
    getCurrent: async () => {
      await delay();
      const user = mockUsers[0]; // 返回admin用户
      const { password, ...userWithoutPassword } = user;
      return response(200, 'User fetched', userWithoutPassword);
    },
    update: async (id: string, email: string) => {
      await delay();
      const user = mockUsers.find(u => u.id === id);
      if (!user) {
        throw { code: 404, message: 'User not found' };
      }
      user.email = email;
      user.updated_at = new Date().toISOString();
      const { password, ...userWithoutPassword } = user;
      return response(200, 'User updated', userWithoutPassword);
    },
    delete: async (id: string) => {
      await delay();
      const index = mockUsers.findIndex(u => u.id === id);
      if (index === -1) {
        throw { code: 404, message: 'User not found' };
      }
      mockUsers.splice(index, 1);
      return response(200, 'User deleted');
    },
    changePassword: async (oldPassword: string, newPassword: string) => {
      await delay();
      return response(200, 'Password changed');
    },
    assignRole: async (userId: string, roleId: string) => {
      await delay();
      return response(200, 'Role assigned');
    },
    removeRole: async (userId: string, roleId: string) => {
      await delay();
      return response(200, 'Role removed');
    },
    assignPermissionGroup: async (userId: string, groupId: string) => {
      await delay();
      return response(200, 'Permission group assigned');
    },
    removePermissionGroup: async (userId: string, groupId: string) => {
      await delay();
      return response(200, 'Permission group removed');
    },
  },

  // ===== 权限管理 =====
  permission: {
    createRole: async (name: string, description: string) => {
      await delay();
      const newRole = {
        id: String(mockRoles.length + 1),
        name,
        description,
        created_at: new Date().toISOString(),
      };
      mockRoles.push(newRole);
      return response(200, 'Role created', newRole);
    },
    listRoles: async (page: number = 1, pageSize: number = 10) => {
      await delay();
      const start = (page - 1) * pageSize;
      const end = start + pageSize;
      // 为每个角色附加权限列表
      const rolesWithPermissions = mockRoles.slice(start, end).map(role => ({
        ...role,
        permissions: (mockRolePermissions[role.id] || []).map(permId => 
          mockPermissions.find(p => p.id === permId)
        ).filter(Boolean),
      }));
      return response(200, 'Roles fetched', {
        items: rolesWithPermissions,
        total: mockRoles.length,
        page,
        page_size: pageSize,
      });
    },
    getRole: async (id: string) => {
      await delay();
      const role = mockRoles.find(r => r.id === id);
      if (!role) {
        throw { code: 404, message: 'Role not found' };
      }
      return response(200, 'Role fetched', role);
    },
    create: async (name: string, description: string, action: string, resource: string) => {
      await delay();
      const newPermission = {
        id: String(mockPermissions.length + 1),
        name,
        description,
        action,
        resource,
        created_at: new Date().toISOString(),
      };
      mockPermissions.push(newPermission);
      return response(200, 'Permission created', newPermission);
    },
    list: async (page: number = 1, pageSize: number = 10) => {
      await delay();
      const start = (page - 1) * pageSize;
      const end = start + pageSize;
      return response(200, 'Permissions fetched', {
        items: mockPermissions.slice(start, end),
        total: mockPermissions.length,
        page,
        page_size: pageSize,
      });
    },
    get: async (id: string) => {
      await delay();
      const permission = mockPermissions.find(p => p.id === id);
      if (!permission) {
        throw { code: 404, message: 'Permission not found' };
      }
      return response(200, 'Permission fetched', permission);
    },
    createGroup: async (name: string, description: string) => {
      await delay();
      const newGroup = {
        id: String(mockGroups.length + 1),
        name,
        description,
        created_at: new Date().toISOString(),
      };
      mockGroups.push(newGroup);
      return response(200, 'Group created', newGroup);
    },
    listGroups: async (page: number = 1, pageSize: number = 10) => {
      await delay();
      const start = (page - 1) * pageSize;
      const end = start + pageSize;
      return response(200, 'Groups fetched', {
        items: mockGroups.slice(start, end),
        total: mockGroups.length,
        page,
        page_size: pageSize,
      });
    },
    getGroup: async (id: string) => {
      await delay();
      const group = mockGroups.find(g => g.id === id);
      if (!group) {
        throw { code: 404, message: 'Group not found' };
      }
      return response(200, 'Group fetched', group);
    },
    assignPermissionToRole: async (roleId: string, permissionId: string) => {
      await delay();
      if (!mockRolePermissions[roleId]) {
        mockRolePermissions[roleId] = [];
      }
      if (!mockRolePermissions[roleId].includes(permissionId)) {
        mockRolePermissions[roleId].push(permissionId);
      }
      return response(200, 'Permission assigned to role');
    },
    removePermissionFromRole: async (roleId: string, permissionId: string) => {
      await delay();
      if (mockRolePermissions[roleId]) {
        mockRolePermissions[roleId] = mockRolePermissions[roleId].filter(id => id !== permissionId);
      }
      return response(200, 'Permission removed from role');
    },
    addResourceToGroup: async (groupId: string, resourceId: string) => {
      await delay();
      return response(200, 'Resource added to group');
    },
    removeResourceFromGroup: async (groupId: string, resourceId: string) => {
      await delay();
      return response(200, 'Resource removed from group');
    },
    getUserPermissions: async (userId: string) => {
      await delay();
      return response(200, 'Permissions fetched', mockPermissions);
    },
    checkAccess: async (userId: string, resourceId: string, action: string) => {
      await delay();
      return response(200, 'Access granted', { has_access: true });
    },
    grant: async (userId: string, resourceId: string, action: string) => {
      await delay();
      return response(200, 'Permission granted');
    },
    revoke: async (userId: string, resourceId: string, action: string) => {
      await delay();
      return response(200, 'Permission revoked');
    },
  },
};
