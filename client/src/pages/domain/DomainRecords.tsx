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
import { Plus, Trash2, ArrowLeft, RefreshCw, Save, X, Upload, Check, Copy, CheckCheck } from 'lucide-react';
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

// Mock数据 - 生成更多数据用于测试分页
const generateMockRecords = (): DnsRecord[] => {
  const records: DnsRecord[] = [];
  const types: DnsRecord['type'][] = ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS'];
  const names = ['@', 'www', 'blog', 'mail', 'api', 'cdn', 'ftp', 'admin', 'shop', 'forum'];
  
  for (let i = 1; i <= 25; i++) {
    const type = types[Math.floor(Math.random() * types.length)];
    records.push({
      id: i,
      type,
      name: names[Math.floor(Math.random() * names.length)],
      value: type === 'A' ? `192.0.2.${i}` : type === 'CNAME' ? 'example.com' : `mail${i}.example.com`,
      ttl: [60, 300, 600, 1800, 3600][Math.floor(Math.random() * 5)],
      priority: type === 'MX' ? 10 : undefined,
      proxied: type === 'A' || type === 'AAAA' ? Math.random() > 0.5 : undefined,
      createdAt: '2024-01-01',
      updatedAt: '2024-01-01',
    });
  }
  return records;
};

const mockRecords = generateMockRecords();

const recordTypeColors = {
  A: 'bg-blue-100 text-blue-800',
  AAAA: 'bg-purple-100 text-purple-800',
  CNAME: 'bg-green-100 text-green-800',
  MX: 'bg-orange-100 text-orange-800',
  TXT: 'bg-gray-100 text-gray-800',
  NS: 'bg-yellow-100 text-yellow-800',
  SRV: 'bg-pink-100 text-pink-800',
};

// TTL固定选项
const TTL_OPTIONS = [
  { value: 60, label: '1分钟' },
  { value: 300, label: '5分钟' },
  { value: 600, label: '10分钟' },
  { value: 1800, label: '30分钟' },
  { value: 3600, label: '1小时' },
  { value: 7200, label: '2小时' },
  { value: 86400, label: '1天' },
];

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
  const [copiedId, setCopiedId] = useState<number | null>(null);

  // 分页状态
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const totalPages = Math.ceil(records.length / pageSize);
  const paginatedRecords = records.slice(
    (currentPage - 1) * pageSize,
    currentPage * pageSize
  );

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
   * GET /api/v1/domains/:domain/records?page=1&pageSize=20 - 获取解析记录列表（分页）
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

  const handleSaveNew = (continueAdding: boolean = false) => {
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
    
    if (continueAdding) {
      // 连续添加：保留类型和TTL，清空其他字段
      setNewRecord({
        type: newRecord.type,
        name: '',
        value: '',
        ttl: newRecord.ttl,
        priority: newRecord.type === 'MX' ? 10 : undefined,
        proxied: newRecord.proxied,
      });
      toast.success('解析记录已添加，继续添加下一条');
    } else {
      setIsAdding(false);
      toast.success('解析记录已添加');
    }
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
    // await fetch(`/api/v1/domains/${domainName}/records?page=${currentPage}&pageSize=${pageSize}`, { method: 'GET' });
    
    toast.success('正在刷新解析记录...');
  };

  const handleCopyValue = async (value: string, id: number) => {
    try {
      await navigator.clipboard.writeText(value);
      setCopiedId(id);
      setTimeout(() => setCopiedId(null), 2000);
      toast.success('已复制到剪贴板');
    } catch (error) {
      toast.error('复制失败');
    }
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
          <div className="flex items-center gap-2">
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
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              onClick={() => handleCopyValue(value, record.id)}
            >
              {copiedId === record.id ? (
                <CheckCheck className="w-3 h-3 text-green-600" />
              ) : (
                <Copy className="w-3 h-3" />
              )}
            </Button>
          </div>
        );
      }
      if (field === 'ttl') {
        const ttlOption = TTL_OPTIONS.find(opt => opt.value === value);
        return <span className="font-mono">{ttlOption?.label || `${value}s`}</span>;
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

    if (field === 'ttl') {
      return (
        <Select value={value.toString()} onValueChange={(v) => onChange(parseInt(v))}>
          <SelectTrigger className="h-8 w-28">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {TTL_OPTIONS.map(opt => (
              <SelectItem key={opt.value} value={opt.value.toString()}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      );
    }

    if (field === 'priority') {
      return (
        <Input
          type="number"
          value={value}
          onChange={(e) => onChange(parseInt(e.target.value))}
          className="h-8 w-20"
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
                    <Select
                      value={newRecord.ttl?.toString()}
                      onValueChange={(value) => setNewRecord({ ...newRecord, ttl: parseInt(value) })}
                    >
                      <SelectTrigger className="h-8 w-28">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {TTL_OPTIONS.map(opt => (
                          <SelectItem key={opt.value} value={opt.value.toString()}>
                            {opt.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
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
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button variant="ghost" size="icon" onClick={() => handleSaveNew(true)}>
                            <Plus className="w-4 h-4 text-blue-600" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>保存并继续添加</TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button variant="ghost" size="icon" onClick={() => handleSaveNew(false)}>
                            <Check className="w-4 h-4 text-green-600" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>保存</TooltipContent>
                      </Tooltip>
                      <Button variant="ghost" size="icon" onClick={handleCancelAdd}>
                        <X className="w-4 h-4 text-red-600" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              )}

              {/* 记录列表 */}
              {paginatedRecords.map((record) => {
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

          {/* 分页控件 */}
          <div className="flex items-center justify-between px-4 py-3 border-t">
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted-foreground">每页显示</span>
              <Select
                value={pageSize.toString()}
                onValueChange={(value) => {
                  setPageSize(parseInt(value));
                  setCurrentPage(1);
                }}
              >
                <SelectTrigger className="h-8 w-20">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="10">10</SelectItem>
                  <SelectItem value="20">20</SelectItem>
                  <SelectItem value="50">50</SelectItem>
                </SelectContent>
              </Select>
              <span className="text-sm text-muted-foreground">
                共 {records.length} 条记录
              </span>
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCurrentPage(1)}
                disabled={currentPage === 1}
              >
                首页
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCurrentPage(currentPage - 1)}
                disabled={currentPage === 1}
              >
                上一页
              </Button>
              <span className="text-sm">
                第 {currentPage} / {totalPages} 页
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCurrentPage(currentPage + 1)}
                disabled={currentPage === totalPages}
              >
                下一页
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCurrentPage(totalPages)}
                disabled={currentPage === totalPages}
              >
                末页
              </Button>
            </div>
          </div>
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
