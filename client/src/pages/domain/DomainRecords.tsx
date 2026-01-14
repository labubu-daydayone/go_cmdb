import { useState, useEffect } from 'react';
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
import { Plus, Trash2, ArrowLeft, RefreshCw, Save, X, Upload, Check } from 'lucide-react';
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
  const [editingId, setEditingId] = useState<number | null>(null);
  const [isAdding, setIsAdding] = useState(false);
  const [isBatchImportOpen, setIsBatchImportOpen] = useState(false);
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [editForm, setEditForm] = useState<Partial<DnsRecord>>({});
  const [newRecord, setNewRecord] = useState<Partial<DnsRecord>>({
    type: 'A',
    name: '',
    value: '',
    ttl: 3600,
    priority: 10,
    proxied: false,
  });

  /**
   * WebSocket实时更新
   * TODO: 对接Go WebSocket服务
   * ws://localhost:8080/api/v1/ws/dns-records/:domainName
   * 
   * 消息格式：
   * {
   *   type: 'record_added' | 'record_updated' | 'record_deleted' | 'domain_status_changed',
   *   data: DnsRecord | { domainName: string, nsStatus: string }
   * }
   */
  useEffect(() => {
    // const ws = new WebSocket(`ws://localhost:8080/api/v1/ws/dns-records/example.com`);
    // 
    // ws.onmessage = (event) => {
    //   const message = JSON.parse(event.data);
    //   
    //   switch (message.type) {
    //     case 'record_added':
    //       setRecords(prev => [...prev, message.data]);
    //       toast.success('新增解析记录');
    //       break;
    //     case 'record_updated':
    //       setRecords(prev => prev.map(r => r.id === message.data.id ? message.data : r));
    //       toast.info('解析记录已更新');
    //       break;
    //     case 'record_deleted':
    //       setRecords(prev => prev.filter(r => r.id !== message.data.id));
    //       toast.info('解析记录已删除');
    //       break;
    //     case 'domain_status_changed':
    //       toast.info(`域名状态变更: ${message.data.nsStatus}`);
    //       break;
    //   }
    // };
    // 
    // ws.onerror = () => toast.error('WebSocket连接失败');
    // 
    // return () => ws.close();
  }, []);

  /**
   * TODO: 对接Go API
   * GET /api/v1/domains/:domain/records - 获取解析记录列表
   * POST /api/v1/domains/:domain/records - 创建解析记录
   * PUT /api/v1/domains/:domain/records/:id - 更新解析记录
   * DELETE /api/v1/domains/:domain/records/:id - 删除解析记录
   * POST /api/v1/domains/:domain/records/batch - 批量导入解析记录
   */

  const handleStartAdd = () => {
    setIsAdding(true);
    setNewRecord({
      type: 'A',
      name: '',
      value: '',
      ttl: 3600,
      priority: 10,
      proxied: false,
    });
  };

  const handleSaveNew = () => {
    if (!newRecord.type || !newRecord.name || !newRecord.value) {
      toast.error('请填写必填项');
      return;
    }

    // TODO: 调用Go API创建解析记录
    // const response = await fetch(`/api/v1/domains/${domainName}/records`, {
    //   method: 'POST',
    //   body: JSON.stringify(newRecord),
    // });

    const record: DnsRecord = {
      id: Date.now(),
      type: newRecord.type as DnsRecord['type'],
      name: newRecord.name,
      value: newRecord.value,
      ttl: newRecord.ttl || 3600,
      priority: newRecord.type === 'MX' ? newRecord.priority : undefined,
      proxied: newRecord.proxied,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };

    setRecords([record, ...records]);
    setIsAdding(false);
    toast.success('解析记录已添加');
  };

  const handleCancelAdd = () => {
    setIsAdding(false);
    setNewRecord({});
  };

  const handleStartEdit = (record: DnsRecord) => {
    setEditingId(record.id);
    setEditForm(record);
  };

  const handleSaveEdit = () => {
    if (!editForm.type || !editForm.name || !editForm.value) {
      toast.error('请填写必填项');
      return;
    }

    // TODO: 调用Go API更新解析记录
    // await fetch(`/api/v1/domains/${domainName}/records/${editingId}`, {
    //   method: 'PUT',
    //   body: JSON.stringify(editForm),
    // });

    setRecords(records.map(r => 
      r.id === editingId
        ? { ...r, ...editForm, updatedAt: new Date().toISOString() }
        : r
    ));
    setEditingId(null);
    setEditForm({});
    toast.success('解析记录已更新');
  };

  const handleCancelEdit = () => {
    setEditingId(null);
    setEditForm({});
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

  const handleBatchImport = () => {
    if (!uploadFile) {
      toast.error('请选择文件');
      return;
    }

    // TODO: 调用Go API批量导入解析记录
    // const formData = new FormData();
    // formData.append('file', uploadFile);
    // await fetch(`/api/v1/domains/${domainName}/records/batch`, {
    //   method: 'POST',
    //   body: formData,
    // });

    toast.success(`正在导入 ${uploadFile.name}...`);
    setIsBatchImportOpen(false);
    setUploadFile(null);
  };

  const renderEditableCell = (
    record: DnsRecord,
    field: keyof DnsRecord,
    isEditing: boolean,
    value: any,
    onChange: (value: any) => void
  ) => {
    if (!isEditing) {
      if (field === 'value') {
        return (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="font-mono text-sm max-w-md truncate block cursor-help">
                {value}
              </span>
            </TooltipTrigger>
            <TooltipContent>
              <p className="max-w-xs break-all">{value}</p>
            </TooltipContent>
          </Tooltip>
        );
      }
      return <span className="font-mono">{value || '-'}</span>;
    }

    if (field === 'type') {
      return (
        <Select value={value} onValueChange={onChange}>
          <SelectTrigger className="h-8">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="A">A</SelectItem>
            <SelectItem value="AAAA">AAAA</SelectItem>
            <SelectItem value="CNAME">CNAME</SelectItem>
            <SelectItem value="MX">MX</SelectItem>
            <SelectItem value="TXT">TXT</SelectItem>
            <SelectItem value="NS">NS</SelectItem>
            <SelectItem value="SRV">SRV</SelectItem>
          </SelectContent>
        </Select>
      );
    }

    if (field === 'ttl' || field === 'priority') {
      return (
        <Input
          type="number"
          value={value}
          onChange={(e) => onChange(parseInt(e.target.value))}
          className="h-8 w-24"
        />
      );
    }

    if (field === 'proxied') {
      return (
        <input
          type="checkbox"
          checked={value}
          onChange={(e) => onChange(e.target.checked)}
          className="w-4 h-4"
        />
      );
    }

    return (
      <Input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="h-8"
      />
    );
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
            <Tooltip>
              <TooltipTrigger asChild>
                <p className="text-muted-foreground mt-1 max-w-md truncate cursor-help">
                  example.com - 管理DNS解析记录
                </p>
              </TooltipTrigger>
              <TooltipContent>
                <p>example.com - 管理DNS解析记录</p>
              </TooltipContent>
            </Tooltip>
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setIsBatchImportOpen(true)}>
            <Upload className="w-4 h-4 mr-2" />
            批量导入
          </Button>
          <Button variant="outline" onClick={handleRefresh}>
            <RefreshCw className="w-4 h-4 mr-2" />
            刷新
          </Button>
          <Button onClick={handleStartAdd} disabled={isAdding}>
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
              {/* 新增记录行 */}
              {isAdding && (
                <TableRow className="bg-blue-50">
                  <TableCell>
                    <Select
                      value={newRecord.type}
                      onValueChange={(value) => setNewRecord({ ...newRecord, type: value as DnsRecord['type'] })}
                    >
                      <SelectTrigger className="h-8">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="A">A</SelectItem>
                        <SelectItem value="AAAA">AAAA</SelectItem>
                        <SelectItem value="CNAME">CNAME</SelectItem>
                        <SelectItem value="MX">MX</SelectItem>
                        <SelectItem value="TXT">TXT</SelectItem>
                        <SelectItem value="NS">NS</SelectItem>
                        <SelectItem value="SRV">SRV</SelectItem>
                      </SelectContent>
                    </Select>
                  </TableCell>
                  <TableCell>
                    <Input
                      placeholder="@ 或 www"
                      value={newRecord.name}
                      onChange={(e) => setNewRecord({ ...newRecord, name: e.target.value })}
                      className="h-8"
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      placeholder="192.0.2.1"
                      value={newRecord.value}
                      onChange={(e) => setNewRecord({ ...newRecord, value: e.target.value })}
                      className="h-8"
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      type="number"
                      value={newRecord.ttl}
                      onChange={(e) => setNewRecord({ ...newRecord, ttl: parseInt(e.target.value) })}
                      className="h-8 w-24"
                    />
                  </TableCell>
                  <TableCell>
                    {newRecord.type === 'MX' && (
                      <Input
                        type="number"
                        value={newRecord.priority}
                        onChange={(e) => setNewRecord({ ...newRecord, priority: parseInt(e.target.value) })}
                        className="h-8 w-20"
                      />
                    )}
                  </TableCell>
                  <TableCell>
                    {(newRecord.type === 'A' || newRecord.type === 'AAAA' || newRecord.type === 'CNAME') && (
                      <input
                        type="checkbox"
                        checked={newRecord.proxied}
                        onChange={(e) => setNewRecord({ ...newRecord, proxied: e.target.checked })}
                        className="w-4 h-4"
                      />
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-2">
                      <Button variant="ghost" size="icon" onClick={handleSaveNew}>
                        <Check className="w-4 h-4 text-green-600" />
                      </Button>
                      <Button variant="ghost" size="icon" onClick={handleCancelAdd}>
                        <X className="w-4 h-4 text-red-600" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              )}

              {/* 记录列表 */}
              {records.map((record) => {
                const isEditing = editingId === record.id;
                const currentData = isEditing ? editForm : record;

                return (
                  <TableRow
                    key={record.id}
                    onDoubleClick={() => !isEditing && handleStartEdit(record)}
                    className={isEditing ? 'bg-yellow-50' : 'cursor-pointer hover:bg-gray-50'}
                    title={isEditing ? '' : '双击编辑'}
                  >
                    <TableCell>
                      {isEditing ? (
                        renderEditableCell(record, 'type', true, currentData.type, (value) =>
                          setEditForm({ ...editForm, type: value })
                        )
                      ) : (
                        <Badge className={recordTypeColors[record.type]}>
                          {record.type}
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      {renderEditableCell(record, 'name', isEditing, currentData.name, (value) =>
                        setEditForm({ ...editForm, name: value })
                      )}
                    </TableCell>
                    <TableCell>
                      {renderEditableCell(record, 'value', isEditing, currentData.value, (value) =>
                        setEditForm({ ...editForm, value: value })
                      )}
                    </TableCell>
                    <TableCell>
                      {renderEditableCell(record, 'ttl', isEditing, currentData.ttl, (value) =>
                        setEditForm({ ...editForm, ttl: value })
                      )}
                      {!isEditing && 's'}
                    </TableCell>
                    <TableCell>
                      {record.type === 'MX' ? (
                        renderEditableCell(record, 'priority', isEditing, currentData.priority, (value) =>
                          setEditForm({ ...editForm, priority: value })
                        )
                      ) : (
                        '-'
                      )}
                    </TableCell>
                    <TableCell>
                      {record.proxied !== undefined && (
                        isEditing ? (
                          renderEditableCell(record, 'proxied', true, currentData.proxied, (value) =>
                            setEditForm({ ...editForm, proxied: value })
                          )
                        ) : (
                          <Badge variant={record.proxied ? 'default' : 'outline'}>
                            {record.proxied ? '已代理' : '仅DNS'}
                          </Badge>
                        )
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        {isEditing ? (
                          <>
                            <Button variant="ghost" size="icon" onClick={handleSaveEdit}>
                              <Save className="w-4 h-4 text-green-600" />
                            </Button>
                            <Button variant="ghost" size="icon" onClick={handleCancelEdit}>
                              <X className="w-4 h-4 text-red-600" />
                            </Button>
                          </>
                        ) : (
                          <Button
                            variant="ghost"
                            size="icon"
                            className="text-red-600 hover:text-red-700 hover:bg-red-50"
                            onClick={() => handleDeleteRecord(record.id)}
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* 批量导入弹窗 */}
      <Dialog open={isBatchImportOpen} onOpenChange={setIsBatchImportOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>批量导入解析记录</DialogTitle>
            <DialogDescription>
              上传CSV或Excel文件批量添加DNS解析记录
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>选择文件</Label>
              <Input
                type="file"
                accept=".csv,.xlsx,.xls"
                onChange={(e) => setUploadFile(e.target.files?.[0] || null)}
              />
              <p className="text-sm text-muted-foreground">
                支持CSV、Excel格式，文件格式：类型,名称,值,TTL,优先级,代理状态
              </p>
            </div>
            <div className="bg-muted p-3 rounded text-sm">
              <p className="font-medium mb-2">示例格式：</p>
              <pre className="text-xs">
                A,www,192.0.2.1,3600,,true{'\n'}
                CNAME,blog,example.com,3600,,false{'\n'}
                MX,@,mail.example.com,3600,10,
              </pre>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsBatchImportOpen(false)}>
              取消
            </Button>
            <Button onClick={handleBatchImport}>开始导入</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
