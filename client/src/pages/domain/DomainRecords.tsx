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
import { Plus, Trash2, Edit, ArrowLeft, RefreshCw } from 'lucide-react';
import { toast } from 'sonner';
import { useLocation } from 'wouter';

/**
 * DNS Record type
 * 对应Go后端的 DnsRecord 结构体
 */
interface DnsRecord {
  id: number;
  type: 'A' | 'AAAA' | 'CNAME' | 'MX' | 'TXT' | 'NS' | 'SRV';
  name: string;
  value: string;
  ttl: number;
  priority?: number;
  proxied?: boolean;
  createdAt: string;
  updatedAt: string;
}

// Mock数据
const mockRecords: DnsRecord[] = [
  {
    id: 1,
    type: 'A',
    name: '@',
    value: '192.0.2.1',
    ttl: 3600,
    proxied: true,
    createdAt: '2024-01-01',
    updatedAt: '2024-01-01',
  },
  {
    id: 2,
    type: 'A',
    name: 'www',
    value: '192.0.2.1',
    ttl: 3600,
    proxied: true,
    createdAt: '2024-01-01',
    updatedAt: '2024-01-01',
  },
  {
    id: 3,
    type: 'CNAME',
    name: 'blog',
    value: 'example.com',
    ttl: 3600,
    proxied: false,
    createdAt: '2024-01-05',
    updatedAt: '2024-01-05',
  },
  {
    id: 4,
    type: 'MX',
    name: '@',
    value: 'mail.example.com',
    ttl: 3600,
    priority: 10,
    createdAt: '2024-01-01',
    updatedAt: '2024-01-01',
  },
  {
    id: 5,
    type: 'TXT',
    name: '@',
    value: 'v=spf1 include:_spf.example.com ~all',
    ttl: 3600,
    createdAt: '2024-01-01',
    updatedAt: '2024-01-01',
  },
];

const recordTypeColors = {
  A: 'bg-blue-100 text-blue-800',
  AAAA: 'bg-purple-100 text-purple-800',
  CNAME: 'bg-green-100 text-green-800',
  MX: 'bg-orange-100 text-orange-800',
  TXT: 'bg-gray-100 text-gray-800',
  NS: 'bg-yellow-100 text-yellow-800',
  SRV: 'bg-pink-100 text-pink-800',
};

