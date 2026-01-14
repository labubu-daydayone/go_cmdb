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
import { Plus, Trash2, RefreshCw, CheckCircle2, XCircle, AlertCircle, Copy } from 'lucide-react';
import { toast } from 'sonner';

/**
 * Domain type
 * 对应Go后端的 Domain 结构体
 */
interface Domain {
  id: number;
  domainName: string;
  dnsProvider: string;
  source: 'auto_sync' | 'manual';
  nsStatus: 'pending' | 'active' | 'failed' | 'unknown';
  nsRecords?: string[];
  expireDate?: string;
  createdAt: string;
}

// Mock数据
const mockDomains: Domain[] = [
  {
    id: 1,
    domainName: 'example.com',
    dnsProvider: 'Cloudflare主账号',
    source: 'auto_sync',
    nsStatus: 'active',
    nsRecords: ['ns1.cloudflare.com', 'ns2.cloudflare.com'],
    expireDate: '2025-06-15',
    createdAt: '2024-01-01',
  },
  {
    id: 2,
    domainName: 'test.net',
    dnsProvider: 'Cloudflare主账号',
    source: 'manual',
    nsStatus: 'pending',
    nsRecords: ['ns1.cloudflare.com', 'ns2.cloudflare.com'],
    createdAt: '2024-01-10',
  },
  {
    id: 3,
    domainName: 'demo.org',
    dnsProvider: 'GoDaddy备用账号',
    source: 'auto_sync',
    nsStatus: 'failed',
    createdAt: '2024-01-05',
  },
];

const mockDnsAccounts = [
  { id: 1, name: 'Cloudflare主账号', provider: 'cloudflare' },
  { id: 2, name: 'GoDaddy备用账号', provider: 'godaddy' },
  { id: 3, name: '阿里云DNS', provider: 'aliyun' },
];

const nsStatusConfig = {
  pending: { label: '待配置', icon: AlertCircle, color: 'bg-yellow-100 text-yellow-800' },
  active: { label: 'NS已生效', icon: CheckCircle2, color: 'bg-green-100 text-green-800' },
  failed: { label: 'NS失败', icon: XCircle, color: 'bg-red-100 text-red-800' },
  unknown: { label: '未知', icon: AlertCircle, color: 'bg-gray-100 text-gray-800' },
};

