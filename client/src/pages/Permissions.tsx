import React, { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { permissionAPI } from '@/lib/api';
import { toast } from 'sonner';
import { Plus, Trash2 } from 'lucide-react';

interface Role {
  id: string;
  name: string;
  description: string;
  created_at: string;
}

interface Permission {
  id: string;
  name: string;
  description: string;
  action: string;
  resource: string;
  created_at: string;
}

interface PermissionGroup {
  id: string;
  name: string;
  description: string;
  created_at: string;
}

export default function Permissions() {
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [groups, setGroups] = useState<PermissionGroup[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [dialogType, setDialogType] = useState<'role' | 'permission' | 'group'>('role');
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    action: '',
    resource: '',
  });

  // 加载数据
  const loadData = async () => {
    try {
      setIsLoading(true);
      const [rolesRes, permissionsRes, groupsRes] = await Promise.all([
        permissionAPI.listRoles(1, 100),
        permissionAPI.list(1, 100),
        permissionAPI.listGroups(1, 100),
      ]);

      setRoles((rolesRes as any).data.items || []);
      setPermissions((permissionsRes as any).data.items || []);
      setGroups((groupsRes as any).data.items || []);
    } catch (error) {
      toast.error('加载数据失败');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleCreateRole = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await permissionAPI.createRole(formData.name, formData.description);
      toast.success('角色创建成功');
      setIsDialogOpen(false);
      setFormData({ name: '', description: '', action: '', resource: '' });
      loadData();
    } catch (error: any) {
      toast.error(error?.message || '创建失败');
    }
  };

  const handleCreatePermission = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await permissionAPI.create(
        formData.name,
        formData.description,
        formData.action,
        formData.resource
      );
      toast.success('权限创建成功');
      setIsDialogOpen(false);
      setFormData({ name: '', description: '', action: '', resource: '' });
      loadData();
    } catch (error: any) {
      toast.error(error?.message || '创建失败');
    }
  };

  const handleCreateGroup = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await permissionAPI.createGroup(formData.name, formData.description);
      toast.success('权限组创建成功');
      setIsDialogOpen(false);
      setFormData({ name: '', description: '', action: '', resource: '' });
      loadData();
    } catch (error: any) {
      toast.error(error?.message || '创建失败');
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (dialogType === 'role') {
      await handleCreateRole(e);
    } else if (dialogType === 'permission') {
      await handleCreatePermission(e);
    } else {
      await handleCreateGroup(e);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">权限管理</h1>
        <p className="text-gray-600">管理角色、权限和权限组</p>
      </div>

      <Tabs defaultValue="roles" className="w-full">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="roles">角色</TabsTrigger>
          <TabsTrigger value="permissions">权限</TabsTrigger>
          <TabsTrigger value="groups">权限组</TabsTrigger>
        </TabsList>

        {/* 角色标签页 */}
        <TabsContent value="roles" className="space-y-4">
          <div className="flex justify-end">
            <Dialog open={isDialogOpen && dialogType === 'role'} onOpenChange={(open) => {
              setIsDialogOpen(open);
              if (open) setDialogType('role');
            }}>
              <DialogTrigger asChild>
                <Button className="gap-2">
                  <Plus className="w-4 h-4" />
                  创建角色
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>创建新角色</DialogTitle>
                </DialogHeader>
                <form onSubmit={handleSubmit} className="space-y-4">
                  <div className="space-y-2">
                    <label className="text-sm font-medium">角色名称</label>
                    <Input
                      placeholder="输入角色名称"
                      value={formData.name}
                      onChange={(e) =>
                        setFormData({ ...formData, name: e.target.value })
                      }
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">描述</label>
                    <Input
                      placeholder="输入描述"
                      value={formData.description}
                      onChange={(e) =>
                        setFormData({ ...formData, description: e.target.value })
                      }
                      required
                    />
                  </div>
                  <Button type="submit" className="w-full">
                    创建
                  </Button>
                </form>
              </DialogContent>
            </Dialog>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>角色列表</CardTitle>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <div className="text-center py-8">加载中...</div>
              ) : roles.length === 0 ? (
                <div className="text-center py-8 text-gray-500">暂无角色</div>
              ) : (
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>角色名称</TableHead>
                        <TableHead>描述</TableHead>
                        <TableHead>创建时间</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {roles.map((role) => (
                        <TableRow key={role.id}>
                          <TableCell className="font-medium">{role.name}</TableCell>
                          <TableCell>{role.description}</TableCell>
                          <TableCell>
                            {new Date(role.created_at).toLocaleDateString()}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* 权限标签页 */}
        <TabsContent value="permissions" className="space-y-4">
          <div className="flex justify-end">
            <Dialog open={isDialogOpen && dialogType === 'permission'} onOpenChange={(open) => {
              setIsDialogOpen(open);
              if (open) setDialogType('permission');
            }}>
              <DialogTrigger asChild>
                <Button className="gap-2">
                  <Plus className="w-4 h-4" />
                  创建权限
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>创建新权限</DialogTitle>
                </DialogHeader>
                <form onSubmit={handleSubmit} className="space-y-4">
                  <div className="space-y-2">
                    <label className="text-sm font-medium">权限名称</label>
                    <Input
                      placeholder="输入权限名称"
                      value={formData.name}
                      onChange={(e) =>
                        setFormData({ ...formData, name: e.target.value })
                      }
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">描述</label>
                    <Input
                      placeholder="输入描述"
                      value={formData.description}
                      onChange={(e) =>
                        setFormData({ ...formData, description: e.target.value })
                      }
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">操作</label>
                    <Input
                      placeholder="如: read, write, delete"
                      value={formData.action}
                      onChange={(e) =>
                        setFormData({ ...formData, action: e.target.value })
                      }
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">资源</label>
                    <Input
                      placeholder="如: domain, nginx, script"
                      value={formData.resource}
                      onChange={(e) =>
                        setFormData({ ...formData, resource: e.target.value })
                      }
                      required
                    />
                  </div>
                  <Button type="submit" className="w-full">
                    创建
                  </Button>
                </form>
              </DialogContent>
            </Dialog>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>权限列表</CardTitle>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <div className="text-center py-8">加载中...</div>
              ) : permissions.length === 0 ? (
                <div className="text-center py-8 text-gray-500">暂无权限</div>
              ) : (
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>权限名称</TableHead>
                        <TableHead>操作</TableHead>
                        <TableHead>资源</TableHead>
                        <TableHead>描述</TableHead>
                        <TableHead>创建时间</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {permissions.map((permission) => (
                        <TableRow key={permission.id}>
                          <TableCell className="font-medium">
                            {permission.name}
                          </TableCell>
                          <TableCell>{permission.action}</TableCell>
                          <TableCell>{permission.resource}</TableCell>
                          <TableCell>{permission.description}</TableCell>
                          <TableCell>
                            {new Date(permission.created_at).toLocaleDateString()}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* 权限组标签页 */}
        <TabsContent value="groups" className="space-y-4">
          <div className="flex justify-end">
            <Dialog open={isDialogOpen && dialogType === 'group'} onOpenChange={(open) => {
              setIsDialogOpen(open);
              if (open) setDialogType('group');
            }}>
              <DialogTrigger asChild>
                <Button className="gap-2">
                  <Plus className="w-4 h-4" />
                  创建权限组
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>创建新权限组</DialogTitle>
                </DialogHeader>
                <form onSubmit={handleSubmit} className="space-y-4">
                  <div className="space-y-2">
                    <label className="text-sm font-medium">权限组名称</label>
                    <Input
                      placeholder="输入权限组名称"
                      value={formData.name}
                      onChange={(e) =>
                        setFormData({ ...formData, name: e.target.value })
                      }
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">描述</label>
                    <Input
                      placeholder="输入描述"
                      value={formData.description}
                      onChange={(e) =>
                        setFormData({ ...formData, description: e.target.value })
                      }
                      required
                    />
                  </div>
                  <Button type="submit" className="w-full">
                    创建
                  </Button>
                </form>
              </DialogContent>
            </Dialog>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>权限组列表</CardTitle>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <div className="text-center py-8">加载中...</div>
              ) : groups.length === 0 ? (
                <div className="text-center py-8 text-gray-500">暂无权限组</div>
              ) : (
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>权限组名称</TableHead>
                        <TableHead>描述</TableHead>
                        <TableHead>创建时间</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {groups.map((group) => (
                        <TableRow key={group.id}>
                          <TableCell className="font-medium">{group.name}</TableCell>
                          <TableCell>{group.description}</TableCell>
                          <TableCell>
                            {new Date(group.created_at).toLocaleDateString()}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
