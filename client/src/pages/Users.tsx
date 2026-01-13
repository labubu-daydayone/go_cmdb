import React, { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { useAuth } from '@/contexts/AuthContext';
import { useWebSocket } from '@/hooks/useWebSocket';
import { userAPI } from '@/lib/api';
import { toast } from 'sonner';
import { Trash2, Plus, Eye, Lock } from 'lucide-react';

interface User {
  id: string;
  username: string;
  email: string;
  created_at: string;
}

export default function Users() {
  const { token } = useAuth();
  const [users, setUsers] = useState<User[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isPasswordDialogOpen, setIsPasswordDialogOpen] = useState(false);
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
  });
  const [passwordData, setPasswordData] = useState({
    oldPassword: '',
    newPassword: '',
    confirmPassword: '',
  });

  // WebSocket连接
  const { isConnected, subscribe } = useWebSocket(token, {
    onMessage: (message) => {
      if (message.resource === 'user') {
        if (message.action === 'create') {
          setUsers((prev) => [message.data, ...prev]);
          toast.success('新用户已添加');
        } else if (message.action === 'delete') {
          setUsers((prev) => prev.filter((u) => u.id !== message.resource_id));
          toast.success('用户已删除');
        } else if (message.action === 'update') {
          setUsers((prev) =>
            prev.map((u) => (u.id === message.resource_id ? message.data : u))
          );
          toast.success('用户已更新');
        }
      }
    },
  });

  // 加载用户列表
  const loadUsers = async () => {
    try {
      setIsLoading(true);
      const response: any = await userAPI.list(1, 100);
      setUsers(response.data.items || []);
    } catch (error) {
      toast.error('加载用户列表失败');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadUsers();
  }, []);

  // 订阅用户列表更新
  useEffect(() => {
    if (isConnected) {
      subscribe('user:list');
    }
  }, [isConnected, subscribe]);

  const handleCreateUser = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await userAPI.create(formData.username, formData.email, formData.password);
      toast.success('用户创建成功');
      setIsDialogOpen(false);
      setFormData({ username: '', email: '', password: '' });
      loadUsers();
    } catch (error: any) {
      toast.error(error?.message || '创建失败');
    }
  };

  const handleDeleteUser = async (userId: string) => {
    if (!window.confirm('确定要删除这个用户吗？')) return;
    try {
      await userAPI.delete(userId);
      toast.success('用户已删除');
      loadUsers();
    } catch (error: any) {
      toast.error(error?.message || '删除失败');
    }
  };

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (passwordData.newPassword !== passwordData.confirmPassword) {
      toast.error('新密码不一致');
      return;
    }
    try {
      await userAPI.changePassword(passwordData.oldPassword, passwordData.newPassword);
      toast.success('密码已修改');
      setIsPasswordDialogOpen(false);
      setPasswordData({ oldPassword: '', newPassword: '', confirmPassword: '' });
    } catch (error: any) {
      toast.error(error?.message || '修改失败');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold">用户管理</h1>
          <p className="text-gray-600">管理系统用户账号</p>
        </div>
        <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
          <DialogTrigger asChild>
            <Button className="gap-2">
              <Plus className="w-4 h-4" />
              创建用户
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>创建新用户</DialogTitle>
              <DialogDescription>添加一个新的系统用户</DialogDescription>
            </DialogHeader>
            <form onSubmit={handleCreateUser} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">用户名</label>
                <Input
                  placeholder="输入用户名"
                  value={formData.username}
                  onChange={(e) =>
                    setFormData({ ...formData, username: e.target.value })
                  }
                  required
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">邮箱</label>
                <Input
                  type="email"
                  placeholder="输入邮箱"
                  value={formData.email}
                  onChange={(e) =>
                    setFormData({ ...formData, email: e.target.value })
                  }
                  required
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">密码</label>
                <Input
                  type="password"
                  placeholder="输入密码"
                  value={formData.password}
                  onChange={(e) =>
                    setFormData({ ...formData, password: e.target.value })
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
          <CardTitle>用户列表</CardTitle>
          <CardDescription>
            {isConnected && (
              <span className="text-green-600">● 实时连接中</span>
            )}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="text-center py-8">加载中...</div>
          ) : users.length === 0 ? (
            <div className="text-center py-8 text-gray-500">暂无用户</div>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>用户名</TableHead>
                    <TableHead>邮箱</TableHead>
                    <TableHead>创建时间</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {users.map((user) => (
                    <TableRow key={user.id}>
                      <TableCell className="font-medium">{user.username}</TableCell>
                      <TableCell>{user.email}</TableCell>
                      <TableCell>
                        {new Date(user.created_at).toLocaleDateString()}
                      </TableCell>
                      <TableCell className="text-right space-x-2">
                        <Dialog open={isPasswordDialogOpen && selectedUserId === user.id} onOpenChange={(open) => {
                          setIsPasswordDialogOpen(open);
                          if (open) setSelectedUserId(user.id);
                        }}>
                          <DialogTrigger asChild>
                            <Button variant="outline" size="sm" className="gap-1">
                              <Lock className="w-4 h-4" />
                              改密码
                            </Button>
                          </DialogTrigger>
                          <DialogContent>
                            <DialogHeader>
                              <DialogTitle>修改密码</DialogTitle>
                            </DialogHeader>
                            <form onSubmit={handleChangePassword} className="space-y-4">
                              <div className="space-y-2">
                                <label className="text-sm font-medium">旧密码</label>
                                <Input
                                  type="password"
                                  placeholder="输入旧密码"
                                  value={passwordData.oldPassword}
                                  onChange={(e) =>
                                    setPasswordData({
                                      ...passwordData,
                                      oldPassword: e.target.value,
                                    })
                                  }
                                  required
                                />
                              </div>
                              <div className="space-y-2">
                                <label className="text-sm font-medium">新密码</label>
                                <Input
                                  type="password"
                                  placeholder="输入新密码"
                                  value={passwordData.newPassword}
                                  onChange={(e) =>
                                    setPasswordData({
                                      ...passwordData,
                                      newPassword: e.target.value,
                                    })
                                  }
                                  required
                                />
                              </div>
                              <div className="space-y-2">
                                <label className="text-sm font-medium">确认新密码</label>
                                <Input
                                  type="password"
                                  placeholder="确认新密码"
                                  value={passwordData.confirmPassword}
                                  onChange={(e) =>
                                    setPasswordData({
                                      ...passwordData,
                                      confirmPassword: e.target.value,
                                    })
                                  }
                                  required
                                />
                              </div>
                              <Button type="submit" className="w-full">
                                确认修改
                              </Button>
                            </form>
                          </DialogContent>
                        </Dialog>
                        <Button
                          variant="destructive"
                          size="sm"
                          onClick={() => handleDeleteUser(user.id)}
                          className="gap-1"
                        >
                          <Trash2 className="w-4 h-4" />
                          删除
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
