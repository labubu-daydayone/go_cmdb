import React, { useState, useEffect, useMemo } from 'react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from '@/components/ui/drawer';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Plus, Trash2, Link, UserPlus, FolderPlus, X, Search } from 'lucide-react';
import { MultiSelect } from '@/components/ui/multi-select';
import { useWebSocket } from '@/hooks/useWebSocket';
import { permissionAPI, userAPI } from '@/lib/api';

interface Role {
  id: string;
  name: string;
  description: string;
  permissions?: string[];
}

interface Permission {
  id: string;
  name: string;
  resource: string;
  action: string;
  description: string;
}

interface PermissionGroup {
  id: string;
  name: string;
  description: string;
  users?: string[];
  resources?: Array<{ type: string; id: string }>;
}

interface User {
  id: string;
  username: string;
  email: string;
  role: 'admin' | 'user';
}

// 模拟资源数据
const getMockResources = (type: string) => {
  const resources = {
    domain: [
      { id: 'domain-1', name: 'example.com', status: '已激活' },
      { id: 'domain-2', name: 'test.com', status: '验证中' },
      { id: 'domain-3', name: 'demo.com', status: '已激活' },
    ],
    nginx: [
      { id: 'nginx-1', name: 'default.conf', path: '/etc/nginx/conf.d/default.conf' },
      { id: 'nginx-2', name: 'ssl.conf', path: '/etc/nginx/conf.d/ssl.conf' },
    ],
    script: [
      { id: 'script-1', name: 'backup.sh', description: '数据库备份脚本' },
      { id: 'script-2', name: 'deploy.sh', description: '部署脚本' },
    ],
  };
  return resources[type as keyof typeof resources] || [];
};

// 权限颜色映射
const getPermissionColor = (index: number) => {
  const colors = [
    'bg-blue-100 text-blue-800 border-blue-200',
    'bg-green-100 text-green-800 border-green-200',
    'bg-purple-100 text-purple-800 border-purple-200',
    'bg-orange-100 text-orange-800 border-orange-200',
    'bg-pink-100 text-pink-800 border-pink-200',
    'bg-cyan-100 text-cyan-800 border-cyan-200',
  ];
  return colors[index % colors.length];
};

