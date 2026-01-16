import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { ArrowLeft, Save } from 'lucide-react';
import { toast } from 'sonner';
import { useLocation } from 'wouter';

/**
 * Website type
 * 对应Go后端的 Website 结构体
 */
interface Website {
  id: number;
  domains: string[];
  cname: string;
  sslStatus: 'valid' | 'expired' | 'none';
  routeGroupId: number;
  routeGroupName: string;
  permissionGroupId: number;
  permissionGroupName: string;
  status: 'active' | 'inactive' | 'maintenance';
  createdAt: string;
  updatedAt: string;
}

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

// Mock数据 - 用于编辑时获取网站信息
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
];

export default function WebsiteEdit() {
  const [location, setLocation] = useLocation();
  const [loading, setLoading] = useState(true);
  const [domainsText, setDomainsText] = useState(''); // 域名文本（每行一个）
  const [cname, setCname] = useState('');
  const [routeGroupId, setRouteGroupId] = useState<number>(1);
  const [permissionGroupId, setPermissionGroupId] = useState<number>(1);
  const [status, setStatus] = useState<Website['status']>('active');

  // 从路由参数中获取网站ID
  const pathParts = location.split('/');
  const websiteId = pathParts[2] === 'new' ? null : parseInt(pathParts[2]);
  const isNew = websiteId === null;

  /**
   * TODO: 对接Go API
   * GET /api/v1/websites/:id - 获取网站详情
   * POST /api/v1/websites - 创建网站
   * PUT /api/v1/websites/:id - 更新网站
   */

  useEffect(() => {
    const loadData = async () => {
      setLoading(true);
      
      if (!isNew && websiteId) {
        // TODO: 调用Go API获取网站详情
        // const response = await fetch(`/api/v1/websites/${websiteId}`);
        // const website = await response.json();
        
        // 模拟网络延迟
        await new Promise(resolve => setTimeout(resolve, 300));
        
        const website = mockWebsites.find(w => w.id === websiteId);
        if (website) {
          setDomainsText(website.domains.join('\n'));
          setCname(website.cname);
          setRouteGroupId(website.routeGroupId);
          setPermissionGroupId(website.permissionGroupId);
          setStatus(website.status);
        } else {
          toast.error('网站不存在');
          setLocation('/website/list');
        }
      }
      
      setLoading(false);
    };
    
    loadData();
  }, [websiteId, isNew, setLocation]);

  const handleSave = async () => {
    // 验证域名
    const domains = domainsText.split('\n').map(d => d.trim()).filter(d => d);
    if (domains.length === 0) {
      toast.error('请至少输入一个域名');
      return;
    }

    // 域名格式验证
    const domainRegex = /^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/;
    const invalidDomains = domains.filter(d => !domainRegex.test(d));
    if (invalidDomains.length > 0) {
      toast.error(`以下域名格式不正确：${invalidDomains.join(', ')}`);
      return;
    }

    // 验证CNAME
    if (!cname.trim()) {
      toast.error('请输入CNAME');
      return;
    }

    const data = {
      domains,
      cname: cname.trim(),
      routeGroupId,
      permissionGroupId,
      status,
    };

    if (isNew) {
      // TODO: 调用Go API创建网站
      // const response = await fetch('/api/v1/websites', {
      //   method: 'POST',
      //   headers: { 'Content-Type': 'application/json' },
      //   body: JSON.stringify(data),
      // });
      // const newWebsite = await response.json();
      
      toast.success('网站创建成功');
    } else {
      // TODO: 调用Go API更新网站
      // await fetch(`/api/v1/websites/${websiteId}`, {
      //   method: 'PUT',
      //   headers: { 'Content-Type': 'application/json' },
      //   body: JSON.stringify(data),
      // });
      
      toast.success('网站更新成功');
    }

    setLocation('/website/list');
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-muted-foreground">加载中...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => setLocation('/website/list')}
        >
          <ArrowLeft className="w-4 h-4" />
        </Button>
        <div>
          <h1 className="text-2xl font-bold">{isNew ? '添加网站' : '编辑网站'}</h1>
          <p className="text-muted-foreground mt-1">
            {isNew ? '创建新的网站配置' : '修改网站信息'}
          </p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>基本信息</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="space-y-2">
            <Label htmlFor="domains">
              域名 * <span className="text-muted-foreground text-sm">（每行一个域名）</span>
            </Label>
            <Textarea
              id="domains"
              placeholder="例如：&#10;www.example.com&#10;example.com&#10;m.example.com"
              value={domainsText}
              onChange={(e) => setDomainsText(e.target.value)}
              rows={6}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="cname">CNAME *</Label>
            <Input
              id="cname"
              placeholder="例如：cdn.example.com.cdn.cloudflare.net"
              value={cname}
              onChange={(e) => setCname(e.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="routeGroup">线路组 *</Label>
              <Select
                value={routeGroupId.toString()}
                onValueChange={(value) => setRouteGroupId(parseInt(value))}
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
                value={permissionGroupId.toString()}
                onValueChange={(value) => setPermissionGroupId(parseInt(value))}
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
              value={status}
              onValueChange={(value) => setStatus(value as Website['status'])}
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
        </CardContent>
      </Card>

      <div className="flex justify-end gap-4">
        <Button variant="outline" onClick={() => setLocation('/website/list')}>
          取消
        </Button>
        <Button onClick={handleSave}>
          <Save className="w-4 h-4 mr-2" />
          {isNew ? '创建' : '保存'}
        </Button>
      </div>
    </div>
  );
}
