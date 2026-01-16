import React, { useState } from 'react';
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
import { Plus, Trash2, RefreshCw, CheckCircle2, XCircle, AlertCircle, Copy, Upload, ChevronDown, Settings, Edit2, ChevronRight, Check } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
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
  cloudflareStatus?: 'active' | 'paused' | 'pending' | 'unknown';
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
    cloudflareStatus: 'active',
  },
  {
    id: 2,
    domainName: 'test.net',
    dnsProvider: 'Cloudflare主账号',
    source: 'manual',
    nsStatus: 'pending',
    nsRecords: ['ns1.cloudflare.com', 'ns2.cloudflare.com'],
    createdAt: '2024-01-10',
    cloudflareStatus: 'paused',
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

// Cloudflare状态配置
const cloudflareStatusConfig = {
  active: { label: '活跃', color: 'bg-green-100 text-green-800' },
  paused: { label: '暂停', color: 'bg-yellow-100 text-yellow-800' },
  pending: { label: '待激活', color: 'bg-blue-100 text-blue-800' },
  unknown: { label: '未知', color: 'bg-gray-100 text-gray-800' },
};

export default function DomainList() {
  const [domains, setDomains] = useState<Domain[]>(mockDomains);
  const [isAddDialogOpen, setIsAddDialogOpen] = useState(false);
  const [isBatchImportDialogOpen, setIsBatchImportDialogOpen] = useState(false);
  const [isNsDialogOpen, setIsNsDialogOpen] = useState(false);
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [selectedDomain, setSelectedDomain] = useState<Domain | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [expandedDomainId, setExpandedDomainId] = useState<number | null>(null);
  const [copiedNsRecord, setCopiedNsRecord] = useState<string | null>(null);
  const [selectedDomainIds, setSelectedDomainIds] = useState<number[]>([]);
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

  const handleSyncAccount = (accountId: number, accountName: string) => {
    // TODO: 调用Go API同步指定账号的域名
    // await fetch(`/api/v1/dns-accounts/${accountId}/sync`, { method: 'POST' });
    
    toast.success(`正在同步 ${accountName} 的域名...`);
  };

  const handleBatchImport = () => {
    if (!uploadFile || !formData.dnsAccountId) {
      toast.error('请选择文件和DNS服务商');
      return;
    }

    // TODO: 调用Go API批量导入域名
    // const formData = new FormData();
    // formData.append('file', uploadFile);
    // formData.append('dnsAccountId', formData.dnsAccountId);
    // await fetch('/api/v1/domains/batch-import', {
    //   method: 'POST',
    //   body: formData,
    // });

    toast.success(`正在批量导入域名...`);
    setIsBatchImportDialogOpen(false);
    setUploadFile(null);
  };

  const handleCheckNs = (domain: Domain) => {
    // TODO: 调用Go API检查NS状态
    // await fetch(`/api/v1/domains/${domain.id}/check-ns`, { method: 'POST' });
    
    toast.success(`正在检查 ${domain.domainName} 的NS状态...`);
  };

  const handleManageRecords = (domain: Domain) => {
    // 访问控制：只有NS已生效的域名才能管理解析
    if (domain.nsStatus !== 'active') {
      toast.error('请先配置DNS服务商，等待NS记录生效后才能管理解析');
      return;
    }
    
    // 跳转到解析管理页面
    window.location.href = `/domain/${domain.domainName}/records`;
  };

  const handleUpdateNs = (domain: Domain) => {
    // TODO: 调用Go API修改注册商NS记录
    // 前提：域名必须在注册商账号下（source === 'auto_sync' && registrar_account_id != null）
    // await fetch(`/api/v1/domains/${domain.id}/update-ns`, {
    //   method: 'POST',
    //   body: JSON.stringify({ nsRecords: domain.nsRecords }),
    // });
    
    if (domain.source !== 'auto_sync') {
      toast.error('只有自动同步的域名才能修改注册商NS记录');
      return;
    }
    
    toast.success(`正在修改 ${domain.domainName} 的NS记录...`);
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
    <div className="p-6 space-y-6" style={{paddingRight: '0px', paddingLeft: '0px'}}>
      {/* 页面标题和操作 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">域名列表</h1>
          <p className="text-muted-foreground mt-1">
            管理域名资产，支持自动同步和手动添加
          </p>
        </div>
        <div className="flex gap-2">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline">
                <RefreshCw className="w-4 h-4 mr-2" />
                同步域名
                <ChevronDown className="w-4 h-4 ml-2" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {mockDnsAccounts.map((account) => (
                <DropdownMenuItem
                  key={account.id}
                  onClick={() => handleSyncAccount(account.id, account.name)}
                >
                  {account.name}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
          <Button variant="outline" onClick={() => setIsBatchImportDialogOpen(true)}>
            <Upload className="w-4 h-4 mr-2" />
            批量导入
          </Button>
          <Button onClick={() => setIsAddDialogOpen(true)}>
            <Plus className="w-4 h-4 mr-2" />
            手动添加
          </Button>
        </div>
      </div>

      {/* 搜索栏和已选中显示 */}
      <div className="flex items-center justify-between gap-4">
        <Input
          placeholder="搜索域名..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="max-w-sm"
        />
        {selectedDomainIds.length > 0 && (
          <div className="flex items-center gap-4">
            <span className="text-sm text-muted-foreground">
              已选中 {selectedDomainIds.length} 个域名
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setSelectedDomainIds([])}
            >
              取消选择
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => {
                toast.info('批量删除功能开发中...');
              }}
            >
              <Trash2 className="w-3 h-3 mr-1" />
              批量删除
            </Button>
          </div>
        )}
      </div>

      {/* 域名表格 */}
      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[50px]">
                  <input
                    type="checkbox"
                    checked={selectedDomainIds.length === filteredDomains.length && filteredDomains.length > 0}
                    onChange={(e) => {
                      if (e.target.checked) {
                        setSelectedDomainIds(filteredDomains.map(d => d.id));
                      } else {
                        setSelectedDomainIds([]);
                      }
                    }}
                    className="cursor-pointer"
                  />
                </TableHead>
                <TableHead className="w-[50px]"></TableHead>
                <TableHead className="w-[280px]">域名</TableHead>
                <TableHead>DNS服务商</TableHead>
                <TableHead>状态</TableHead>
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
                <>
                {filteredDomains.map((domain) => {
                  const isExpanded = expandedDomainId === domain.id;
                  const isSelected = selectedDomainIds.includes(domain.id);
                  return (
                    <React.Fragment key={domain.id}>
                      <TableRow>
                        <TableCell>
                          <input
                            type="checkbox"
                            checked={isSelected}
                            onChange={(e) => {
                              if (e.target.checked) {
                                setSelectedDomainIds([...selectedDomainIds, domain.id]);
                              } else {
                                setSelectedDomainIds(selectedDomainIds.filter(id => id !== domain.id));
                              }
                            }}
                            className="cursor-pointer"
                          />
                        </TableCell>
                        <TableCell>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-6 w-6 p-0"
                            onClick={() => setExpandedDomainId(isExpanded ? null : domain.id)}
                          >
                            <ChevronRight className={`w-4 h-4 transition-transform ${isExpanded ? 'rotate-90' : ''}`} />
                          </Button>
                        </TableCell>
                        <TableCell className="font-medium">{domain.domainName}</TableCell>
                        <TableCell>{domain.dnsProvider}</TableCell>
                        <TableCell>
                          {domain.cloudflareStatus ? (
                            <Badge variant="outline" className={cloudflareStatusConfig[domain.cloudflareStatus].color}>
                              {cloudflareStatusConfig[domain.cloudflareStatus].label}
                            </Badge>
                          ) : (
                            <span className="text-muted-foreground">-</span>
                          )}
                        </TableCell>
                        <TableCell>{domain.createdAt}</TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-2">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleManageRecords(domain)}
                            disabled={domain.nsStatus !== 'active'}
                            title={domain.nsStatus !== 'active' ? 'NS记录未生效，无法管理解析' : '管理DNS解析记录'}
                          >
                            <Edit2 className="w-4 h-4 mr-1" />
                            管理解析
                          </Button>
                          {domain.source === 'auto_sync' && domain.nsStatus !== 'active' && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleUpdateNs(domain)}
                              title="自动修改注册商NS记录"
                            >
                              修改NS
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
                      {isExpanded && (
                        <TableRow className="bg-muted/50">
                          <TableCell colSpan={7} className="py-4">
                            <div className="px-6 space-y-3">
                              {/* 第一行：2:2:4比例布局 */}
                              <div className="grid grid-cols-8 gap-6 text-sm">
                                {/* NS状态 - 占2份 */}
                                <div className="col-span-2 flex items-center gap-2">
                                  <span className="text-muted-foreground whitespace-nowrap">NS状态:</span>
                                  {(() => {
                                    const StatusIcon = nsStatusConfig[domain.nsStatus].icon;
                                    return (
                                      <Badge variant="outline" className={`${nsStatusConfig[domain.nsStatus].color} text-xs`}>
                                        <StatusIcon className="w-3 h-3 mr-1" />
                                        {nsStatusConfig[domain.nsStatus].label}
                                      </Badge>
                                    );
                                  })()}
                                </div>
                                
                                {/* 来源 - 占2份 */}
                                <div className="col-span-2 flex items-center gap-2">
                                  <span className="text-muted-foreground whitespace-nowrap">来源:</span>
                                  <Badge variant={domain.source === 'auto_sync' ? 'default' : 'secondary'} className="text-xs">
                                    {domain.source === 'auto_sync' ? '自动同步' : '手动添加'}
                                  </Badge>
                                </div>
                                
                                {/* 过期时间 - 占4份 */}
                                <div className="col-span-4 flex items-center gap-2">
                                  <span className="text-muted-foreground whitespace-nowrap">过期时间:</span>
                                  <span>{domain.expireDate || '未设置'}</span>
                                </div>
                              </div>
                              
                              {/* 第二行：NS记录占满整行 + 修改NS按钮 */}
                              {domain.nsRecords && domain.nsRecords.length > 0 && (
                                <div className="flex items-start gap-2 text-sm">
                                  <span className="text-muted-foreground whitespace-nowrap">NS记录:</span>
                                  <div className="flex flex-wrap gap-2 flex-1">
                                    {domain.nsRecords.map((ns, idx) => {
                                      const isCopied = copiedNsRecord === ns;
                                      return (
                                        <Badge
                                          key={idx}
                                          variant="outline"
                                          className="text-xs cursor-pointer hover:bg-muted/50 transition-colors flex items-center gap-1"
                                          onClick={() => {
                                            navigator.clipboard.writeText(ns);
                                            setCopiedNsRecord(ns);
                                            toast.success(`已复制: ${ns}`);
                                            setTimeout(() => setCopiedNsRecord(null), 2000);
                                          }}
                                        >
                                          {ns}
                                          {isCopied ? (
                                            <Check className="w-3 h-3 text-green-600" />
                                          ) : (
                                            <Copy className="w-3 h-3" />
                                          )}
                                        </Badge>
                                      );
                                    })}
                                  </div>
                                  <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => {
                                      toast.info('修改NS功能开发中...');
                                    }}
                                  >
                                    <Edit2 className="w-3 h-3 mr-1" />
                                    修改NS
                                  </Button>
                                </div>
                              )}
                            </div>
                          </TableCell>
                        </TableRow>
                      )}
                    </React.Fragment>
                  );
                })}
                </>
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

      {/* 批量导入弹窗 */}
      <Dialog open={isBatchImportDialogOpen} onOpenChange={setIsBatchImportDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>批量导入域名</DialogTitle>
            <DialogDescription>
              支持CSV/Excel文件，文件格式：每行一个域名
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="dnsAccountBatch">DNS服务商 *</Label>
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
            <div className="space-y-2">
              <Label htmlFor="uploadFile">上传文件 *</Label>
              <Input
                id="uploadFile"
                type="file"
                accept=".csv,.xlsx,.xls"
                onChange={(e) => setUploadFile(e.target.files?.[0] || null)}
              />
              {uploadFile && (
                <p className="text-sm text-muted-foreground">
                  已选择: {uploadFile.name}
                </p>
              )}
            </div>
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 space-y-2">
              <h4 className="font-medium text-blue-900">文件格式说明：</h4>
              <ul className="list-disc list-inside space-y-1 text-sm text-blue-800">
                <li>CSV文件：每行一个域名，例如：example.com</li>
                <li>Excel文件：第一列为域名，忽略表头</li>
                <li>最大支持1000个域名</li>
              </ul>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsBatchImportDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleBatchImport}>开始导入</Button>
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
