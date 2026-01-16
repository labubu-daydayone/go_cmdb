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
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { Plus, Edit2, Trash2, RefreshCw, Search } from 'lucide-react';
import { toast } from 'sonner';
import { useLocation } from 'wouter';

/**
 * Website type
 * 对应Go后端的 Website 结构体
 */
interface Website {
  id: number;
  domains: string[]; // 支持多个域名
  cname: string;
  sslStatus: 'valid' | 'expired' | 'none'; // SSL证书状态
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
    domains: ['www.example.com', 'example.com'],
    cname: 'cdn.example.com.cdn.cloudflare.net',
    sslStatus: 'valid',
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
    domains: ['api.example.com'],
    cname: 'api-lb.example.com.cdn.cloudflare.net',
    sslStatus: 'valid',
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
    domains: ['blog.example.com', 'www.blog.example.com'],
    cname: 'blog.example.com.cdn.cloudflare.net',
    sslStatus: 'expired',
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
    domains: ['shop.example.com', 'm.shop.example.com', 'mobile.shop.example.com'],
    cname: 'shop.example.com.cdn.cloudflare.net',
    sslStatus: 'valid',
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
    domains: ['admin.example.com'],
    cname: 'admin.example.com.cdn.cloudflare.net',
    sslStatus: 'none',
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

const sslStatusColors = {
  valid: 'bg-green-100 text-green-800',
  expired: 'bg-red-100 text-red-800',
  none: 'bg-gray-100 text-gray-800',
};

const sslStatusLabels = {
  valid: '有效',
  expired: '已过期',
  none: '未配置',
};

export default function WebsiteList() {
  const [, setLocation] = useLocation();
  const [websites, setWebsites] = useState<Website[]>(mockWebsites);
  const [searchTerm, setSearchTerm] = useState('');
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [isClearCacheDialogOpen, setIsClearCacheDialogOpen] = useState(false);
  const [selectedWebsite, setSelectedWebsite] = useState<Website | null>(null);

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
    website.domains.some(d => d.toLowerCase().includes(searchTerm.toLowerCase())) ||
    website.cname.toLowerCase().includes(searchTerm.toLowerCase()) ||
    website.routeGroupName.toLowerCase().includes(searchTerm.toLowerCase()) ||
    website.permissionGroupName.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const handleAdd = () => {
    // 跳转到新增页面
    setLocation('/website/new');
  };

  const handleEdit = (websiteId: number) => {
    // 跳转到编辑页面
    setLocation(`/website/${websiteId}/edit`);
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

    toast.success(`正在清理 ${selectedWebsite.domains[0]} 的缓存...`);
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
                <TableHead className="w-[280px]">域名</TableHead>
                <TableHead className="w-[100px]">状态</TableHead>
                <TableHead className="w-[200px]">CNAME</TableHead>
                <TableHead className="w-[90px]">SSL状态</TableHead>
                <TableHead>线路组</TableHead>
                <TableHead>权限组</TableHead>
                <TableHead className="text-right w-[200px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredWebsites.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center text-muted-foreground py-8">
                    {searchTerm ? '未找到匹配的网站' : '暂无网站数据'}
                  </TableCell>
                </TableRow>
              ) : (
                filteredWebsites.map((website) => (
                  <TableRow key={website.id}>
                    {/* 域名列 - 定宽，支持多个域名显示，鼠标悬停显示完整内容 */}
                    <TableCell className="w-[280px]">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <div className="font-medium max-w-[260px] truncate cursor-help">
                            {website.domains.join(', ')}
                          </div>
                        </TooltipTrigger>
                        <TooltipContent>
                          <div className="max-w-xs">
                            {website.domains.map((domain, index) => (
                              <div key={index}>{domain}</div>
                            ))}
                          </div>
                        </TooltipContent>
                      </Tooltip>
                    </TableCell>
                    
                    {/* 状态列 */}
                    <TableCell className="w-[100px]">
                      <Badge className={statusColors[website.status]}>
                        {statusLabels[website.status]}
                      </Badge>
                    </TableCell>
                    
                    {/* CNAME列 - 定宽，鼠标悬停显示完整内容 */}
                    <TableCell className="w-[200px]">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <div className="font-mono text-xs max-w-[180px] truncate cursor-help">
                            {website.cname}
                          </div>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p className="max-w-xs break-all">{website.cname}</p>
                        </TooltipContent>
                      </Tooltip>
                    </TableCell>
                    
                    {/* SSL状态列 */}
                    <TableCell className="w-[90px]">
                      <Badge className={sslStatusColors[website.sslStatus]}>
                        {sslStatusLabels[website.sslStatus]}
                      </Badge>
                    </TableCell>
                    
                    <TableCell>
                      <Badge variant="outline">{website.routeGroupName}</Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{website.permissionGroupName}</Badge>
                    </TableCell>
                    <TableCell className="text-right w-[200px]">
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 px-2"
                          onClick={() => handleEdit(website.id)}
                        >
                          <Edit2 className="w-3 h-3 mr-1" />
                          编辑
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 px-2"
                          onClick={() => handleClearCache(website)}
                        >
                          <RefreshCw className="w-3 h-3 mr-1" />
                          缓存
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-8 px-2"
                          onClick={() => {
                            setSelectedWebsite(website);
                            setIsDeleteDialogOpen(true);
                          }}
                        >
                          <Trash2 className="w-3 h-3 mr-1" />
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

      {/* 删除确认对话框 */}
      <Dialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>
              确定要删除网站 <span className="font-semibold">{selectedWebsite?.domains[0]}</span> 吗？此操作无法撤销。
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
              确定要清理网站 <span className="font-semibold">{selectedWebsite?.domains[0]}</span> 的缓存吗？
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
