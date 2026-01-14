import React from 'react';
import { useLocation } from 'wouter';
import { useAuth } from '@/contexts/AuthContext';
import { Button } from '@/components/ui/button';
import { LogOut, ChevronLeft, ChevronRight } from 'lucide-react';
import { useState } from 'react';
import { TabBar, useTabManager } from './TabManager';

interface LayoutProps {
  children: React.ReactNode;
}

export default function Layout({ children }: LayoutProps) {
  const [, setLocation] = useLocation();
  const { user, logout } = useAuth();
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  const handleLogout = () => {
    logout();
    setLocation('/login');
  };

  const { addTab } = useTabManager();

  const navItems = [
    { label: '仪表板', href: '/', closable: false },
    { label: '用户管理', href: '/users', closable: true },
    { label: '权限管理', href: '/permissions', closable: true },
  ];

  return (
    <div className="flex h-screen bg-gray-50 overflow-hidden">
      {/* 侧边栏 */}
      <aside
        className={`
          bg-gradient-to-b from-blue-600 to-blue-700 text-white
          transition-all duration-300 ease-in-out flex-shrink-0
          ${sidebarCollapsed ? 'w-16' : 'w-64'}
        `}
      >
        <div className="p-6">
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

        <nav className="space-y-2 px-4">
          {navItems.map((item) => (
            <a
              key={item.href}
              href={item.href}
              onClick={(e) => {
                e.preventDefault();
                addTab({ title: item.label, path: item.href, closable: item.closable });
                setLocation(item.href);
              }}
              className="block px-4 py-2 rounded-lg hover:bg-blue-500 transition-colors truncate"
              title={item.label}
            >
              {sidebarCollapsed ? item.label.charAt(0) : item.label}
            </a>
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
