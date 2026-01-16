import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Plus, Edit2, Trash2, RefreshCw, Search } from 'lucide-react';
import { toast } from 'sonner';

/**
 * Website type
 * 对应Go后端的 Website 结构体
 */
interface Website {
  id: number;
  domain: string;
  cname: string;
  routeGroupId: number;
  routeGroupName: string;
  permissionGroupId: number;
  permissionGroupName: string;
  status: 'active' | 'inactive' | 'maintenance';
  createdAt: string;
  updatedAt: string;
}

// Mock数据
const mockWebsites: Website[] = [
  {
    id: 1,
    domain: 'www.example.com',
    cname: 'cdn.example.com.cdn.cloudflare.net',
    routeGroupId: 1,
    routeGroupName: '国内线路组',
    permissionGroupId: 1,
    permissionGroupName: '运维组',
    status: 'active',
    createdAt: '2024-01-01',
    updatedAt: '2024-01-15',
  },
  {
    id: 2,
    domain: 'api.example.com',
    cname: 'api-lb.example.com.cdn.cloudflare.net',
    routeGroupId: 2,
    routeGroupName: '海外线路组',
    permissionGroupId: 2,
    permissionGroupName: '开发组',
    status: 'active',
    createdAt: '2024-01-02',
    updatedAt: '2024-01-16',
  },
  {
    id: 3,
    domain: 'blog.example.com',
    cname: 'blog.example.com.cdn.cloudflare.net',
    routeGroupId: 1,
    routeGroupName: '国内线路组',
    permissionGroupId: 1,
    permissionGroupName: '运维组',
    status: 'maintenance',
    createdAt: '2024-01-03',
    updatedAt: '2024-01-17',
  },
  {
    id: 4,
    domain: 'shop.example.com',
    cname: 'shop.example.com.cdn.cloudflare.net',
    routeGroupId: 3,
    routeGroupName: '全球线路组',
    permissionGroupId: 3,
    permissionGroupName: '电商组',
    status: 'active',
    createdAt: '2024-01-04',
    updatedAt: '2024-01-18',
  },
  {
    id: 5,
    domain: 'admin.example.com',
    cname: 'admin.example.com.cdn.cloudflare.net',
    routeGroupId: 1,
    routeGroupName: '国内线路组',
    permissionGroupId: 1,
    permissionGroupName: '运维组',
    status: 'inactive',
    createdAt: '2024-01-05',
    updatedAt: '2024-01-19',
  },
];

// Mock线路组数据
const mockRouteGroups = [
  { id: 1, name: '国内线路组' },
  { id: 2, name: '海外线路组' },
  { id: 3, name: '全球线路组' },
];

// Mock权限组数据
const mockPermissionGroups = [
  { id: 1, name: '运维组' },
  { id: 2, name: '开发组' },
  { id: 3, name: '电商组' },
];

const statusColors = {
  active: 'bg-green-100 text-green-800',
  inactive: 'bg-gray-100 text-gray-800',
  maintenance: 'bg-yellow-100 text-yellow-800',
};

const statusLabels = {
  active: '运行中',
  inactive: '已停用',
  maintenance: '维护中',
};

