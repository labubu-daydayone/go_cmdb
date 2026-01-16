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
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Checkbox } from '@/components/ui/checkbox';
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
  cname: string; // CNAME（由系统自动生成）
  originType: 'ip' | 'origin_list'; // 回源类型：IP或回源列表
  originValue: string; // 回源地址IP或回源列表ID
  sslStatus: 'valid' | 'expired' | 'none';
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
    cname: 'cdn-1.example.com.cdn.cloudflare.net',
    originType: 'ip',
    originValue: '192.168.1.100',
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
    originType: 'origin_list',
    originValue: '3',
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
    domains: ['shop.example.com', 'm.shop.example.com'],
    cname: 'shop.example.com.cdn.cloudflare.net',
    originType: 'ip',
    originValue: '192.168.1.200',
    sslStatus: 'expired',
    routeGroupId: 3,
    routeGroupName: '全球线路组',
    permissionGroupId: 3,
    permissionGroupName: '电商组',
    status: 'active',
    createdAt: '2024-01-03',
    updatedAt: '2024-01-17',
  },
  {
    id: 4,
    domains: ['blog.example.com'],
    cname: 'blog.example.com.cdn.cloudflare.net',
    originType: 'origin_list',
    originValue: '1',
    sslStatus: 'none',
    routeGroupId: 1,
    routeGroupName: '国内线路组',
    permissionGroupId: 1,
    permissionGroupName: '运维组',
    status: 'inactive',
    createdAt: '2024-01-04',
    updatedAt: '2024-01-18',
  },
  {
    id: 5,
    domains: ['test.example.com'],
    cname: 'test.example.com.cdn.cloudflare.net',
    originType: 'ip',
    originValue: '192.168.1.50',
    sslStatus: 'valid',
    routeGroupId: 2,
    routeGroupName: '海外线路组',
    permissionGroupId: 2,
    permissionGroupName: '开发组',
    status: 'maintenance',
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
  const [selectedWebsite, setSelectedWebsite] = useState<Website | null>(null);
  const [selectedIds, setSelectedIds] = useState<number[]>([]); // 多选的网站ID列表

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
    website.originValue.toLowerCase().includes(searchTerm.toLowerCase()) ||
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

  // 复制CNAME到剪贴板
  const handleCopyCname = async (cname: string) => {
    try {
      await navigator.clipboard.writeText(cname);
      toast.success('已复制CNAME到剪贴板');
    } catch (error) {
      toast.error('复制失败，请手动复制');
    }
  };

  // 多选框相关函数
  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedIds(filteredWebsites.map(w => w.id));
    } else {
      setSelectedIds([]);
    }
  };

  const handleSelectOne = (id: number, checked: boolean) => {
    if (checked) {
      setSelectedIds([...selectedIds, id]);
    } else {
      setSelectedIds(selectedIds.filter(selectedId => selectedId !== id));
    }
  };

  const isAllSelected = filteredWebsites.length > 0 && selectedIds.length === filteredWebsites.length;
  const isSomeSelected = selectedIds.length > 0 && selectedIds.length < filteredWebsites.length;

  // 删除功能（Popover气泡确认）
  const handleDelete = (website: Website) => {
    // TODO: 调用Go API删除网站
    // await fetch(`/api/v1/websites/${website.id}`, {
    //   method: 'DELETE',
    // });

    setWebsites(websites.filter(w => w.id !== website.id));
    toast.success('网站删除成功');
  };

  // 缓存清理功能（使用confirm）
  const handleClearCache = (website: Website) => {
    if (confirm(`确定要清理网站 "${website.domains[0]}" 的缓存吗？此操作将清除CDN节点上的所有缓存文件。`)) {
      // TODO: 调用Go API清理缓存
      // await fetch(`/api/v1/websites/${website.id}/clear-cache`, {
      //   method: 'POST',
      // });

      toast.success(`正在清理 ${website.domains[0]} 的缓存...`);
    }
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

          {/* 已选择数量显示 */}
          {selectedIds.length > 0 && (
            <div className="mb-4 flex items-center gap-2 text-sm text-muted-foreground">
              <span>已选择 {selectedIds.length} 个网站</span>
            </div>
          )}

          {/* 网站列表表格 */}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[50px]">
                  <Checkbox
                    checked={isAllSelected}
                    onCheckedChange={handleSelectAll}
                    aria-label="全选"
                  />
                </TableHead>
                <TableHead className="w-[240px]">域名</TableHead>
                <TableHead className="w-[90px]">状态</TableHead>
                <TableHead className="w-[200px]">CNAME</TableHead>
                <TableHead className="w-[120px]">回源地址</TableHead>
                <TableHead className="w-[80px]">SSL状态</TableHead>
                <TableHead>线路组</TableHead>
                <TableHead className="text-right w-[160px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredWebsites.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={9} className="text-center text-muted-foreground py-8">
                    {searchTerm ? '未找到匹配的网站' : '暂无网站数据'}
                  </TableCell>
                </TableRow>
              ) : (
                filteredWebsites.map((website) => (
                  <TableRow key={website.id}>
                    {/* 多选框列 */}
                    <TableCell className="w-[50px]">
                      <Checkbox
                        checked={selectedIds.includes(website.id)}
                        onCheckedChange={(checked) => handleSelectOne(website.id, checked as boolean)}
                        aria-label={`选择 ${website.domains[0]}`}
                      />
                    </TableCell>
                    
                    {/* 域名列 - 定宽，支持多个域名显示，鼠标悬停显示完整内容 */}
                    <TableCell className="w-[240px]">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <div className="font-medium max-w-[220px] truncate cursor-help">
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
                    <TableCell className="w-[90px]">
                      <Badge className={statusColors[website.status]}>
                        {statusLabels[website.status]}
                      </Badge>
                    </TableCell>
                    
                    {/* CNAME列 - 定宽，鼠标悬停显示完整内容 */}
                    <TableCell className="w-[180px]">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <div className="font-mono text-xs max-w-[160px] truncate cursor-help">
                            {website.cname}
                          </div>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p className="max-w-xs break-all">{website.cname}</p>
                        </TooltipContent>
                      </Tooltip>
                    </TableCell>
                    
                    {/* 回源地址列 - 定宽，鼠标悬停显示完整内容 */}
                    <TableCell className="w-[120px]">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <div className="font-mono text-xs max-w-[100px] truncate cursor-help">
                            {website.originType === 'ip' ? website.originValue : `列表#${website.originValue}`}
                          </div>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p className="max-w-xs break-all">
                            {website.originType === 'ip' ? `IP: ${website.originValue}` : `回源列表 ID: ${website.originValue}`}
                          </p>
                        </TooltipContent>
                      </Tooltip>
                    </TableCell>
                    
                    {/* SSL状态列 */}
                    <TableCell className="w-[80px]">
                      <Badge className={sslStatusColors[website.sslStatus]}>
                        {sslStatusLabels[website.sslStatus]}
                      </Badge>
                    </TableCell>
                    
                    <TableCell>
                      <Badge variant="outline">{website.routeGroupName}</Badge>
                    </TableCell>
                    <TableCell className="text-right w-[160px]">
                      <div className="flex justify-end gap-0.5">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 px-1.5 text-xs"
                          onClick={() => handleEdit(website.id)}
                        >
                          <Edit2 className="w-3 h-3 mr-0.5" />
                          编辑
                        </Button>
                        <Popover>
                          <PopoverTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-7 px-1.5 text-xs text-destructive hover:text-destructive"
                            >
                              <Trash2 className="w-3 h-3 mr-0.5" />
                              删除
                            </Button>
                          </PopoverTrigger>
                          <PopoverContent className="w-80">
                            <div className="space-y-3">
                              <div className="space-y-2">
                                <h4 className="font-medium leading-none">确认删除</h4>
                                <p className="text-sm text-muted-foreground">
                                  确定要删除网站 "{website.domains[0]}" 吗？此操作无法撤销。
                                </p>
                              </div>
                              <div className="flex justify-end gap-2">
                                <PopoverTrigger asChild>
                                  <Button variant="outline" size="sm">
                                    取消
                                  </Button>
                                </PopoverTrigger>
                                <PopoverTrigger asChild>
                                  <Button 
                                    variant="destructive" 
                                    size="sm"
                                    onClick={() => handleDelete(website)}
                                  >
                                    确认删除
                                  </Button>
                                </PopoverTrigger>
                              </div>
                            </div>
                          </PopoverContent>
                        </Popover>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 px-1.5 text-xs"
                          onClick={() => handleClearCache(website)}
                        >
                          <RefreshCw className="w-3 h-3 mr-0.5" />
                          缓存
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


    </div>
  );
}
