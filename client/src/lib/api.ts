import axios, { AxiosInstance } from 'axios';
import { mockAPI } from './mockData';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';
const USE_MOCK_DATA = import.meta.env.VITE_USE_MOCK !== 'false'; // 默认使用Mock数据

// 创建axios实例
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器 - 添加token
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器 - 处理错误
apiClient.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error.response?.data || error);
  }
);

// ===== 认证相关 =====
export const authAPI = USE_MOCK_DATA ? mockAPI.auth : {
  login: (username: string, password: string) =>
    apiClient.post('/auth/login', { username, password }),
  register: (username: string, email: string, password: string) =>
    apiClient.post('/auth/register', { username, email, password }),
  refreshToken: (token: string) =>
    apiClient.post('/auth/refresh', { token }),
};

// ===== 用户管理 =====
export const userAPI = USE_MOCK_DATA ? mockAPI.user : {
  create: (username: string, email: string, password: string) =>
    apiClient.post('/users/create', { username, email, password }),
  list: (page: number = 1, pageSize: number = 10) =>
    apiClient.get('/users/list', { params: { page, page_size: pageSize } }),
  get: (id: string) =>
    apiClient.get('/users/get', { params: { id } }),
  getCurrent: () =>
    apiClient.get('/users/me'),
  update: (id: string, email: string) =>
    apiClient.post('/users/update', { id, email }),
  delete: (id: string) =>
    apiClient.post('/users/delete', { id }),
  changePassword: (oldPassword: string, newPassword: string) =>
    apiClient.post('/users/change-password', { old_password: oldPassword, new_password: newPassword }),
  assignRole: (userId: string, roleId: string) =>
    apiClient.post('/users/assign-role', { user_id: userId, role_id: roleId }),
  removeRole: (userId: string, roleId: string) =>
    apiClient.post('/users/remove-role', { user_id: userId, role_id: roleId }),
  assignPermissionGroup: (userId: string, groupId: string) =>
    apiClient.post('/users/assign-permission-group', { user_id: userId, group_id: groupId }),
  removePermissionGroup: (userId: string, groupId: string) =>
    apiClient.post('/users/remove-permission-group', { user_id: userId, group_id: groupId }),
};

// ===== 权限管理 =====
export const permissionAPI = USE_MOCK_DATA ? mockAPI.permission : {
  // 角色
  createRole: (name: string, description: string) =>
    apiClient.post('/permissions/roles/create', { name, description }),
  listRoles: (page: number = 1, pageSize: number = 10) =>
    apiClient.get('/permissions/roles/list', { params: { page, page_size: pageSize } }),
  getRole: (id: string) =>
    apiClient.get('/permissions/roles/get', { params: { id } }),
  assignPermissionToRole: (roleId: string, permissionId: string) =>
    apiClient.post('/permissions/roles/assign-permission', { role_id: roleId, permission_id: permissionId }),
  removePermissionFromRole: (roleId: string, permissionId: string) =>
    apiClient.post('/permissions/roles/remove-permission', { role_id: roleId, permission_id: permissionId }),

  // 权限
  create: (name: string, description: string, action: string, resource: string) =>
    apiClient.post('/permissions/create', { name, description, action, resource }),
  list: (page: number = 1, pageSize: number = 10) =>
    apiClient.get('/permissions/list', { params: { page, page_size: pageSize } }),
  get: (id: string) =>
    apiClient.get('/permissions/get', { params: { id } }),

  // 权限组
  createGroup: (name: string, description: string) =>
    apiClient.post('/permissions/groups/create', { name, description }),
  listGroups: (page: number = 1, pageSize: number = 10) =>
    apiClient.get('/permissions/groups/list', { params: { page, page_size: pageSize } }),
  getGroup: (id: string) =>
    apiClient.get('/permissions/groups/get', { params: { id } }),
  addResourceToGroup: (groupId: string, resourceId: string) =>
    apiClient.post('/permissions/groups/add-resource', { group_id: groupId, resource_id: resourceId }),
  removeResourceFromGroup: (groupId: string, resourceId: string) =>
    apiClient.post('/permissions/groups/remove-resource', { group_id: groupId, resource_id: resourceId }),

  // 删除
  deleteRole: (roleId: string) =>
    apiClient.post('/permissions/roles/delete', { role_id: roleId }),
  deletePermission: (permId: string) =>
    apiClient.post('/permissions/delete', { permission_id: permId }),
  deletePermissionGroup: (groupId: string) =>
    apiClient.post('/permissions/groups/delete', { group_id: groupId }),

  // 权限组用户管理
  addUserToGroup: (groupId: string, userId: string) =>
    apiClient.post('/permissions/groups/add-user', { group_id: groupId, user_id: userId }),

  // 权限检查
  getUserPermissions: (userId: string) =>
    apiClient.get('/permissions/users/get-permissions', { params: { user_id: userId } }),
  checkAccess: (userId: string, resourceId: string, action: string) =>
    apiClient.post('/permissions/check', { user_id: userId, resource_id: resourceId, action }),
  grant: (userId: string, resourceId: string, action: string) =>
    apiClient.post('/permissions/grant', { user_id: userId, resource_id: resourceId, action }),
  revoke: (userId: string, resourceId: string, action: string) =>
    apiClient.post('/permissions/revoke', { user_id: userId, resource_id: resourceId, action }),
};

export default apiClient;