export default function Permissions() {
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [groups, setGroups] = useState<PermissionGroup[]>([]);
  const [users, setUsers] = useState<User[]>([]);

  // 角色相关状态
  const [newRoleName, setNewRoleName] = useState('');
  const [newRoleDesc, setNewRoleDesc] = useState('');
  const [createRoleOpen, setCreateRoleOpen] = useState(false);

  // 权限相关状态
  const [newPermName, setNewPermName] = useState('');
  const [newPermResource, setNewPermResource] = useState('');
  const [newPermAction, setNewPermAction] = useState('');
  const [newPermDesc, setNewPermDesc] = useState('');
  const [createPermOpen, setCreatePermOpen] = useState(false);

  // 权限组相关状态
  const [newGroupName, setNewGroupName] = useState('');
  const [newGroupDesc, setNewGroupDesc] = useState('');
  const [createGroupOpen, setCreateGroupOpen] = useState(false);

  // 权限分配相关状态
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [assignPermOpen, setAssignPermOpen] = useState(false);
  const [permissionSearchTerm, setPermissionSearchTerm] = useState('');
  const [selectedPermissionId, setSelectedPermissionId] = useState('');

  // 权限组用户管理
  const [selectedGroup, setSelectedGroup] = useState<PermissionGroup | null>(null);
  const [addUserToGroupOpen, setAddUserToGroupOpen] = useState(false);
  const [selectedUserId, setSelectedUserId] = useState('');

  // 权限组资源管理
  const [addResourceToGroupOpen, setAddResourceToGroupOpen] = useState(false);
  const [selectedResourceType, setSelectedResourceType] = useState<string>('domain');
  const [selectedResourceId, setSelectedResourceId] = useState('');

  const token = localStorage.getItem('token');
  const { subscribe, unsubscribe } = useWebSocket(token);

  useEffect(() => {
    loadData();

    // 订阅实时更新
    subscribe('role:list');
    subscribe('permission:list');
    subscribe('group:list');

    return () => {
      unsubscribe('role:list');
      unsubscribe('permission:list');
      unsubscribe('group:list');
    };
  }, []);

  const loadData = async () => {
    try {
      const [rolesRes, permsRes, groupsRes, usersRes] = await Promise.all([
        permissionAPI.listRoles(),
        permissionAPI.list(),
        permissionAPI.listGroups(),
        userAPI.list(),
      ]);
      // 确保数据格式正确，处理分页响应和直接数组响应
      const rolesData = rolesRes.data?.items || rolesRes.data || rolesRes;
      const permsData = permsRes.data?.items || permsRes.data || permsRes;
      const groupsData = groupsRes.data?.items || groupsRes.data || groupsRes;
      const usersData = usersRes.data?.items || usersRes.data || usersRes;
      
      setRoles(Array.isArray(rolesData) ? rolesData : []);
      setPermissions(Array.isArray(permsData) ? permsData : []);
      setGroups(Array.isArray(groupsData) ? groupsData : []);
      setUsers(Array.isArray(usersData) ? usersData : []);
    } catch (error) {
      console.error('加载数据失败:', error);
      // 出错时设置为空数组，避免map错误
      setRoles([]);
      setPermissions([]);
      setGroups([]);
      setUsers([]);
    }
  };

  // 创建角色
  const handleCreateRole = async () => {
    if (!newRoleName.trim()) return;
    try {
      await permissionAPI.createRole(newRoleName, newRoleDesc);
      setNewRoleName('');
      setNewRoleDesc('');
      setCreateRoleOpen(false);
      loadData();
    } catch (error) {
      console.error('创建角色失败:', error);
    }
  };

  // 删除角色
  const handleDeleteRole = async (id: string) => {
    try {
      await permissionAPI.deleteRole(id);
      loadData();
    } catch (error) {
      console.error('删除角色失败:', error);
    }
  };

  // 创建权限
  const handleCreatePermission = async () => {
    if (!newPermName.trim() || !newPermResource.trim() || !newPermAction.trim()) return;
    try {
      await permissionAPI.create(newPermName, newPermDesc, newPermAction, newPermResource);
      setNewPermName('');
      setNewPermResource('');
      setNewPermAction('');
      setNewPermDesc('');
      setCreatePermOpen(false);
      loadData();
    } catch (error) {
      console.error('创建权限失败:', error);
    }
  };

  // 删除权限
  const handleDeletePermission = async (id: string) => {
    try {
      await permissionAPI.deletePermission(id);
      loadData();
    } catch (error) {
      console.error('删除权限失败:', error);
    }
  };

  // 创建权限组
  const handleCreateGroup = async () => {
    if (!newGroupName.trim()) return;
    try {
      await permissionAPI.createGroup(newGroupName, newGroupDesc);
      setNewGroupName('');
      setNewGroupDesc('');
      setCreateGroupOpen(false);
      loadData();
    } catch (error) {
      console.error('创建权限组失败:', error);
    }
  };

  // 删除权限组
  const handleDeleteGroup = async (id: string) => {
    try {
      await permissionAPI.deletePermissionGroup(id);
      loadData();
    } catch (error) {
      console.error('删除权限组失败:', error);
    }
  };

  // 为角色分配权限
  const handleAssignPermission = async () => {
    if (!selectedRole || !selectedPermissionId) return;
    try {
      await permissionAPI.assignPermissionToRole(selectedRole.id, selectedPermissionId);
      setSelectedPermissionId('');
      setPermissionSearchTerm('');
      loadData();
    } catch (error) {
      console.error('分配权限失败:', error);
    }
  };

  // 从角色移除权限
  const handleRemovePermission = async (roleId: string, permissionId: string) => {
    try {
      await permissionAPI.removePermissionFromRole(roleId, permissionId);
      loadData();
    } catch (error) {
      console.error('移除权限失败:', error);
    }
  };

  // 添加用户到权限组
  const handleAddUserToGroup = async () => {
    if (!selectedGroup || !selectedUserId) return;
    try {
      await permissionAPI.addUserToGroup(selectedGroup.id, selectedUserId);
      setSelectedUserId('');
      setAddUserToGroupOpen(false);
      loadData();
    } catch (error) {
      console.error('添加用户失败:', error);
    }
  };

  // 添加资源到权限组
  const handleAddResourceToGroup = async () => {
    if (!selectedGroup || !selectedResourceType || !selectedResourceId) return;
    try {
      await permissionAPI.addResourceToGroup(selectedGroup.id, selectedResourceId);
      setSelectedResourceType('domain');
      setSelectedResourceId('');
      setAddResourceToGroupOpen(false);
      loadData();
    } catch (error) {
      console.error('添加资源失败:', error);
    }
  };

  // 过滤权限列表（模糊搜索）
  const filteredPermissions = useMemo(() => {
    if (!permissionSearchTerm) return permissions;
    const term = permissionSearchTerm.toLowerCase();
    return permissions.filter(
      (p) =>
        p.name.toLowerCase().includes(term) ||
        p.resource.toLowerCase().includes(term) ||
        p.action.toLowerCase().includes(term) ||
        p.description.toLowerCase().includes(term)
    );
  }, [permissions, permissionSearchTerm]);

  // 获取角色已分配的权限
  const getRolePermissions = (role: Role) => {
    if (!role.permissions) return [];
    return permissions.filter((p) => role.permissions?.includes(p.id));
  };

  // 获取未分配的权限
  const getUnassignedPermissions = (role: Role) => {
    if (!role.permissions) return permissions;
    return permissions.filter((p) => !role.permissions?.includes(p.id));
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">权限管理</h1>
        <p className="text-muted-foreground mt-2">管理系统的角色、权限和权限组</p>
      </div>

      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="overview">概览</TabsTrigger>
          <TabsTrigger value="roles">角色</TabsTrigger>
          <TabsTrigger value="permissions">权限</TabsTrigger>
          <TabsTrigger value="groups">权限组</TabsTrigger>
        </TabsList>

        {/* 概览标签页 */}
        <TabsContent value="overview" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-3">
            <Card>
              <CardHeader>
                <CardTitle>角色统计</CardTitle>
                <CardDescription>系统中的角色数量</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="text-3xl font-bold">{roles.length}</div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>权限统计</CardTitle>
                <CardDescription>系统中的权限数量</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="text-3xl font-bold">{permissions.length}</div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>权限组统计</CardTitle>
                <CardDescription>系统中的权限组数量</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="text-3xl font-bold">{groups.length}</div>
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>角色权限关系</CardTitle>
              <CardDescription>查看每个角色拥有的权限</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {roles.map((role) => (
                  <div key={role.id} className="border rounded-lg p-4">
                    <div className="flex items-center justify-between mb-2">
                      <div>
                        <h3 className="font-semibold">{role.name}</h3>
                        <p className="text-sm text-muted-foreground">{role.description}</p>
                      </div>
                      <Badge variant="outline">{role.permissions?.length || 0} 个权限</Badge>
                    </div>
                    <div className="flex flex-wrap gap-2 mt-3">
                      {getRolePermissions(role).map((perm) => (
                        <Badge key={perm.id} variant="secondary">
                          {perm.name}
                        </Badge>
                      ))}
                      {(!role.permissions || role.permissions.length === 0) && (
                        <span className="text-sm text-muted-foreground">暂无权限</span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>权限组详情</CardTitle>
              <CardDescription>查看每个权限组的用户和资源</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {groups.map((group) => (
                  <div key={group.id} className="border rounded-lg p-4">
                    <div className="flex items-center justify-between mb-2">
                      <div>
                        <h3 className="font-semibold">{group.name}</h3>
                        <p className="text-sm text-muted-foreground">{group.description}</p>
                      </div>
                      <div className="flex gap-2">
                        <Badge variant="outline">{group.users?.length || 0} 个用户</Badge>
                        <Badge variant="outline">{group.resources?.length || 0} 个资源</Badge>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 角色标签页 */}
        <TabsContent value="roles" className="space-y-4">
          <div className="flex justify-between items-center">
            <h2 className="text-xl font-semibold">角色列表</h2>
            <Drawer open={createRoleOpen} onOpenChange={setCreateRoleOpen}>
              <DrawerTrigger asChild>
                <Button>
                  <Plus className="mr-2 h-4 w-4" />
                  新建角色
                </Button>
              </DrawerTrigger>
              <DrawerContent className="h-[80vh]">
                <DrawerHeader>
                  <DrawerTitle>创建新角色</DrawerTitle>
                  <DrawerDescription>填写角色信息</DrawerDescription>
                </DrawerHeader>
                <div className="px-4 space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="role-name">角色名称</Label>
                    <Input
                      id="role-name"
                      value={newRoleName}
                      onChange={(e) => setNewRoleName(e.target.value)}
                      placeholder="例如：运维人员"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="role-desc">角色描述</Label>
                    <Input
                      id="role-desc"
                      value={newRoleDesc}
                      onChange={(e) => setNewRoleDesc(e.target.value)}
                      placeholder="例如：负责系统运维工作"
                    />
                  </div>
                </div>
                <DrawerFooter>
                  <Button onClick={handleCreateRole}>创建</Button>
                  <DrawerClose asChild>
                    <Button variant="outline">取消</Button>
                  </DrawerClose>
                </DrawerFooter>
              </DrawerContent>
            </Drawer>
          </div>

          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {roles.map((role) => (
              <Card key={role.id}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-lg">{role.name}</CardTitle>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleDeleteRole(role.id)}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                  <CardDescription>{role.description}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    <div>
                      <div className="text-sm font-medium mb-2">已分配权限</div>
                      <div className="flex flex-wrap gap-2 min-h-[40px] p-2 border rounded-lg bg-muted/30">
                        {getRolePermissions(role).map((perm, index) => (
                          <Badge
                            key={perm.id}
                            className={`${getPermissionColor(index)} cursor-pointer hover:opacity-80`}
                            onClick={() => handleRemovePermission(role.id, perm.id)}
                          >
                            {perm.name}
                            <X className="ml-1 h-3 w-3" />
                          </Badge>
                        ))}
                        {(!role.permissions || role.permissions.length === 0) && (
                          <span className="text-sm text-muted-foreground">暂无权限</span>
                        )}
                      </div>
                    </div>

                    <Drawer
                      open={assignPermOpen && selectedRole?.id === role.id}
                      onOpenChange={(open) => {
                        setAssignPermOpen(open);
                        if (open) {
                          setSelectedRole(role);
                          setPermissionSearchTerm('');
                          setSelectedPermissionId('');
                        }
                      }}
                    >
                      <DrawerTrigger asChild>
                        <Button variant="outline" size="sm" className="w-full">
                          <Link className="mr-2 h-4 w-4" />
                          分配权限
                        </Button>
                      </DrawerTrigger>
                      <DrawerContent className="h-[80vh]">
                        <DrawerHeader>
                          <DrawerTitle>为 {role.name} 分配权限</DrawerTitle>
                          <DrawerDescription>使用搜索框查找并选择权限</DrawerDescription>
                        </DrawerHeader>
                        <div className="px-4 space-y-4 flex-1 overflow-y-auto">
                          <div className="space-y-2">
                            <Label>选择权限</Label>
                            <MultiSelect
                              options={permissions.map((p) => ({
                                id: p.id,
                                name: p.name,
                                description: `${p.resource} - ${p.action}`,
                              }))}
                              selected={role.permissions || []}
                              onChange={async (selectedIds) => {
                                // 找出新增的权限
                                const added = selectedIds.filter(
                                  (id) => !role.permissions?.includes(id)
                                );
                                // 找出删除的权限
                                const removed = (role.permissions || []).filter(
                                  (id) => !selectedIds.includes(id)
                                );

                                try {
                                  // 执行添加操作
                                  const addPromises = added.map((id) =>
                                    permissionAPI.assignPermissionToRole(role.id, id)
                                  );

                                  // 执行删除操作
                                  const removePromises = removed.map((id) =>
                                    permissionAPI.removePermissionFromRole(role.id, id)
                                  );

                                  // 等待所有操作完成
                                  await Promise.all([...addPromises, ...removePromises]);

                                  // 重新加载数据
                                  await loadData();
                                } catch (error) {
                                  console.error('权限分配失败:', error);
                                }
                              }}
                              placeholder="点击选择权限..."
                              searchPlaceholder="搜索权限名称、资源或操作..."
                              getOptionColor={getPermissionColor}
                            />
                          </div>
                        </div>
                        <DrawerFooter>
                          <DrawerClose asChild>
                            <Button variant="outline">关闭</Button>
                          </DrawerClose>
                        </DrawerFooter>
                      </DrawerContent>
                    </Drawer>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* 权限标签页 */}
        <TabsContent value="permissions" className="space-y-4">
          <div className="flex justify-between items-center">
            <h2 className="text-xl font-semibold">权限列表</h2>
            <Drawer open={createPermOpen} onOpenChange={setCreatePermOpen}>
              <DrawerTrigger asChild>
                <Button>
                  <Plus className="mr-2 h-4 w-4" />
                  新建权限
                </Button>
              </DrawerTrigger>
              <DrawerContent className="h-[80vh]">
                <DrawerHeader>
                  <DrawerTitle>创建新权限</DrawerTitle>
                  <DrawerDescription>填写权限信息</DrawerDescription>
                </DrawerHeader>
                <div className="px-4 space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="perm-name">权限名称</Label>
                    <Input
                      id="perm-name"
                      value={newPermName}
                      onChange={(e) => setNewPermName(e.target.value)}
                      placeholder="例如：用户列表-读取"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="perm-resource">资源类型</Label>
                    <Select value={newPermResource} onValueChange={setNewPermResource}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择资源类型" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="user">用户管理</SelectItem>
                        <SelectItem value="permission">权限管理</SelectItem>
                        <SelectItem value="domain">域名管理</SelectItem>
                        <SelectItem value="nginx">Nginx配置</SelectItem>
                        <SelectItem value="script">脚本管理</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="perm-action">操作类型</Label>
                    <Select value={newPermAction} onValueChange={setNewPermAction}>
                      <SelectTrigger>
                        <SelectValue placeholder="选择操作类型" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="read">读取</SelectItem>
                        <SelectItem value="create">创建</SelectItem>
                        <SelectItem value="update">更新</SelectItem>
                        <SelectItem value="delete">删除</SelectItem>
                        <SelectItem value="execute">执行</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="perm-desc">权限描述</Label>
                    <Input
                      id="perm-desc"
                      value={newPermDesc}
                      onChange={(e) => setNewPermDesc(e.target.value)}
                      placeholder="例如：允许查看用户列表"
                    />
                  </div>
                </div>
                <DrawerFooter>
                  <Button onClick={handleCreatePermission}>创建</Button>
                  <DrawerClose asChild>
                    <Button variant="outline">取消</Button>
                  </DrawerClose>
                </DrawerFooter>
              </DrawerContent>
            </Drawer>
          </div>

          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {permissions.map((perm) => (
              <Card key={perm.id}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-lg">{perm.name}</CardTitle>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleDeletePermission(perm.id)}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                  <CardDescription>{perm.description}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">资源</span>
                      <Badge variant="outline">{perm.resource}</Badge>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">操作</span>
                      <Badge variant="outline">{perm.action}</Badge>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* 权限组标签页 */}
        <TabsContent value="groups" className="space-y-4">
          <div className="flex justify-between items-center">
            <h2 className="text-xl font-semibold">权限组列表</h2>
            <Drawer open={createGroupOpen} onOpenChange={setCreateGroupOpen}>
              <DrawerTrigger asChild>
                <Button>
                  <Plus className="mr-2 h-4 w-4" />
                  新建权限组
                </Button>
              </DrawerTrigger>
              <DrawerContent className="h-[80vh]">
                <DrawerHeader>
                  <DrawerTitle>创建新权限组</DrawerTitle>
                  <DrawerDescription>填写权限组信息</DrawerDescription>
                </DrawerHeader>
                <div className="px-4 space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="group-name">权限组名称</Label>
                    <Input
                      id="group-name"
                      value={newGroupName}
                      onChange={(e) => setNewGroupName(e.target.value)}
                      placeholder="例如：运维团队"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="group-desc">权限组描述</Label>
                    <Input
                      id="group-desc"
                      value={newGroupDesc}
                      onChange={(e) => setNewGroupDesc(e.target.value)}
                      placeholder="例如：负责系统运维的团队"
                    />
                  </div>
                </div>
                <DrawerFooter>
                  <Button onClick={handleCreateGroup}>创建</Button>
                  <DrawerClose asChild>
                    <Button variant="outline">取消</Button>
                  </DrawerClose>
                </DrawerFooter>
              </DrawerContent>
            </Drawer>
          </div>

          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {groups.map((group) => (
              <Card key={group.id}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-lg">{group.name}</CardTitle>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleDeleteGroup(group.id)}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                  <CardDescription>{group.description}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">用户数</span>
                      <Badge variant="outline">{group.users?.length || 0}</Badge>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">资源数</span>
                      <Badge variant="outline">{group.resources?.length || 0}</Badge>
                    </div>

                    <div className="flex gap-2 pt-2">
                      <Drawer open={addUserToGroupOpen && selectedGroup?.id === group.id} onOpenChange={(open) => {
                        setAddUserToGroupOpen(open);
                        if (open) setSelectedGroup(group);
                      }}>
                        <DrawerTrigger asChild>
                          <Button variant="outline" size="sm" className="flex-1">
                            <UserPlus className="mr-2 h-4 w-4" />
                            添加用户
                          </Button>
                        </DrawerTrigger>
                        <DrawerContent className="h-[80vh]">
                          <DrawerHeader>
                            <DrawerTitle>添加用户到 {group.name}</DrawerTitle>
                            <DrawerDescription>选择要添加的用户</DrawerDescription>
                          </DrawerHeader>
                          <div className="px-4 space-y-4 flex-1 overflow-y-auto">
                            <div className="space-y-2">
                              <Label>选择用户</Label>
                              <div className="space-y-2">
                                {users.filter(u => !group.users?.includes(u.id)).map((user) => (
                                  <div
                                    key={user.id}
                                    className={`p-3 border rounded-lg cursor-pointer transition-colors ${
                                      selectedUserId === user.id
                                        ? 'border-blue-500 bg-blue-50'
                                        : 'border-gray-200 hover:border-blue-300'
                                    }`}
                                    onClick={() => setSelectedUserId(user.id)}
                                  >
                                    <div className="flex items-center justify-between">
                                      <div>
                                        <div className="font-medium">{user.username}</div>
                                        <div className="text-sm text-muted-foreground">{user.email}</div>
                                      </div>
                                      {user.role === 'admin' && (
                                        <Badge variant="secondary">管理员</Badge>
                                      )}
                                    </div>
                                  </div>
                                ))}
                                {users.filter(u => !group.users?.includes(u.id)).length === 0 && (
                                  <div className="text-center text-muted-foreground py-4">
                                    所有用户已添加到该权限组
                                  </div>
                                )}
                              </div>
                            </div>
                          </div>
                          <DrawerFooter>
                            <Button onClick={handleAddUserToGroup} disabled={!selectedUserId}>
                              添加用户
                            </Button>
                            <DrawerClose asChild>
                              <Button variant="outline">取消</Button>
                            </DrawerClose>
                          </DrawerFooter>
                        </DrawerContent>
                      </Drawer>

                      <Drawer open={addResourceToGroupOpen && selectedGroup?.id === group.id} onOpenChange={(open) => {
                        setAddResourceToGroupOpen(open);
                        if (open) setSelectedGroup(group);
                      }}>
                        <DrawerTrigger asChild>
                          <Button variant="outline" size="sm" className="flex-1">
                            <FolderPlus className="mr-2 h-4 w-4" />
                            添加资源
                          </Button>
                        </DrawerTrigger>
                        <DrawerContent className="h-[80vh]">
                          <DrawerHeader>
                            <DrawerTitle>添加资源到 {group.name}</DrawerTitle>
                            <DrawerDescription>选择资源类型和ID</DrawerDescription>
                          </DrawerHeader>
                          <div className="px-4 space-y-4 flex-1 overflow-y-auto">
                            <div className="space-y-2">
                              <Label>资源类型</Label>
                              <Select value={selectedResourceType} onValueChange={setSelectedResourceType}>
                                <SelectTrigger>
                                  <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectItem value="domain">域名</SelectItem>
                                  <SelectItem value="nginx">Nginx配置</SelectItem>
                                  <SelectItem value="script">脚本</SelectItem>
                                </SelectContent>
                              </Select>
                            </div>

                            <div className="space-y-2">
                              <Label>选择资源</Label>
                              <div className="space-y-2">
                                {getMockResources(selectedResourceType).map((resource: any) => (
                                  <div
                                    key={resource.id}
                                    className={`p-3 border rounded-lg cursor-pointer transition-colors ${
                                      selectedResourceId === resource.id
                                        ? 'border-blue-500 bg-blue-50'
                                        : 'border-gray-200 hover:border-blue-300'
                                    }`}
                                    onClick={() => setSelectedResourceId(resource.id)}
                                  >
                                    <div className="font-medium">{resource.name}</div>
                                    {resource.status && (
                                      <div className="text-sm text-muted-foreground">状态: {resource.status}</div>
                                    )}
                                    {resource.path && (
                                      <div className="text-sm text-muted-foreground">{resource.path}</div>
                                    )}
                                    {resource.description && (
                                      <div className="text-sm text-muted-foreground">{resource.description}</div>
                                    )}
                                  </div>
                                ))}
                              </div>
                            </div>
                          </div>
                          <DrawerFooter>
                            <Button onClick={handleAddResourceToGroup} disabled={!selectedResourceId}>
                              添加资源
                            </Button>
                            <DrawerClose asChild>
                              <Button variant="outline">取消</Button>
                            </DrawerClose>
                          </DrawerFooter>
                        </DrawerContent>
                      </Drawer>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
