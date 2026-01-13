import { useState, useEffect } from 'react';
import { permissionAPI } from '../lib/api';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card';
import { Badge } from '../components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '../components/ui/dialog';
import { Label } from '../components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../components/ui/select';
import { useWebSocket } from '../hooks/useWebSocket';
import { toast } from 'sonner';
import { Shield, Users, Key, FolderKey, Plus, Trash2, Link } from 'lucide-react';

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

export default function Permissions() {
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [groups, setGroups] = useState<PermissionGroup[]>([]);
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [selectedGroup, setSelectedGroup] = useState<PermissionGroup | null>(null);
  const [loading, setLoading] = useState(false);

  // 新建对话框状态
  const [newRoleOpen, setNewRoleOpen] = useState(false);
  const [newPermissionOpen, setNewPermissionOpen] = useState(false);
  const [newGroupOpen, setNewGroupOpen] = useState(false);
  const [assignPermissionOpen, setAssignPermissionOpen] = useState(false);

  // 表单数据
  const [roleName, setRoleName] = useState('');
  const [roleDesc, setRoleDesc] = useState('');
  const [permName, setPermName] = useState('');
  const [permDesc, setPermDesc] = useState('');
  const [permAction, setPermAction] = useState('read');
  const [permResource, setPermResource] = useState('*');
  const [groupName, setGroupName] = useState('');
  const [groupDesc, setGroupDesc] = useState('');
  const [selectedPermissionId, setSelectedPermissionId] = useState('');

  const token = localStorage.getItem('token');

  // WebSocket连接
  useWebSocket(token, {
    onMessage: (message) => {
      if (message.resource === 'role' || message.resource === 'permission' || message.resource === 'group') {
        loadData();
      }
    },
  });

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [rolesRes, permsRes, groupsRes] = await Promise.all([
        permissionAPI.listRoles(),
        permissionAPI.list(),
        permissionAPI.listGroups(),
      ]);
      setRoles(rolesRes.data.items || []);
      setPermissions(permsRes.data.items || []);
      setGroups(groupsRes.data.items || []);
    } catch (error: any) {
      toast.error(error.message || '加载失败');
    } finally {
      setLoading(false);
    }
  };

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
    } catch (error: any) {
      toast.error(error.message || '创建失败');
    }
  };

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
      loadData();
    } catch (error: any) {
      toast.error(error.message || '创建失败');
    }
  };

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
    } catch (error: any) {
      toast.error(error.message || '创建失败');
    }
  };

  const handleAssignPermission = async () => {
    if (!selectedRole || !selectedPermissionId) {
      toast.error('请选择权限');
      return;
    }
    try {
      await permissionAPI.assignPermissionToRole(selectedRole.id, selectedPermissionId);
      toast.success('权限分配成功');
      setAssignPermissionOpen(false);
      setSelectedPermissionId('');
      loadData();
    } catch (error: any) {
      toast.error(error.message || '分配失败');
    }
  };

  return (
    <div className="p-4 md:p-6 space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl md:text-3xl font-bold">权限管理</h1>
          <p className="text-sm text-muted-foreground mt-1">管理系统角色、权限和权限组</p>
        </div>
      </div>

      <Tabs defaultValue="overview" className="w-full">
        <TabsList className="grid w-full grid-cols-2 sm:grid-cols-4 gap-2">
          <TabsTrigger value="overview" className="flex items-center gap-2">
            <Shield className="h-4 w-4" />
            <span className="hidden sm:inline">概览</span>
          </TabsTrigger>
          <TabsTrigger value="roles" className="flex items-center gap-2">
            <Users className="h-4 w-4" />
            <span className="hidden sm:inline">角色</span>
          </TabsTrigger>
          <TabsTrigger value="permissions" className="flex items-center gap-2">
            <Key className="h-4 w-4" />
            <span className="hidden sm:inline">权限</span>
          </TabsTrigger>
          <TabsTrigger value="groups" className="flex items-center gap-2">
            <FolderKey className="h-4 w-4" />
            <span className="hidden sm:inline">权限组</span>
          </TabsTrigger>
        </TabsList>

        {/* 概览标签 */}
        <TabsContent value="overview" className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">角色总数</CardTitle>
                <Users className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{roles.length}</div>
                <p className="text-xs text-muted-foreground">系统中的角色数量</p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">权限总数</CardTitle>
                <Key className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{permissions.length}</div>
                <p className="text-xs text-muted-foreground">系统中的权限数量</p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">权限组总数</CardTitle>
                <FolderKey className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{groups.length}</div>
                <p className="text-xs text-muted-foreground">系统中的权限组数量</p>
              </CardContent>
            </Card>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* 角色与权限关系 */}
            <Card>
              <CardHeader>
                <CardTitle>角色与权限</CardTitle>
                <CardDescription>查看角色拥有的权限</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {roles.slice(0, 3).map((role) => (
                  <div key={role.id} className="border rounded-lg p-4">
                    <div className="flex items-center justify-between mb-2">
                      <h3 className="font-semibold">{role.name}</h3>
                      <Badge variant="outline">{role.permissions?.length || 0} 权限</Badge>
                    </div>
                    <p className="text-sm text-muted-foreground mb-2">{role.description}</p>
                    <div className="flex flex-wrap gap-2">
                      {role.permissions?.slice(0, 3).map((perm) => (
                        <Badge key={perm.id} variant="secondary" className="text-xs">
                          {perm.name}
                        </Badge>
                      ))}
                      {(role.permissions?.length || 0) > 3 && (
                        <Badge variant="secondary" className="text-xs">
                          +{(role.permissions?.length || 0) - 3} 更多
                        </Badge>
                      )}
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>

            {/* 权限组 */}
            <Card>
              <CardHeader>
                <CardTitle>权限组</CardTitle>
                <CardDescription>资源共享组</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {groups.slice(0, 3).map((group) => (
                  <div key={group.id} className="border rounded-lg p-4">
                    <div className="flex items-center justify-between mb-2">
                      <h3 className="font-semibold">{group.name}</h3>
                      <Badge variant="outline">{group.users?.length || 0} 用户</Badge>
                    </div>
                    <p className="text-sm text-muted-foreground">{group.description}</p>
                  </div>
                ))}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* 角色标签 */}
        <TabsContent value="roles" className="space-y-4">
          <div className="flex justify-between items-center">
            <h2 className="text-xl font-semibold">角色列表</h2>
            <Dialog open={newRoleOpen} onOpenChange={setNewRoleOpen}>
              <DialogTrigger asChild>
                <Button size="sm">
                  <Plus className="h-4 w-4 mr-2" />
                  新建角色
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>创建新角色</DialogTitle>
                  <DialogDescription>为系统创建一个新的角色</DialogDescription>
                </DialogHeader>
                <div className="space-y-4">
                  <div>
                    <Label htmlFor="role-name">角色名称</Label>
                    <Input
                      id="role-name"
                      value={roleName}
                      onChange={(e) => setRoleName(e.target.value)}
                      placeholder="输入角色名称"
                    />
                  </div>
                  <div>
                    <Label htmlFor="role-desc">描述</Label>
                    <Input
                      id="role-desc"
                      value={roleDesc}
                      onChange={(e) => setRoleDesc(e.target.value)}
                      placeholder="输入角色描述"
                    />
                  </div>
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setNewRoleOpen(false)}>取消</Button>
                  <Button onClick={handleCreateRole}>创建</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {roles.map((role) => (
              <Card key={role.id} className="hover:shadow-lg transition-shadow cursor-pointer" onClick={() => setSelectedRole(role)}>
                <CardHeader>
                  <CardTitle className="flex items-center justify-between">
                    <span className="text-lg">{role.name}</span>
                    <Badge variant="outline">{role.permissions?.length || 0}</Badge>
                  </CardTitle>
                  <CardDescription>{role.description}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">权限数量</span>
                      <span className="font-medium">{role.permissions?.length || 0}</span>
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      className="w-full"
                      onClick={(e) => {
                        e.stopPropagation();
                        setSelectedRole(role);
                        setAssignPermissionOpen(true);
                      }}
                    >
                      <Link className="h-4 w-4 mr-2" />
                      分配权限
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* 权限标签 */}
        <TabsContent value="permissions" className="space-y-4">
          <div className="flex justify-between items-center">
            <h2 className="text-xl font-semibold">权限列表</h2>
            <Dialog open={newPermissionOpen} onOpenChange={setNewPermissionOpen}>
              <DialogTrigger asChild>
                <Button size="sm">
                  <Plus className="h-4 w-4 mr-2" />
                  新建权限
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>创建新权限</DialogTitle>
                  <DialogDescription>为系统创建一个新的权限</DialogDescription>
                </DialogHeader>
                <div className="space-y-4">
                  <div>
                    <Label htmlFor="perm-name">权限名称</Label>
                    <Input
                      id="perm-name"
                      value={permName}
                      onChange={(e) => setPermName(e.target.value)}
                      placeholder="输入权限名称"
                    />
                  </div>
                  <div>
                    <Label htmlFor="perm-desc">描述</Label>
                    <Input
                      id="perm-desc"
                      value={permDesc}
                      onChange={(e) => setPermDesc(e.target.value)}
                      placeholder="输入权限描述"
                    />
                  </div>
                  <div>
                    <Label htmlFor="perm-action">操作</Label>
                    <Select value={permAction} onValueChange={setPermAction}>
                      <SelectTrigger id="perm-action">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="read">读取</SelectItem>
                        <SelectItem value="write">写入</SelectItem>
                        <SelectItem value="delete">删除</SelectItem>
                        <SelectItem value="execute">执行</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <Label htmlFor="perm-resource">资源</Label>
                    <Input
                      id="perm-resource"
                      value={permResource}
                      onChange={(e) => setPermResource(e.target.value)}
                      placeholder="* 表示所有资源"
                    />
                  </div>
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setNewPermissionOpen(false)}>取消</Button>
                  <Button onClick={handleCreatePermission}>创建</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {permissions.map((perm) => (
              <Card key={perm.id}>
                <CardHeader>
                  <CardTitle className="text-base">{perm.name}</CardTitle>
                  <CardDescription className="text-xs">{perm.description}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2 text-sm">
                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground">操作</span>
                      <Badge variant="secondary">{perm.action}</Badge>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground">资源</span>
                      <Badge variant="outline">{perm.resource}</Badge>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* 权限组标签 */}
        <TabsContent value="groups" className="space-y-4">
          <div className="flex justify-between items-center">
            <h2 className="text-xl font-semibold">权限组列表</h2>
            <Dialog open={newGroupOpen} onOpenChange={setNewGroupOpen}>
              <DialogTrigger asChild>
                <Button size="sm">
                  <Plus className="h-4 w-4 mr-2" />
                  新建权限组
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>创建新权限组</DialogTitle>
                  <DialogDescription>创建一个资源共享组</DialogDescription>
                </DialogHeader>
                <div className="space-y-4">
                  <div>
                    <Label htmlFor="group-name">权限组名称</Label>
                    <Input
                      id="group-name"
                      value={groupName}
                      onChange={(e) => setGroupName(e.target.value)}
                      placeholder="输入权限组名称"
                    />
                  </div>
                  <div>
                    <Label htmlFor="group-desc">描述</Label>
                    <Input
                      id="group-desc"
                      value={groupDesc}
                      onChange={(e) => setGroupDesc(e.target.value)}
                      placeholder="输入权限组描述"
                    />
                  </div>
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setNewGroupOpen(false)}>取消</Button>
                  <Button onClick={handleCreateGroup}>创建</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {groups.map((group) => (
              <Card key={group.id} className="hover:shadow-lg transition-shadow cursor-pointer" onClick={() => setSelectedGroup(group)}>
                <CardHeader>
                  <CardTitle className="text-lg">{group.name}</CardTitle>
                  <CardDescription>{group.description}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">成员数量</span>
                      <Badge variant="outline">{group.users?.length || 0}</Badge>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">资源数量</span>
                      <Badge variant="outline">{group.resources?.length || 0}</Badge>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>
      </Tabs>

      {/* 分配权限对话框 */}
      <Dialog open={assignPermissionOpen} onOpenChange={setAssignPermissionOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>分配权限到角色</DialogTitle>
            <DialogDescription>
              为角色 "{selectedRole?.name}" 分配权限
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label htmlFor="select-permission">选择权限</Label>
              <Select value={selectedPermissionId} onValueChange={setSelectedPermissionId}>
                <SelectTrigger id="select-permission">
                  <SelectValue placeholder="选择一个权限" />
                </SelectTrigger>
                <SelectContent>
                  {permissions.map((perm) => (
                    <SelectItem key={perm.id} value={perm.id}>
                      {perm.name} ({perm.action} - {perm.resource})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setAssignPermissionOpen(false)}>取消</Button>
            <Button onClick={handleAssignPermission}>分配</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
