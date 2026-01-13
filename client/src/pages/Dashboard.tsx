import React from 'react';
import { useAuth } from '@/contexts/AuthContext';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Users, Lock, Shield, BarChart3 } from 'lucide-react';

export default function Dashboard() {
  const { user } = useAuth();

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold">欢迎，{user?.username}！</h1>
        <p className="text-gray-600">CMDB运维管理系统</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">用户管理</CardTitle>
            <Users className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">管理系统用户</div>
            <p className="text-xs text-gray-500">创建、删除、修改用户</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">权限管理</CardTitle>
            <Lock className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">角色与权限</div>
            <p className="text-xs text-gray-500">配置用户权限</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">权限组</CardTitle>
            <Shield className="h-4 w-4 text-purple-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">批量管理</div>
            <p className="text-xs text-gray-500">权限组资源分配</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">实时更新</CardTitle>
            <BarChart3 className="h-4 w-4 text-orange-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">WebSocket</div>
            <p className="text-xs text-gray-500">实时数据推送</p>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>系统说明</CardTitle>
          <CardDescription>了解CMDB系统的功能</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <h3 className="font-semibold mb-2">👥 用户管理</h3>
            <p className="text-sm text-gray-600">
              创建、删除、修改用户账号。支持修改密码和权限分配。
            </p>
          </div>
          <div>
            <h3 className="font-semibold mb-2">🔐 权限管理</h3>
            <p className="text-sm text-gray-600">
              基于RBAC的权限系统，支持角色、权限、权限组的灵活配置。
            </p>
          </div>
          <div>
            <h3 className="font-semibold mb-2">📊 资源级权限</h3>
            <p className="text-sm text-gray-600">
              细粒度的资源级权限控制，支持用户、管理员、权限组等多层次权限。
            </p>
          </div>
          <div>
            <h3 className="font-semibold mb-2">⚡ 实时推送</h3>
            <p className="text-sm text-gray-600">
              使用WebSocket实现实时数据推送，用户列表、权限变化实时更新。
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
