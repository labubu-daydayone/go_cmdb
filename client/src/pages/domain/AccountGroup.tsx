import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
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
import { Plus, Trash2, RefreshCw, CheckCircle2, XCircle, AlertCircle } from 'lucide-react';
import { toast } from 'sonner';

/**
 * DNS Provider Account type
 * 对应Go后端的 DnsProviderAccount 结构体
 */
interface DnsProviderAccount {
  id: number;
  name: string;
  provider: 'cloudflare' | 'godaddy' | 'namecheap' | 'aliyun' | 'dnspod';
  email?: string;
  status: 'active' | 'inactive' | 'error';
  lastSyncAt?: string;
  createdAt: string;
}

// Mock数据 - 实际使用时需要从Go API获取
const mockAccounts: DnsProviderAccount[] = [
  {
    id: 1,
    name: 'Cloudflare主账号',
    provider: 'cloudflare',
    email: 'admin@example.com',
    status: 'active',
    lastSyncAt: '2024-01-15 10:30:00',
    createdAt: '2024-01-01 00:00:00',
  },
  {
    id: 2,
    name: 'GoDaddy备用账号',
    provider: 'godaddy',
    status: 'active',
    lastSyncAt: '2024-01-15 09:15:00',
    createdAt: '2024-01-05 00:00:00',
  },
  {
    id: 3,
    name: '阿里云DNS',
    provider: 'aliyun',
    status: 'error',
    createdAt: '2024-01-10 00:00:00',
  },
];

const providerLabels = {
  cloudflare: 'Cloudflare',
  godaddy: 'GoDaddy',
  namecheap: 'Namecheap',
  aliyun: '阿里云',
  dnspod: 'DNSPod',
};

const statusConfig = {
  active: { label: '正常', icon: CheckCircle2, color: 'text-green-600' },
  inactive: { label: '未激活', icon: AlertCircle, color: 'text-gray-600' },
  error: { label: '错误', icon: XCircle, color: 'text-red-600' },
};

