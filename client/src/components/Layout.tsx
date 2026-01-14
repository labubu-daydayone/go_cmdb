import React from 'react';
import { useLocation } from 'wouter';
import { useAuth } from '@/contexts/AuthContext';
import { Button } from '@/components/ui/button';
import { LogOut, ChevronLeft, ChevronRight, ChevronDown, ChevronUp } from 'lucide-react';
import { useState } from 'react';
import { TabBar, useTabManager } from './TabManager';

interface LayoutProps {
  children: React.ReactNode;
}

interface MenuItem {
  label: string;
  href?: string;
  closable?: boolean;
  children?: MenuItem[];
}

export default function Layout({ children }: LayoutProps) {
  const [, setLocation] = useLocation();
  const { user, logout } = useAuth();
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [expandedMenus, setExpandedMenus] = useState<string[]>(['domain', 'website', 'system']);

  const handleLogout = () => {
    logout();
    setLocation('/login');
  };

  const { addTab } = useTabManager();

  const navItems: MenuItem[] = [
    { label: '仪表板', href: '/', closable: false },
    {
      label: '域名管理',
      children: [
        { label: '域名列表', href: '/domain/list', closable: true },
        { label: '解析模版', href: '/domain/template', closable: true },
        { label: '账号分组', href: '/domain/group', closable: true },
        { label: '证书管理', href: '/domain/cert', closable: true },
      ],
    },
    {
      label: '网站管理',
      children: [
        { label: '网站列表', href: '/website/list', closable: true },
        { label: 'DNS配置', href: '/website/dns', closable: true },
        { label: '解析线路', href: '/website/line', closable: true },
        { label: '节点列表', href: '/website/nodes', closable: true },
        { label: '节点分组', href: '/website/node-group', closable: true },
        { label: '华为CDN', href: '/website/huawei-cdn', closable: true },
        { label: '回源列表', href: '/website/origin', closable: true },
      ],
    },
    {
      label: '系统设置',
      children: [
        { label: '用户管理', href: '/users', closable: true },
        { label: '权限设置', href: '/permissions', closable: true },
        { label: '密钥配置', href: '/system/keys', closable: true },
      ],
    },
  ];

  const toggleMenu = (label: string) => {
    setExpandedMenus(prev =>
      prev.includes(label)
        ? prev.filter(item => item !== label)
        : [...prev, label]
    );
  };

  const handleMenuClick = (item: MenuItem) => {
    if (item.href) {
      addTab({ title: item.label, path: item.href, closable: item.closable ?? true });
      setLocation(item.href);
    }
  };

  const getMenuId = (label: string) => {
    return label.toLowerCase().replace(/\s+/g, '-');
  };

  return (
    <div className="flex h-screen bg-gray-50 overflow-hidden">
      {/* 侧边栏 */}
      <aside
        className={`
          bg-gradient-to-b from-blue-600 to-blue-700 text-white
          transition-all duration-300 ease-in-out flex-shrink-0 flex flex-col
          ${sidebarCollapsed ? 'w-16' : 'w-64'}
        `}
      >
        <div className="p-6 flex-shrink-0">
          <div className="flex items-center justify-between">
            {!sidebarCollapsed && (
              <div>
                <h1 className="text-xl font-bold">CMDB</h1>
                <p className="text-blue-100 text-sm">运维管理系统</p>
              </div>
            )}
            <button
              onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
              className="p-1 hover:bg-blue-500 rounded ml-auto"
              title={sidebarCollapsed ? '展开侧边栏' : '折叠侧边栏'}
            >
              {sidebarCollapsed ? (
                <ChevronRight className="w-5 h-5" />
              ) : (
                <ChevronLeft className="w-5 h-5" />
              )}
            </button>
          </div>
        </div>

        <nav className="space-y-1 px-4 flex-1 overflow-y-auto">
          {navItems.map((item) => (
            <div key={item.label}>
              {item.children ? (
                // 有子菜单的项
                <div>
                  <button
                    onClick={() => !sidebarCollapsed && toggleMenu(getMenuId(item.label))}
                    className="w-full flex items-center justify-between px-4 py-2 rounded-lg hover:bg-blue-500 transition-colors"
                    title={item.label}
                  >
                    <span className="truncate">
                      {sidebarCollapsed ? item.label.charAt(0) : item.label}
                    </span>
                    {!sidebarCollapsed && (
                      expandedMenus.includes(getMenuId(item.label)) ? (
                        <ChevronUp className="w-4 h-4 flex-shrink-0" />
                      ) : (
                        <ChevronDown className="w-4 h-4 flex-shrink-0" />
                      )
                    )}
                  </button>
                  {!sidebarCollapsed && expandedMenus.includes(getMenuId(item.label)) && (
                    <div className="ml-4 mt-1 space-y-1">
                      {item.children.map((child) => (
                        <a
                          key={child.href}
                          href={child.href}
                          onClick={(e) => {
                            e.preventDefault();
                            handleMenuClick(child);
                          }}
                          className="block px-4 py-2 rounded-lg hover:bg-blue-500 transition-colors text-sm truncate"
                          title={child.label}
                        >
                          {child.label}
                        </a>
                      ))}
                    </div>
                  )}
                </div>
              ) : (
                // 没有子菜单的项
                <a
                  href={item.href}
                  onClick={(e) => {
                    e.preventDefault();
                    handleMenuClick(item);
                  }}
                  className="block px-4 py-2 rounded-lg hover:bg-blue-500 transition-colors truncate"
                  title={item.label}
                >
                  {sidebarCollapsed ? item.label.charAt(0) : item.label}
                </a>
              )}
            </div>
          ))}
        </nav>
      </aside>

      {/* 主内容区 */}
      <div className="flex-1 flex flex-col overflow-hidden w-full">
        {/* 顶部栏 */}
        <header className="bg-white border-b border-gray-200 shadow-sm">
          <div className="flex items-center justify-between h-16 px-6">
            <div className="flex items-center gap-4">
              {/* PC端不需要额外的标题 */}
            </div>

            <div className="flex items-center gap-4">
              <div className="text-right">
                <p className="text-sm font-medium">{user?.username}</p>
                <p className="text-xs text-gray-500">{user?.email}</p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={handleLogout}
                className="gap-2"
              >
                <LogOut className="w-4 h-4" />
                <span>退出</span>
              </Button>
            </div>
          </div>
        </header>

        {/* 标签栏 */}
        <TabBar />

        {/* 页面内容 */}
        <main className="flex-1 overflow-auto">
          <div className="container mx-auto p-8 max-w-7xl">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}
