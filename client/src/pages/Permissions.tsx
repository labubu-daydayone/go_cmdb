import { useState, useEffect } from 'react';
import { permissionAPI } from '../lib/api';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card';
import { Badge } from '../components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { Label } from '../components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';
import { Checkbox } from '../components/ui/checkbox';
import { useWebSocket } from '../hooks/useWebSocket';
import { toast } from 'sonner';
import { Shield, Users, Key, FolderKey, Plus, Trash2, Link, X, UserPlus, FolderPlus } from 'lucide-react';
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from '../components/ui/drawer';

interface Role {
  id: string;
  name: string;
  description: string;
  permissions?: Permission[];
}

interface Permission {
  id: string;
  name: string;
  description: string;
  action: string;
  resource: string;
}

interface PermissionGroup {
  id: string;
  name: string;
  description: string;
  users?: any[];
  resources?: any[];
}

interface User {
  id: string;
  username: string;
  email: string;
  is_admin: boolean;
}

export default function Permissions() {
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [groups, setGroups] = useState<PermissionGroup[]>([]);
  const [users, setUsers] = useState<User[]>([]);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [selectedGroup, setSelectedGroup] = useState<PermissionGroup | null>(null);
  const [loading, setLoading] = useState(false);

  // 新建抽屉状态
  const [newRoleOpen, setNewRoleOpen] = useState(false);
  const [newPermissionOpen, setNewPermissionOpen] = useState(false);
  const [newGroupOpen, setNewGroupOpen] = useState(false);
  const [assignPermOpen, setAssignPermOpen] = useState(false);
  const [addUserToGroupOpen, setAddUserToGroupOpen] = useState(false);
  const [addResourceToGroupOpen, setAddResourceToGroupOpen] = useState(false);

  // 表单状态
  const [roleName, setRoleName] = useState('');
  const [roleDesc, setRoleDesc] = useState('');
  const [permName, setPermName] = useState('');
  const [permDesc, setPermDesc] = useState('');
  const [permAction, setPermAction] = useState('read');
  const [permResource, setPermResource] = useState('user');
  const [groupName, setGroupName] = useState('');
  const [groupDesc, setGroupDesc] = useState('');
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>([]);
  const [selectedUserId, setSelectedUserId] = useState('');
  const [resourceType, setResourceType] = useState('domain');
  const [resourceId, setResourceId] = useState('');

  // WebSocket
  const token = localStorage.getItem('token');
  const { subscribe, unsubscribe } = useWebSocket(token, {
    onMessage: (message) => {
      if (message.type === 'list_update' || message.type === 'resource_update') {
        loadData();
      }
    },
  });

  useEffect(() => {
    loadData();
    
    // 订阅实时更新
    subscribe('permission:list');
    
    return () => {
      unsubscribe('permission:list');
    };
  }, []);

  // 模拟资源数据
  const getMockResources = (type: string) => {
    switch (type) {
      case 'domain':
        return [
          { id: '1', name: 'example.com', description: '主域名', status: 'active' },
          { id: '2', name: 'test.com', description: '测试域名', status: 'pending' },
          { id: '3', name: 'demo.com', description: '演示域名', status: 'active' },
        ];
      case 'nginx':
        return [
          { id: '1', name: 'default.conf', description: '默认配置', status: 'active' },
          { id: '2', name: 'ssl.conf', description: 'SSL配置', status: 'active' },
        ];
      case 'script':
        return [
          { id: '1', name: 'backup.sh', description: '备份脚本', status: 'active' },
          { id: '2', name: 'deploy.sh', description: '部署脚本', status: 'active' },
        ];
      default:
        return [];
    }
  };

  const loadData = async () => {
    setLoading(true);
    try {
      const { userAPI } = await import('../lib/api');
      const [rolesData, permsData, groupsData, usersData] = await Promise.all([
        permissionAPI.listRoles(),
        permissionAPI.list(),
        permissionAPI.listGroups(),
        userAPI.list(),
      ]);
      setRoles(rolesData.data?.items || []);
      setPermissions(permsData.data?.items || []);
      setGroups(groupsData.data?.items || []);
      setUsers(usersData.data?.items || []);
    } catch (error) {
      toast.error('加载数据失败');
    } finally {
      setLoading(false);
    }
  };

  // 创建角色
  const handleCreateRole = async () => {
    if (!roleName.trim()) {
      toast.error('请输入角色名称');
      return;
    }
    try {
      await permissionAPI.createRole(roleName, roleDesc);
      toast.success('角色创建成功');
      setNewRoleOpen(false);
      setRoleName('');
      setRoleDesc('');
      loadData();
    } catch (error) {
      toast.error('创建失败');
    }
  };

  // 创建权限
  const handleCreatePermission = async () => {
    if (!permName.trim()) {
      toast.error('请输入权限名称');
      return;
    }
    try {
      await permissionAPI.create(permName, permDesc, permAction, permResource);
      toast.success('权限创建成功');
      setNewPermissionOpen(false);
      setPermName('');
      setPermDesc('');
      setPermAction('read');
      setPermResource('user');
      loadData();
    } catch (error) {
      toast.error('创建失败');
    }
  };

  // 创建权限组
  const handleCreateGroup = async () => {
    if (!groupName.trim()) {
      toast.error('请输入权限组名称');
      return;
    }
    try {
      await permissionAPI.createGroup(groupName, groupDesc);
      toast.success('权限组创建成功');
      setNewGroupOpen(false);
      setGroupName('');
      setGroupDesc('');
      loadData();
    } catch (error) {
      toast.error('创建失败');
    }
  };

  // 分配权限到角色
  const handleAssignPermissions = async () => {
    if (!selectedRole || selectedPermissions.length === 0) {
      toast.error('请选择要分配的权限');
      return;
    }
    try {
      for (const permId of selectedPermissions) {
        await permissionAPI.assignPermissionToRole(selectedRole.id, permId);
      }
      toast.success(`成功分配 ${selectedPermissions.length} 个权限`);
      setAssignPermOpen(false);
      setSelectedPermissions([]);
      loadData();
    } catch (error) {
      toast.error('分配失败');
    }
  };

  // 添加用户到权限组
  const handleAddUserToGroup = async () => {
    if (!selectedGroup || !selectedUserId) {
      toast.error('请选择用户');
      return;
    }
    try {
      await permissionAPI.addUserToGroup(selectedGroup.id, selectedUserId);
      toast.success('用户添加成功');
      setAddUserToGroupOpen(false);
      setSelectedUserId('');
      loadData();
    } catch (error) {
      toast.error('添加失败');
    }
  };

  // 添加资源到权限组
  const handleAddResourceToGroup = async () => {
    if (!selectedGroup || !resourceId) {
      toast.error('请输入资源ID');
      return;
    }
    try {
      await permissionAPI.addResourceToGroup(selectedGroup.id, `${resourceType}:${resourceId}`);
      toast.success('资源添加成功');
      setAddResourceToGroupOpen(false);
      setResourceId('');
      loadData();
    } catch (error) {
      toast.error('添加失败');
    }
  };

  // 删除角色
  const handleDeleteRole = async (roleId: string) => {
    if (!confirm('确定要删除这个角色吗？')) return;
    try {
      await permissionAPI.deleteRole(roleId);
      toast.success('角色删除成功');
      loadData();
    } catch (error) {
      toast.error('删除失败');
    }
  };

  // 删除权限
  const handleDeletePermission = async (permId: string) => {
    if (!confirm('确定要删除这个权限吗？')) return;
    try {
      await permissionAPI.deletePermission(permId);
      toast.success('权限删除成功');
      loadData();
    } catch (error) {
      toast.error('删除失败');
    }
  };

  // 删除权限组
  const handleDeleteGroup = async (groupId: string) => {
    if (!confirm('确定要删除这个权限组吗？')) return;
    try {
      await permissionAPI.deletePermissionGroup(groupId);
      toast.success('权限组删除成功');
      loadData();
    } catch (error) {
      toast.error('删除失败');
    }
  };

  // 切换权限选择
  const togglePermission = (permId: string) => {
    setSelectedPermissions(prev =>
      prev.includes(permId) ? prev.filter(id => id !== permId) : [...prev, permId]
    );
  };

  return (
    <div className="container mx-auto py-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">权限管理</h1>
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
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">角色总数</CardTitle>
                <Users className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{roles.length}</div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">权限总数</CardTitle>
                <Key className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{permissions.length}</div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">权限组总数</CardTitle>
                <FolderKey className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{groups.length}</div>
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>角色权限关联</CardTitle>
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
                      <Badge>{role.permissions?.length || 0} 个权限</Badge>
                    </div>
                    <div className="flex flex-wrap gap-2 mt-3">
                      {role.permissions?.map((perm) => (
                        <Badge key={perm.id} variant="secondary">
                          {perm.name}
                        </Badge>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>权限组详情</CardTitle>
              <CardDescription>查看权限组的用户和资源</CardDescription>
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
            <Drawer open={newRoleOpen} onOpenChange={setNewRoleOpen}>
              <DrawerTrigger asChild>
                <Button>
                  <Plus className="mr-2 h-4 w-4" />
                  新建角色
                </Button>
              </DrawerTrigger>
              <DrawerContent>
                <DrawerHeader>
                  <DrawerTitle>创建新角色</DrawerTitle>
                  <DrawerDescription>填写角色信息</DrawerDescription>
                </DrawerHeader>
                <div className="px-4 space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="role-name">角色名称</Label>
                    <Input
                      id="role-name"
                      value={roleName}
                      onChange={(e) => setRoleName(e.target.value)}
                      placeholder="例如：管理员"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="role-desc">描述</Label>
                    <Input
                      id="role-desc"
                      value={roleDesc}
                      onChange={(e) => setRoleDesc(e.target.value)}
                      placeholder="角色描述"
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
                  <div className="space-y-2">
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-muted-foreground">权限数量</span>
                      <Badge>{role.permissions?.length || 0}</Badge>
                    </div>
                    <Drawer open={assignPermOpen && selectedRole?.id === role.id} onOpenChange={(open) => {
                      setAssignPermOpen(open);
                      if (open) setSelectedRole(role);
                    }}>
                      <DrawerTrigger asChild>
                        <Button variant="outline" size="sm" className="w-full">
                          <Link className="mr-2 h-4 w-4" />
                          分配权限
                        </Button>
                      </DrawerTrigger>
                      <DrawerContent>
                        <DrawerHeader>
                          <DrawerTitle>为 {role.name} 分配权限</DrawerTitle>
                          <DrawerDescription>选择要分配的权限（可多选）</DrawerDescription>
                        </DrawerHeader>
                        <div className="px-4 max-h-[400px] overflow-y-auto">
                          <div className="space-y-2">
                            {permissions.map((perm) => (
                              <div key={perm.id} className="flex items-center space-x-2 p-2 border rounded">
                                <Checkbox
                                  id={`perm-${perm.id}`}
                                  checked={selectedPermissions.includes(perm.id)}
                                  onCheckedChange={() => togglePermission(perm.id)}
                                />
                                <label
                                  htmlFor={`perm-${perm.id}`}
                                  className="flex-1 cursor-pointer"
                                >
                                  <div className="font-medium">{perm.name}</div>
                                  <div className="text-sm text-muted-foreground">{perm.description}</div>
                                </label>
                              </div>
                            ))}
                          </div>
                        </div>
                        <DrawerFooter>
                          <Button onClick={handleAssignPermissions}>
                            分配 ({selectedPermissions.length})
                          </Button>
                          <DrawerClose asChild>
                            <Button variant="outline">取消</Button>
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
            <Drawer open={newPermissionOpen} onOpenChange={setNewPermissionOpen}>
              <DrawerTrigger asChild>
                <Button>
                  <Plus className="mr-2 h-4 w-4" />
                  新建权限
                </Button>
              </DrawerTrigger>
              <DrawerContent>
                <DrawerHeader>
                  <DrawerTitle>创建新权限</DrawerTitle>
                  <DrawerDescription>填写权限信息</DrawerDescription>
                </DrawerHeader>
                <div className="px-4 space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="perm-name">权限名称</Label>
                    <Input
                      id="perm-name"
                      value={permName}
                      onChange={(e) => setPermName(e.target.value)}
                      placeholder="例如：用户管理-读取"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="perm-desc">描述</Label>
                    <Input
                      id="perm-desc"
                      value={permDesc}
                      onChange={(e) => setPermDesc(e.target.value)}
                      placeholder="权限描述"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="perm-action">操作</Label>
                    <Select value={permAction} onValueChange={setPermAction}>
                      <SelectTrigger id="perm-action">
                        <SelectValue />
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
                    <Label htmlFor="perm-resource">资源</Label>
                    <Select value={permResource} onValueChange={setPermResource}>
                      <SelectTrigger id="perm-resource">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="user">用户</SelectItem>
                        <SelectItem value="permission">权限</SelectItem>
                        <SelectItem value="domain">域名</SelectItem>
                        <SelectItem value="nginx">Nginx</SelectItem>
                        <SelectItem value="script">脚本</SelectItem>
                      </SelectContent>
                    </Select>
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
                    <CardTitle className="text-base">{perm.name}</CardTitle>
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
                  <div className="flex gap-2">
                    <Badge variant="secondary">{perm.action}</Badge>
                    <Badge variant="outline">{perm.resource}</Badge>
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
            <Drawer open={newGroupOpen} onOpenChange={setNewGroupOpen}>
              <DrawerTrigger asChild>
                <Button>
                  <Plus className="mr-2 h-4 w-4" />
                  新建权限组
                </Button>
              </DrawerTrigger>
              <DrawerContent>
                <DrawerHeader>
                  <DrawerTitle>创建新权限组</DrawerTitle>
                  <DrawerDescription>填写权限组信息</DrawerDescription>
                </DrawerHeader>
                <div className="px-4 space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="group-name">权限组名称</Label>
                    <Input
                      id="group-name"
                      value={groupName}
                      onChange={(e) => setGroupName(e.target.value)}
                      placeholder="例如：项目A团队"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="group-desc">描述</Label>
                    <Input
                      id="group-desc"
                      value={groupDesc}
                      onChange={(e) => setGroupDesc(e.target.value)}
                      placeholder="权限组描述"
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
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">用户数量</span>
                      <Badge variant="outline">{group.users?.length || 0}</Badge>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">资源数量</span>
                      <Badge variant="outline">{group.resources?.length || 0}</Badge>
                    </div>
                    <div className="flex gap-2 mt-3">
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
                        <DrawerContent>
                          <DrawerHeader>
                            <DrawerTitle>添加用户到 {group.name}</DrawerTitle>
                            <DrawerDescription>选择要添加的用户</DrawerDescription>
                          </DrawerHeader>
                          <div className="px-4 space-y-4 max-h-[60vh] overflow-y-auto">
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
                                        <p className="font-medium">{user.username}</p>
                                        <p className="text-sm text-gray-500">{user.email}</p>
                                      </div>
                                      {user.is_admin && (
                                        <Badge variant="secondary">管理员</Badge>
                                      )}
                                    </div>
                                  </div>
                                ))}
                                {users.filter(u => !group.users?.includes(u.id)).length === 0 && (
                                  <p className="text-sm text-gray-500 text-center py-4">所有用户已在该组中</p>
                                )}
                              </div>
                            </div>
                          </div>
                          <DrawerFooter>
                            <Button onClick={handleAddUserToGroup}>添加</Button>
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
                        <DrawerContent>
                          <DrawerHeader>
                            <DrawerTitle>添加资源到 {group.name}</DrawerTitle>
                            <DrawerDescription>选择资源类型和ID</DrawerDescription>
                          </DrawerHeader>
                          <div className="px-4 space-y-4 max-h-[60vh] overflow-y-auto">
                            <div className="space-y-2">
                              <Label htmlFor="resource-type">资源类型</Label>
                              <Select value={resourceType} onValueChange={(value) => {
                                setResourceType(value);
                                setResourceId(''); // 清空选中的资源
                              }}>
                                <SelectTrigger id="resource-type">
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
                              <Label>选择{resourceType === 'domain' ? '域名' : resourceType === 'nginx' ? 'Nginx配置' : '脚本'}</Label>
                              <div className="space-y-2">
                                {/* 模拟资源列表 - 实际应该从 API 加载 */}
                                {getMockResources(resourceType).map((resource: any) => (
                                  <div
                                    key={resource.id}
                                    className={`p-3 border rounded-lg cursor-pointer transition-colors ${
                                      resourceId === resource.id
                                        ? 'border-blue-500 bg-blue-50'
                                        : 'border-gray-200 hover:border-blue-300'
                                    }`}
                                    onClick={() => setResourceId(resource.id)}
                                  >
                                    <div className="flex items-center justify-between">
                                      <div>
                                        <p className="font-medium">{resource.name}</p>
                                        <p className="text-sm text-gray-500">{resource.description || resource.id}</p>
                                      </div>
                                      {resource.status && (
                                        <Badge variant={resource.status === 'active' ? 'default' : 'secondary'}>
                                          {resource.status}
                                        </Badge>
                                      )}
                                    </div>
                                  </div>
                                ))}
                                {getMockResources(resourceType).length === 0 && (
                                  <p className="text-sm text-gray-500 text-center py-4">暂无{resourceType === 'domain' ? '域名' : resourceType === 'nginx' ? 'Nginx配置' : '脚本'}</p>
                                )}
                              </div>
                            </div>
                          </div>
                          <DrawerFooter>
                            <Button onClick={handleAddResourceToGroup}>添加</Button>
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