export default function AccountGroup() {
  const [accounts, setAccounts] = useState<DnsProviderAccount[]>(mockAccounts);
  const [isAddDialogOpen, setIsAddDialogOpen] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    provider: '' as DnsProviderAccount['provider'] | '',
    email: '',
    apiKey: '',
    apiSecret: '',
  });

  /**
   * TODO: 对接Go API
   * GET /api/v1/dns-accounts - 获取账号列表
   * POST /api/v1/dns-accounts - 创建账号
   * DELETE /api/v1/dns-accounts/:id - 删除账号
   * POST /api/v1/dns-accounts/:id/sync - 同步账号域名
   * POST /api/v1/dns-accounts/:id/test - 测试账号连接
   */

  const handleAddAccount = () => {
    if (!formData.name || !formData.provider || !formData.apiKey) {
      toast.error('请填写必填项');
      return;
    }

    // TODO: 调用Go API创建账号
    // const response = await fetch('/api/v1/dns-accounts', {
    //   method: 'POST',
    //   body: JSON.stringify(formData),
    // });

    const newAccount: DnsProviderAccount = {
      id: Date.now(),
      name: formData.name,
      provider: formData.provider as DnsProviderAccount['provider'],
      email: formData.email || undefined,
      status: 'active',
      createdAt: new Date().toISOString(),
    };

    setAccounts([...accounts, newAccount]);
    setIsAddDialogOpen(false);
    setFormData({ name: '', provider: '', email: '', apiKey: '', apiSecret: '' });
    toast.success('账号添加成功');
  };

  const handleDeleteAccount = (id: number) => {
    // TODO: 调用Go API删除账号
    // await fetch(`/api/v1/dns-accounts/${id}`, { method: 'DELETE' });
    
    setAccounts(accounts.filter(acc => acc.id !== id));
    toast.success('账号已删除');
  };

  const handleSyncAccount = (id: number, name: string) => {
    // TODO: 调用Go API同步域名
    // await fetch(`/api/v1/dns-accounts/${id}/sync`, { method: 'POST' });
    
    toast.success(`正在同步 ${name} 的域名...`);
  };

  return (
    <div className="p-6 space-y-6" style={{paddingRight: '0px', paddingLeft: '0px'}}>
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">账号分组</h1>
          <p className="text-muted-foreground mt-1">
            管理DNS服务商账号，配置API密钥以实现域名自动同步
          </p>
        </div>
        <Button onClick={() => setIsAddDialogOpen(true)}>
          <Plus className="w-4 h-4 mr-2" />
          添加账号
        </Button>
      </div>

      {/* 账号列表 */}
      <div className="grid gap-4" style={{
        gridTemplateColumns: 'repeat(auto-fit, minmax(min(250px, 100%), 310px))'
      }}>
        {accounts.map((account) => {
          const StatusIcon = statusConfig[account.status].icon;
          return (
            <Card key={account.id}>
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <CardTitle className="text-lg">{account.name}</CardTitle>
                    <CardDescription className="mt-1">
                      {providerLabels[account.provider]}
                    </CardDescription>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="text-red-600 hover:text-red-700 hover:bg-red-50"
                    onClick={() => handleDeleteAccount(account.id)}
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              </CardHeader>
              <CardContent className="space-y-3">
                {/* 状态 */}
                <div className="flex items-center gap-2">
                  <StatusIcon className={`w-4 h-4 ${statusConfig[account.status].color}`} />
                  <span className="text-sm">{statusConfig[account.status].label}</span>
                </div>

                {/* 邮箱 */}
                {account.email && (
                  <div className="text-sm text-muted-foreground">
                    {account.email}
                  </div>
                )}

                {/* 最后同步时间 */}
                {account.lastSyncAt && (
                  <div className="text-xs text-muted-foreground">
                    最后同步: {account.lastSyncAt}
                  </div>
                )}

                {/* 操作按钮 */}
                <Button
                  variant="outline"
                  size="sm"
                  className="w-full"
                  onClick={() => handleSyncAccount(account.id, account.name)}
                >
                  <RefreshCw className="w-4 h-4 mr-2" />
                  同步域名
                </Button>
              </CardContent>
            </Card>
          );
        })}
      </div>

      {/* 添加账号弹窗 */}
      <Dialog open={isAddDialogOpen} onOpenChange={setIsAddDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>添加DNS服务商账号</DialogTitle>
            <DialogDescription>
              配置API密钥后，系统可以自动同步该账号下的域名
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="name">账号名称 *</Label>
              <Input
                id="name"
                placeholder="例如: Cloudflare主账号"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="provider">服务商 *</Label>
              <Select
                value={formData.provider}
                onValueChange={(value) =>
                  setFormData({ ...formData, provider: value as DnsProviderAccount['provider'] })
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择DNS服务商" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="cloudflare">Cloudflare</SelectItem>
                  <SelectItem value="godaddy">GoDaddy</SelectItem>
                  <SelectItem value="namecheap">Namecheap</SelectItem>
                  <SelectItem value="aliyun">阿里云</SelectItem>
                  <SelectItem value="dnspod">DNSPod</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="email">账号邮箱</Label>
              <Input
                id="email"
                type="email"
                placeholder="用于Cloudflare等服务商"
                value={formData.email}
                onChange={(e) => setFormData({ ...formData, email: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="apiKey">API Key *</Label>
              <Input
                id="apiKey"
                type="password"
                placeholder="输入API Key"
                value={formData.apiKey}
                onChange={(e) => setFormData({ ...formData, apiKey: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="apiSecret">API Secret</Label>
              <Input
                id="apiSecret"
                type="password"
                placeholder="部分服务商需要"
                value={formData.apiSecret}
                onChange={(e) => setFormData({ ...formData, apiSecret: e.target.value })}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsAddDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleAddAccount}>确认添加</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