export default function DomainList() {
  const [domains, setDomains] = useState<Domain[]>(mockDomains);
  const [isAddDialogOpen, setIsAddDialogOpen] = useState(false);
  const [isNsDialogOpen, setIsNsDialogOpen] = useState(false);
  const [selectedDomain, setSelectedDomain] = useState<Domain | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [formData, setFormData] = useState({
    domainName: '',
    dnsAccountId: '',
  });

  /**
   * TODO: 对接Go API
   * GET /api/v1/domains - 获取域名列表
   * POST /api/v1/domains - 手动添加域名
   * DELETE /api/v1/domains/:id - 删除域名
   * POST /api/v1/domains/sync - 同步所有账号的域名
   * GET /api/v1/domains/:id/ns-records - 获取域名NS记录
   * POST /api/v1/domains/:id/check-ns - 检查NS状态
   */

  const handleAddDomain = async () => {
    if (!formData.domainName || !formData.dnsAccountId) {
      toast.error('请填写所有必填项');
      return;
    }

    // TODO: 调用Go API添加域名
    // const response = await fetch('/api/v1/domains', {
    //   method: 'POST',
    //   body: JSON.stringify(formData),
    // });
    // const data = await response.json();
    // 返回的data应包含nsRecords

    const selectedAccount = mockDnsAccounts.find(acc => acc.id.toString() === formData.dnsAccountId);
    const newDomain: Domain = {
      id: Date.now(),
      domainName: formData.domainName,
      dnsProvider: selectedAccount?.name || '',
      source: 'manual',
      nsStatus: 'pending',
      nsRecords: ['ns1.cloudflare.com', 'ns2.cloudflare.com'], // 模拟返回的NS记录
      createdAt: new Date().toISOString().split('T')[0],
    };

    setDomains([newDomain, ...domains]);
    setIsAddDialogOpen(false);
    
    // 显示NS配置指引
    setSelectedDomain(newDomain);
    setIsNsDialogOpen(true);
    
    setFormData({ domainName: '', dnsAccountId: '' });
    toast.success('域名添加成功，请配置NS记录');
  };

  const handleDeleteDomain = (id: number) => {
    // TODO: 调用Go API删除域名
    // await fetch(`/api/v1/domains/${id}`, { method: 'DELETE' });
    
    setDomains(domains.filter(d => d.id !== id));
    toast.success('域名已删除');
  };

  const handleSyncAll = () => {
    // TODO: 调用Go API同步所有域名
    // await fetch('/api/v1/domains/sync', { method: 'POST' });
    
    toast.success('正在同步所有账号的域名...');
  };

  const handleCheckNs = (domain: Domain) => {
    // TODO: 调用Go API检查NS状态
    // await fetch(`/api/v1/domains/${domain.id}/check-ns`, { method: 'POST' });
    
    toast.success(`正在检查 ${domain.domainName} 的NS状态...`);
  };

  const handleShowNs = (domain: Domain) => {
    setSelectedDomain(domain);
    setIsNsDialogOpen(true);
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    toast.success('已复制到剪贴板');
  };

  const filteredDomains = domains.filter(domain =>
    domain.domainName.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="p-6 space-y-6">
      {/* 页面标题和操作 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">域名列表</h1>
          <p className="text-muted-foreground mt-1">
            管理域名资产，支持自动同步和手动添加
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={handleSyncAll}>
            <RefreshCw className="w-4 h-4 mr-2" />
            同步域名
          </Button>
          <Button onClick={() => setIsAddDialogOpen(true)}>
            <Plus className="w-4 h-4 mr-2" />
            手动添加
          </Button>
        </div>
      </div>

      {/* 搜索栏 */}
      <div className="flex gap-4">
        <Input
          placeholder="搜索域名..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="max-w-sm"
        />
      </div>

      {/* 域名表格 */}
      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>域名</TableHead>
                <TableHead>DNS服务商</TableHead>
                <TableHead>来源</TableHead>
                <TableHead>NS状态</TableHead>
                <TableHead>过期时间</TableHead>
                <TableHead>添加时间</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredDomains.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center text-muted-foreground py-8">
                    暂无域名数据
                  </TableCell>
                </TableRow>
              ) : (
                filteredDomains.map((domain) => {
                  const StatusIcon = nsStatusConfig[domain.nsStatus].icon;
                  return (
                    <TableRow key={domain.id}>
                      <TableCell className="font-medium">{domain.domainName}</TableCell>
                      <TableCell>{domain.dnsProvider}</TableCell>
                      <TableCell>
                        <Badge variant={domain.source === 'auto_sync' ? 'default' : 'secondary'}>
                          {domain.source === 'auto_sync' ? '自动同步' : '手动添加'}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className={nsStatusConfig[domain.nsStatus].color}>
                          <StatusIcon className="w-3 h-3 mr-1" />
                          {nsStatusConfig[domain.nsStatus].label}
                        </Badge>
                      </TableCell>
                      <TableCell>{domain.expireDate || '-'}</TableCell>
                      <TableCell>{domain.createdAt}</TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-2">
                          {domain.nsRecords && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleShowNs(domain)}
                            >
                              查看NS
                            </Button>
                          )}
                          {domain.nsStatus !== 'active' && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleCheckNs(domain)}
                            >
                              检查NS
                            </Button>
                          )}
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-red-600 hover:text-red-700"
                            onClick={() => handleDeleteDomain(domain.id)}
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* 手动添加域名弹窗 */}
      <Dialog open={isAddDialogOpen} onOpenChange={setIsAddDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>手动添加域名</DialogTitle>
            <DialogDescription>
              添加域名后，系统将返回NS记录供您配置
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="domainName">域名 *</Label>
              <Input
                id="domainName"
                placeholder="example.com"
                value={formData.domainName}
                onChange={(e) => setFormData({ ...formData, domainName: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="dnsAccount">DNS服务商 *</Label>
              <Select
                value={formData.dnsAccountId}
                onValueChange={(value) => setFormData({ ...formData, dnsAccountId: value })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择DNS服务商账号" />
                </SelectTrigger>
                <SelectContent>
                  {mockDnsAccounts.map((account) => (
                    <SelectItem key={account.id} value={account.id.toString()}>
                      {account.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsAddDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleAddDomain}>确认添加</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* NS配置指引弹窗 */}
      <Dialog open={isNsDialogOpen} onOpenChange={setIsNsDialogOpen}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>NS记录配置指引</DialogTitle>
            <DialogDescription>
              请前往域名注册商修改NS记录为以下地址
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>域名</Label>
              <div className="flex items-center gap-2">
                <Input value={selectedDomain?.domainName || ''} readOnly />
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => copyToClipboard(selectedDomain?.domainName || '')}
                >
                  <Copy className="w-4 h-4" />
                </Button>
              </div>
            </div>
            <div className="space-y-2">
              <Label>NS记录</Label>
              {selectedDomain?.nsRecords?.map((ns, index) => (
                <div key={index} className="flex items-center gap-2">
                  <Input value={ns} readOnly />
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={() => copyToClipboard(ns)}
                  >
                    <Copy className="w-4 h-4" />
                  </Button>
                </div>
              ))}
            </div>
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 space-y-2">
              <h4 className="font-medium text-blue-900">配置步骤：</h4>
              <ol className="list-decimal list-inside space-y-1 text-sm text-blue-800">
                <li>登录您的域名注册商管理后台</li>
                <li>找到域名的DNS设置或NS记录管理</li>
                <li>将上述NS记录替换原有的NS记录</li>
                <li>保存更改并等待生效（通常需要24-48小时）</li>
                <li>返回本页面点击"检查NS"按钮验证配置</li>
              </ol>
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => setIsNsDialogOpen(false)}>我知道了</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
