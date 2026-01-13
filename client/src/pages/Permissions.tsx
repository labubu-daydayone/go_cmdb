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
import { Checkbox } from '../components/ui/checkbox';
import { useWebSocket } from '../hooks/useWebSocket';
import { toast } from 'sonner';
import { Shield, Users, Key, FolderKey, Plus, Trash2, Link, X } from 'lucide-react';

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
  const [permResource, setPermResource] = useState('user');
  const [groupName, setGroupName] = useState('');
  const [groupDesc, setGroupDesc] = useState('');
  const [selectedPermissionIds, setSelectedPermissionIds] = useState<string[]>([]);

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
      setPermAction('read');
      setPermResource('user');
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

  const handleAssignPermissions = async () => {
    if (!selectedRole || selectedPermissionIds.length === 0) {
      toast.error('请选择权限');
      return;
    }
    try {
      // 批量分配权限
      await Promise.all(
        selectedPermissionIds.map(permId =>
          permissionAPI.assignPermissionToRole(selectedRole.id, permId)
        )
      );
      toast.success(`已分配 ${selectedPermissionIds.length} 个权限`);
      setAssignPermissionOpen(false);
      setSelectedPermissionIds([]);
      loadData();
    } catch (error: any) {
      toast.error(error.message || '分配失败');
    }
  };

  const handleRemovePermission = async (roleId: string, permId: string) => {
    try {
      await permissionAPI.removePermissionFromRole(roleId, permId);
      toast.success('权限已移除');
      loadData();
    } catch (error: any) {
      toast.error(error.message || '移除失败');
    }
  };

  const togglePermissionSelection = (permId: string) => {
    setSelectedPermissionIds(prev =>
      prev.includes(permId)
        ? prev.filter(id => id !== permId)
        : [...prev, permId]
    );
  };

  // 获取可分配的权限（排除已分配的）
  const getAvailablePermissions = () => {
    if (!selectedRole) return permissions;
    const assignedIds = (selectedRole.permissions || []).map(p => p.id);
    return permissions.filter(p => !assignedIds.includes(p.id));
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl md:text-3xl font-bold">权限管理</h1>
          <p className="text-gray-500 text-sm md:text-base">管理系统角色、权限和权限组</p>
        </div>
      </div>

      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList className="grid w-full grid-cols-4 h-auto">
          <TabsTrigger value="overview" className="text-xs sm:text-sm py-2">
            <Shield className="w-4 h-4 mr-1 sm:mr-2" />
            <span className="hidden sm:inline">概览</span>
          </TabsTrigger>
          <TabsTrigger value="roles" className="text-xs sm:text-sm py-2">
            <Users className="w-4 h-4 mr-1 sm:mr-2" />
            <span className="hidden sm:inline">角色</span>
          </TabsTrigger>
          <TabsTrigger value="permissions" className="text-xs sm:text-sm py-2">
            <Key className="w-4 h-4 mr-1 sm:mr-2" />
            <span className="hidden sm:inline">权限</span>
          </TabsTrigger>
          <TabsTrigger value="groups" className="text-xs sm:text-sm py-2">
            <FolderKey className="w-4 h-4 mr-1 sm:mr-2" />
            <span className="hidden sm:inline">权限组</span>
          </TabsTrigger>
        </TabsList>

        {/* 概览标签 */}
        <TabsContent value="overview" className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {roles.map((role) => (
              <Card key={role.id} className="hover:shadow-md transition-shadow">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0">
                      <CardTitle className="text-lg truncate">{role.name}</CardTitle>
                      <CardDescription className="text-sm truncate">{role.description}</CardDescription>
                    </div>
                    <Badge variant="outline" className="ml-2 shrink-0">
                      {role.permissions?.length || 0} 权限
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-gray-600">已分配权限:</span>
                    </div>
                    <div className="flex flex-wrap gap-1.5 max-h-24 overflow-y-auto">
                      {role.permissions && role.permissions.length > 0 ? (
                        role.permissions.map((perm) => (
                          <Badge
                            key={perm.id}
                            variant="secondary"
                            className="text-xs flex items-center gap-1"
                          >
                            {perm.name}
                            <button
                              onClick={() => handleRemovePermission(role.id, perm.id)}
                              className="hover:text-red-600"
                            >
                              <X className="w-3 h-3" />
                            </button>
                          </Badge>
                        ))
                      ) : (
                        <span className="text-sm text-gray-400">暂无权限</span>
                      )}
                    </div>
                    <Button
                      size="sm"
                      variant="outline"
                      className="w-full mt-2"
                      onClick={() => {
                        setSelectedRole(role);
                        setAssignPermissionOpen(true);
                      }}
                    >
                      <Plus className="w-4 h-4 mr-1" />
                      分配权限
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* 角色标签 */}
        <TabsContent value="roles" className="space-y-4">
          <div className="flex justify-end">
            <Dialog open={newRoleOpen} onOpenChange={setNewRoleOpen}>
              <DialogTrigger asChild>
                <Button>
                  <Plus className="w-4 h-4 mr-2" />
                  新建角色
                </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                  <DialogTitle>新建角色</DialogTitle>
                  <DialogDescription>创建一个新的系统角色</DialogDescription>
                </DialogHeader>
                <div className="grid gap-4 py-4">
                  <div className="space-y-2">
                    <Label htmlFor="role-name">角色名称</Label>
                    <Input
                      id="role-name"
                      value={roleName}
                      onChange={(e) => setRoleName(e.target.value)}
                      placeholder="例如: 管理员"
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
                <DialogFooter>
                  <Button onClick={handleCreateRole}>创建</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {roles.map((role) => (
              <Card key={role.id}>
                <CardHeader>
                  <CardTitle>{role.name}</CardTitle>
                  <CardDescription>{role.description}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-gray-600">权限数量:</span>
                    <Badge>{role.permissions?.length || 0}</Badge>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* 权限标签 */}
        <TabsContent value="permissions" className="space-y-4">
          <div className="flex justify-end">
            <Dialog open={newPermissionOpen} onOpenChange={setNewPermissionOpen}>
              <DialogTrigger asChild>
                <Button>
                  <Plus className="w-4 h-4 mr-2" />
                  新建权限
                </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                  <DialogTitle>新建权限</DialogTitle>
                  <DialogDescription>创建一个新的系统权限</DialogDescription>
                </DialogHeader>
                <div className="grid gap-4 py-4">
                  <div className="space-y-2">
                    <Label htmlFor="perm-name">权限名称</Label>
                    <Input
                      id="perm-name"
                      value={permName}
                      onChange={(e) => setPermName(e.target.value)}
                      placeholder="例如: 用户管理-读取"
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
                        <SelectItem value="write">写入</SelectItem>
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
                <DialogFooter>
                  <Button onClick={handleCreatePermission}>创建</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {permissions.map((perm) => (
              <Card key={perm.id}>
                <CardHeader className="pb-3">
                  <CardTitle className="text-base">{perm.name}</CardTitle>
                  <CardDescription className="text-sm">{perm.description}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="flex flex-wrap gap-2">
                    <Badge variant="outline">{perm.action}</Badge>
                    <Badge variant="secondary">{perm.resource}</Badge>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* 权限组标签 */}
        <TabsContent value="groups" className="space-y-4">
          <div className="flex justify-end">
            <Dialog open={newGroupOpen} onOpenChange={setNewGroupOpen}>
              <DialogTrigger asChild>
                <Button>
                  <Plus className="w-4 h-4 mr-2" />
                  新建权限组
                </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                  <DialogTitle>新建权限组</DialogTitle>
                  <DialogDescription>创建一个新的权限组</DialogDescription>
                </DialogHeader>
                <div className="grid gap-4 py-4">
                  <div className="space-y-2">
                    <Label htmlFor="group-name">权限组名称</Label>
                    <Input
                      id="group-name"
                      value={groupName}
                      onChange={(e) => setGroupName(e.target.value)}
                      placeholder="例如: 开发组"
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
                <DialogFooter>
                  <Button onClick={handleCreateGroup}>创建</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {groups.map((group) => (
              <Card key={group.id}>
                <CardHeader>
                  <CardTitle>{group.name}</CardTitle>
                  <CardDescription>{group.description}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-gray-600">成员数:</span>
                      <Badge>{group.users?.length || 0}</Badge>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-gray-600">资源数:</span>
                      <Badge>{group.resources?.length || 0}</Badge>
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
        <DialogContent className="sm:max-w-[600px] max-h-[80vh] overflow-hidden flex flex-col">
          <DialogHeader>
            <DialogTitle>为 {selectedRole?.name} 分配权限</DialogTitle>
            <DialogDescription>
              选择要分配的权限（已选择 {selectedPermissionIds.length} 个）
            </DialogDescription>
          </DialogHeader>
          <div className="flex-1 overflow-y-auto py-4">
            <div className="space-y-2">
              {getAvailablePermissions().map((perm) => (
                <div
                  key={perm.id}
                  className={`
                    flex items-start gap-3 p-3 rounded-lg border cursor-pointer
                    transition-colors hover:bg-gray-50
                    ${selectedPermissionIds.includes(perm.id) ? 'border-blue-500 bg-blue-50' : 'border-gray-200'}
                  `}
                  onClick={() => togglePermissionSelection(perm.id)}
                >
                  <Checkbox
                    checked={selectedPermissionIds.includes(perm.id)}
                    onCheckedChange={() => togglePermissionSelection(perm.id)}
                    className="mt-0.5"
                  />
                  <div className="flex-1 min-w-0">
                    <div className="font-medium text-sm">{perm.name}</div>
                    <div className="text-xs text-gray-500 mt-0.5">{perm.description}</div>
                    <div className="flex gap-1.5 mt-1.5">
                      <Badge variant="outline" className="text-xs">{perm.action}</Badge>
                      <Badge variant="secondary" className="text-xs">{perm.resource}</Badge>
                    </div>
                  </div>
                </div>
              ))}
              {getAvailablePermissions().length === 0 && (
                <div className="text-center py-8 text-gray-500">
                  所有权限已分配
                </div>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setAssignPermissionOpen(false);
                setSelectedPermissionIds([]);
              }}
            >
              取消
            </Button>
            <Button
              onClick={handleAssignPermissions}
              disabled={selectedPermissionIds.length === 0}
            >
              分配 {selectedPermissionIds.length > 0 && `(${selectedPermissionIds.length})`}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