export default function WebsiteList() {
  const [websites, setWebsites] = useState<Website[]>(mockWebsites);
  const [searchTerm, setSearchTerm] = useState('');
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [isClearCacheDialogOpen, setIsClearCacheDialogOpen] = useState(false);
  const [selectedWebsite, setSelectedWebsite] = useState<Website | null>(null);
  const [editForm, setEditForm] = useState<Partial<Website>>({});

  /**
   * TODO: 对接Go API
   * GET /api/v1/websites?page=1&pageSize=20&search=xxx - 获取网站列表（分页+搜索）
   * POST /api/v1/websites - 创建网站
   * PUT /api/v1/websites/:id - 更新网站
   * DELETE /api/v1/websites/:id - 删除网站
   * POST /api/v1/websites/:id/clear-cache - 清理网站缓存
   * 
   * WebSocket实时更新：
   * ws://localhost:8080/api/v1/ws/websites
   * 消息格式：
   * {
   *   type: 'website_added' | 'website_updated' | 'website_deleted' | 'cache_cleared',
   *   data: Website | { id: number }
   * }
   */

  // 搜索过滤
  const filteredWebsites = websites.filter(website =>
    website.domain.toLowerCase().includes(searchTerm.toLowerCase()) ||
    website.cname.toLowerCase().includes(searchTerm.toLowerCase()) ||
    website.routeGroupName.toLowerCase().includes(searchTerm.toLowerCase()) ||
    website.permissionGroupName.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const handleAdd = () => {
    setSelectedWebsite(null);
    setEditForm({
      domain: '',
      cname: '',
      routeGroupId: 1,
      permissionGroupId: 1,
      status: 'active',
    });
    setIsEditDialogOpen(true);
  };

  const handleEdit = (website: Website) => {
    setSelectedWebsite(website);
    setEditForm({
      domain: website.domain,
      cname: website.cname,
      routeGroupId: website.routeGroupId,
      permissionGroupId: website.permissionGroupId,
      status: website.status,
    });
    setIsEditDialogOpen(true);
  };

  const handleSave = () => {
    if (!editForm.domain || !editForm.cname) {
      toast.error('请填写域名和CNAME');
      return;
    }

    // 域名格式验证
    const domainRegex = /^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/;
    if (!domainRegex.test(editForm.domain)) {
      toast.error('域名格式不正确');
      return;
    }

    if (selectedWebsite) {
      // 更新
      // TODO: 调用Go API更新网站
      // await fetch(`/api/v1/websites/${selectedWebsite.id}`, {
      //   method: 'PUT',
      //   headers: { 'Content-Type': 'application/json' },
      //   body: JSON.stringify(editForm),
      // });

      const routeGroup = mockRouteGroups.find(g => g.id === editForm.routeGroupId);
      const permissionGroup = mockPermissionGroups.find(g => g.id === editForm.permissionGroupId);

      setWebsites(websites.map(w =>
        w.id === selectedWebsite.id
          ? {
              ...w,
              ...editForm,
              routeGroupName: routeGroup?.name || w.routeGroupName,
              permissionGroupName: permissionGroup?.name || w.permissionGroupName,
              updatedAt: new Date().toISOString().split('T')[0],
            }
          : w
      ));
      toast.success('网站更新成功');
    } else {
      // 新增
      // TODO: 调用Go API创建网站
      // const response = await fetch('/api/v1/websites', {
      //   method: 'POST',
      //   headers: { 'Content-Type': 'application/json' },
      //   body: JSON.stringify(editForm),
      // });
      // const newWebsite = await response.json();

      const routeGroup = mockRouteGroups.find(g => g.id === editForm.routeGroupId);
      const permissionGroup = mockPermissionGroups.find(g => g.id === editForm.permissionGroupId);

      const newWebsite: Website = {
        id: Math.max(...websites.map(w => w.id)) + 1,
        domain: editForm.domain!,
        cname: editForm.cname!,
        routeGroupId: editForm.routeGroupId!,
        routeGroupName: routeGroup?.name || '未知线路组',
        permissionGroupId: editForm.permissionGroupId!,
        permissionGroupName: permissionGroup?.name || '未知权限组',
        status: editForm.status as Website['status'],
        createdAt: new Date().toISOString().split('T')[0],
        updatedAt: new Date().toISOString().split('T')[0],
      };
      setWebsites([...websites, newWebsite]);
      toast.success('网站创建成功');
    }

    setIsEditDialogOpen(false);
  };

  const handleDeleteConfirm = () => {
    if (!selectedWebsite) return;

    // TODO: 调用Go API删除网站
    // await fetch(`/api/v1/websites/${selectedWebsite.id}`, {
    //   method: 'DELETE',
    // });

    setWebsites(websites.filter(w => w.id !== selectedWebsite.id));
    toast.success('网站删除成功');
    setIsDeleteDialogOpen(false);
    setSelectedWebsite(null);
  };

  const handleClearCache = (website: Website) => {
    setSelectedWebsite(website);
    setIsClearCacheDialogOpen(true);
  };

  const handleClearCacheConfirm = () => {
    if (!selectedWebsite) return;

    // TODO: 调用Go API清理缓存
    // await fetch(`/api/v1/websites/${selectedWebsite.id}/clear-cache`, {
    //   method: 'POST',
    // });

    toast.success(`正在清理 ${selectedWebsite.domain} 的缓存...`);
    setIsClearCacheDialogOpen(false);
    setSelectedWebsite(null);
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">网站列表</h1>
          <p className="text-muted-foreground mt-1">管理所有网站信息</p>
        </div>
        <Button onClick={handleAdd}>
          <Plus className="w-4 h-4 mr-2" />
          添加网站
        </Button>
      </div>

      <Card>
        <CardContent className="p-6">
          {/* 搜索栏 */}
          <div className="mb-4">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input
                placeholder="搜索域名、CNAME、线路组、权限组..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="pl-10"
              />
            </div>
          </div>

          {/* 网站列表表格 */}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>域名</TableHead>
                <TableHead>CNAME</TableHead>
                <TableHead>线路组</TableHead>
                <TableHead>权限组</TableHead>
                <TableHead>状态</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredWebsites.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                    {searchTerm ? '未找到匹配的网站' : '暂无网站数据'}
                  </TableCell>
                </TableRow>
              ) : (
                filteredWebsites.map((website) => (
                  <TableRow key={website.id}>
                    <TableCell className="font-medium">{website.domain}</TableCell>
                    <TableCell className="font-mono text-sm max-w-xs truncate" title={website.cname}>
                      {website.cname}
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{website.routeGroupName}</Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{website.permissionGroupName}</Badge>
                    </TableCell>
                    <TableCell>
                      <Badge className={statusColors[website.status]}>
                        {statusLabels[website.status]}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleEdit(website)}
                        >
                          <Edit2 className="w-4 h-4 mr-1" />
                          编辑
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleClearCache(website)}
                        >
                          <RefreshCw className="w-4 h-4 mr-1" />
                          缓存清理
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setSelectedWebsite(website);
                            setIsDeleteDialogOpen(true);
                          }}
                        >
                          <Trash2 className="w-4 h-4 mr-1" />
                          删除
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* 编辑/新增对话框 */}
      <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{selectedWebsite ? '编辑网站' : '添加网站'}</DialogTitle>
            <DialogDescription>
              {selectedWebsite ? '修改网站信息' : '创建新的网站配置'}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="domain">域名 *</Label>
              <Input
                id="domain"
                placeholder="例如：www.example.com"
                value={editForm.domain || ''}
                onChange={(e) => setEditForm({ ...editForm, domain: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="cname">CNAME *</Label>
              <Input
                id="cname"
                placeholder="例如：cdn.example.com.cdn.cloudflare.net"
                value={editForm.cname || ''}
                onChange={(e) => setEditForm({ ...editForm, cname: e.target.value })}
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="routeGroup">线路组 *</Label>
                <Select
                  value={editForm.routeGroupId?.toString()}
                  onValueChange={(value) => setEditForm({ ...editForm, routeGroupId: parseInt(value) })}
                >
                  <SelectTrigger id="routeGroup">
                    <SelectValue placeholder="选择线路组" />
                  </SelectTrigger>
                  <SelectContent>
                    {mockRouteGroups.map((group) => (
                      <SelectItem key={group.id} value={group.id.toString()}>
                        {group.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="permissionGroup">权限组 *</Label>
                <Select
                  value={editForm.permissionGroupId?.toString()}
                  onValueChange={(value) => setEditForm({ ...editForm, permissionGroupId: parseInt(value) })}
                >
                  <SelectTrigger id="permissionGroup">
                    <SelectValue placeholder="选择权限组" />
                  </SelectTrigger>
                  <SelectContent>
                    {mockPermissionGroups.map((group) => (
                      <SelectItem key={group.id} value={group.id.toString()}>
                        {group.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="status">状态 *</Label>
              <Select
                value={editForm.status}
                onValueChange={(value) => setEditForm({ ...editForm, status: value as Website['status'] })}
              >
                <SelectTrigger id="status">
                  <SelectValue placeholder="选择状态" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="active">运行中</SelectItem>
                  <SelectItem value="inactive">已停用</SelectItem>
                  <SelectItem value="maintenance">维护中</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsEditDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleSave}>
              {selectedWebsite ? '保存' : '创建'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 删除确认对话框 */}
      <Dialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>
              确定要删除网站 <span className="font-semibold">{selectedWebsite?.domain}</span> 吗？此操作无法撤销。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsDeleteDialogOpen(false)}>
              取消
            </Button>
            <Button variant="destructive" onClick={handleDeleteConfirm}>
              删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 缓存清理确认对话框 */}
      <Dialog open={isClearCacheDialogOpen} onOpenChange={setIsClearCacheDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认清理缓存</DialogTitle>
            <DialogDescription>
              确定要清理网站 <span className="font-semibold">{selectedWebsite?.domain}</span> 的缓存吗？
              <br />
              此操作将清除CDN节点上的所有缓存文件。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsClearCacheDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleClearCacheConfirm}>
              确认清理
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