export default function DomainRecords() {
  const [, setLocation] = useLocation();
  const [records, setRecords] = useState<DnsRecord[]>(mockRecords);
  const [isAddDialogOpen, setIsAddDialogOpen] = useState(false);
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [selectedRecord, setSelectedRecord] = useState<DnsRecord | null>(null);
  const [formData, setFormData] = useState({
    type: '' as DnsRecord['type'] | '',
    name: '',
    value: '',
    ttl: 3600,
    priority: 10,
    proxied: false,
  });

  /**
   * TODO: 对接Go API
   * GET /api/v1/domains/:domain/records - 获取解析记录列表
   * POST /api/v1/domains/:domain/records - 创建解析记录
   * PUT /api/v1/domains/:domain/records/:id - 更新解析记录
   * DELETE /api/v1/domains/:domain/records/:id - 删除解析记录
   * 
   * 注意：需要调用DNS服务商API（Cloudflare/GoDaddy等）
   */

  const handleAddRecord = () => {
    if (!formData.type || !formData.name || !formData.value) {
      toast.error('请填写必填项');
      return;
    }

    // TODO: 调用Go API创建解析记录
    // const response = await fetch(`/api/v1/domains/${domainName}/records`, {
    //   method: 'POST',
    //   body: JSON.stringify(formData),
    // });

    const newRecord: DnsRecord = {
      id: Date.now(),
      type: formData.type as DnsRecord['type'],
      name: formData.name,
      value: formData.value,
      ttl: formData.ttl,
      priority: formData.type === 'MX' ? formData.priority : undefined,
      proxied: formData.proxied,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };

    setRecords([...records, newRecord]);
    setIsAddDialogOpen(false);
    setFormData({ type: '', name: '', value: '', ttl: 3600, priority: 10, proxied: false });
    toast.success('解析记录已添加');
  };

  const handleEditRecord = () => {
    if (!selectedRecord || !formData.type || !formData.name || !formData.value) {
      toast.error('请填写必填项');
      return;
    }

    // TODO: 调用Go API更新解析记录
    // await fetch(`/api/v1/domains/${domainName}/records/${selectedRecord.id}`, {
    //   method: 'PUT',
    //   body: JSON.stringify(formData),
    // });

    setRecords(records.map(r => 
      r.id === selectedRecord.id
        ? { ...r, ...formData, type: formData.type as DnsRecord['type'], updatedAt: new Date().toISOString() }
        : r
    ));
    setIsEditDialogOpen(false);
    setSelectedRecord(null);
    toast.success('解析记录已更新');
  };

  const handleDeleteRecord = (id: number) => {
    // TODO: 调用Go API删除解析记录
    // await fetch(`/api/v1/domains/${domainName}/records/${id}`, { method: 'DELETE' });
    
    setRecords(records.filter(r => r.id !== id));
    toast.success('解析记录已删除');
  };

  const handleRefresh = () => {
    // TODO: 调用Go API刷新解析记录
    // await fetch(`/api/v1/domains/${domainName}/records`, { method: 'GET' });
    
    toast.success('正在刷新解析记录...');
  };

  const openEditDialog = (record: DnsRecord) => {
    setSelectedRecord(record);
    setFormData({
      type: record.type,
      name: record.name,
      value: record.value,
      ttl: record.ttl,
      priority: record.priority || 10,
      proxied: record.proxied || false,
    });
    setIsEditDialogOpen(true);
  };

  return (
    <div className="p-6 space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setLocation('/domain/list')}
          >
            <ArrowLeft className="w-4 h-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold">解析记录管理</h1>
            <p className="text-muted-foreground mt-1">
              example.com - 管理DNS解析记录
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={handleRefresh}>
            <RefreshCw className="w-4 h-4 mr-2" />
            刷新
          </Button>
          <Button onClick={() => setIsAddDialogOpen(true)}>
            <Plus className="w-4 h-4 mr-2" />
            添加记录
          </Button>
        </div>
      </div>

      {/* 解析记录列表 */}
      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>类型</TableHead>
                <TableHead>名称</TableHead>
                <TableHead>值</TableHead>
                <TableHead>TTL</TableHead>
                <TableHead>优先级</TableHead>
                <TableHead>代理状态</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {records.map((record) => (
                <TableRow key={record.id}>
                  <TableCell>
                    <Badge className={recordTypeColors[record.type]}>
                      {record.type}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono">{record.name}</TableCell>
                  <TableCell className="font-mono text-sm max-w-md truncate">
                    {record.value}
                  </TableCell>
                  <TableCell>{record.ttl}s</TableCell>
                  <TableCell>{record.priority || '-'}</TableCell>
                  <TableCell>
                    {record.proxied !== undefined && (
                      <Badge variant={record.proxied ? 'default' : 'outline'}>
                        {record.proxied ? '已代理' : '仅DNS'}
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-2">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => openEditDialog(record)}
                      >
                        <Edit className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="text-red-600 hover:text-red-700 hover:bg-red-50"
                        onClick={() => handleDeleteRecord(record.id)}
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* 添加记录弹窗 */}
      <Dialog open={isAddDialogOpen} onOpenChange={setIsAddDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>添加解析记录</DialogTitle>
            <DialogDescription>
              为域名添加新的DNS解析记录
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="recordType">记录类型 *</Label>
              <Select
                value={formData.type}
                onValueChange={(value) => setFormData({ ...formData, type: value as DnsRecord['type'] })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择记录类型" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="A">A - IPv4地址</SelectItem>
                  <SelectItem value="AAAA">AAAA - IPv6地址</SelectItem>
                  <SelectItem value="CNAME">CNAME - 别名</SelectItem>
                  <SelectItem value="MX">MX - 邮件服务器</SelectItem>
                  <SelectItem value="TXT">TXT - 文本记录</SelectItem>
                  <SelectItem value="NS">NS - 域名服务器</SelectItem>
                  <SelectItem value="SRV">SRV - 服务记录</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="recordName">名称 *</Label>
              <Input
                id="recordName"
                placeholder="@ 或 www 或 subdomain"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="recordValue">值 *</Label>
              <Input
                id="recordValue"
                placeholder={formData.type === 'A' ? '192.0.2.1' : 'example.com'}
                value={formData.value}
                onChange={(e) => setFormData({ ...formData, value: e.target.value })}
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="recordTtl">TTL (秒)</Label>
                <Input
                  id="recordTtl"
                  type="number"
                  value={formData.ttl}
                  onChange={(e) => setFormData({ ...formData, ttl: parseInt(e.target.value) })}
                />
              </div>
              {formData.type === 'MX' && (
                <div className="space-y-2">
                  <Label htmlFor="recordPriority">优先级</Label>
                  <Input
                    id="recordPriority"
                    type="number"
                    value={formData.priority}
                    onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) })}
                  />
                </div>
              )}
            </div>
            {(formData.type === 'A' || formData.type === 'AAAA' || formData.type === 'CNAME') && (
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="recordProxied"
                  checked={formData.proxied}
                  onChange={(e) => setFormData({ ...formData, proxied: e.target.checked })}
                  className="w-4 h-4"
                />
                <Label htmlFor="recordProxied" className="cursor-pointer">
                  启用CDN代理（Cloudflare）
                </Label>
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsAddDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleAddRecord}>确认添加</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 编辑记录弹窗 */}
      <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>编辑解析记录</DialogTitle>
            <DialogDescription>
              修改DNS解析记录信息
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="editRecordType">记录类型 *</Label>
              <Select
                value={formData.type}
                onValueChange={(value) => setFormData({ ...formData, type: value as DnsRecord['type'] })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择记录类型" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="A">A - IPv4地址</SelectItem>
                  <SelectItem value="AAAA">AAAA - IPv6地址</SelectItem>
                  <SelectItem value="CNAME">CNAME - 别名</SelectItem>
                  <SelectItem value="MX">MX - 邮件服务器</SelectItem>
                  <SelectItem value="TXT">TXT - 文本记录</SelectItem>
                  <SelectItem value="NS">NS - 域名服务器</SelectItem>
                  <SelectItem value="SRV">SRV - 服务记录</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="editRecordName">名称 *</Label>
              <Input
                id="editRecordName"
                placeholder="@ 或 www 或 subdomain"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="editRecordValue">值 *</Label>
              <Input
                id="editRecordValue"
                placeholder={formData.type === 'A' ? '192.0.2.1' : 'example.com'}
                value={formData.value}
                onChange={(e) => setFormData({ ...formData, value: e.target.value })}
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="editRecordTtl">TTL (秒)</Label>
                <Input
                  id="editRecordTtl"
                  type="number"
                  value={formData.ttl}
                  onChange={(e) => setFormData({ ...formData, ttl: parseInt(e.target.value) })}
                />
              </div>
              {formData.type === 'MX' && (
                <div className="space-y-2">
                  <Label htmlFor="editRecordPriority">优先级</Label>
                  <Input
                    id="editRecordPriority"
                    type="number"
                    value={formData.priority}
                    onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) })}
                  />
                </div>
              )}
            </div>
            {(formData.type === 'A' || formData.type === 'AAAA' || formData.type === 'CNAME') && (
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="editRecordProxied"
                  checked={formData.proxied}
                  onChange={(e) => setFormData({ ...formData, proxied: e.target.checked })}
                  className="w-4 h-4"
                />
                <Label htmlFor="editRecordProxied" className="cursor-pointer">
                  启用CDN代理（Cloudflare）
                </Label>
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsEditDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleEditRecord}>保存修改</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
