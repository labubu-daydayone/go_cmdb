import React, { createContext, useContext, useState, ReactNode } from 'react';
import { useLocation } from 'wouter';
import { X } from 'lucide-react';
import { Button } from '@/components/ui/button';

export interface Tab {
  id: string;
  title: string;
  path: string;
  closable: boolean;
}

interface TabContextType {
  tabs: Tab[];
  activeTabId: string;
  addTab: (tab: Omit<Tab, 'id'>) => void;
  removeTab: (id: string) => void;
  setActiveTab: (id: string) => void;
}

const TabContext = createContext<TabContextType | undefined>(undefined);

export const useTabManager = () => {
  const context = useContext(TabContext);
  if (!context) {
    throw new Error('useTabManager must be used within TabProvider');
  }
  return context;
};

interface TabProviderProps {
  children: ReactNode;
}

export const TabProvider: React.FC<TabProviderProps> = ({ children }) => {
  const [, setLocation] = useLocation();
  const [tabs, setTabs] = useState<Tab[]>([
    { id: 'dashboard', title: '仪表板', path: '/', closable: false }
  ]);
  const [activeTabId, setActiveTabId] = useState('dashboard');

  const addTab = (tab: Omit<Tab, 'id'>) => {
    const id = tab.path.replace(/\//g, '-').substring(1) || 'home';
    const existingTab = tabs.find(t => t.id === id);
    
    if (existingTab) {
      setActiveTabId(id);
    } else {
      setTabs(prev => [...prev, { ...tab, id }]);
      setActiveTabId(id);
    }
  };

  const removeTab = (id: string) => {
    const tab = tabs.find(t => t.id === id);
    if (!tab || !tab.closable) return;

    const index = tabs.findIndex(t => t.id === id);
    const newTabs = tabs.filter(t => t.id !== id);
    setTabs(newTabs);

    if (activeTabId === id) {
      const newActiveTab = newTabs[Math.max(0, index - 1)];
      setActiveTabId(newActiveTab.id);
      setLocation(newActiveTab.path);
    }
  };

  const setActiveTab = (id: string) => {
    const tab = tabs.find(t => t.id === id);
    if (tab) {
      setActiveTabId(id);
      setLocation(tab.path);
    }
  };

  return (
    <TabContext.Provider value={{ tabs, activeTabId, addTab, removeTab, setActiveTab }}>
      {children}
    </TabContext.Provider>
  );
};

export const TabBar: React.FC = () => {
  const { tabs, activeTabId, removeTab, setActiveTab } = useTabManager();

  return (
    <div className="flex items-center gap-1 px-4 py-2 bg-background border-b overflow-x-auto">
      {tabs.map(tab => (
        <div
          key={tab.id}
          className={`
            flex items-center gap-2 px-3 py-1.5 rounded-t-md cursor-pointer
            transition-colors min-w-[100px] max-w-[200px]
            ${activeTabId === tab.id 
              ? 'bg-primary text-primary-foreground' 
              : 'bg-muted hover:bg-muted/80'
            }
          `}
          onClick={() => setActiveTab(tab.id)}
        >
          <span className="text-sm truncate flex-1">{tab.title}</span>
          {tab.closable && (
            <Button
              variant="ghost"
              size="sm"
              className="h-4 w-4 p-0 hover:bg-transparent"
              onClick={(e) => {
                e.stopPropagation();
                removeTab(tab.id);
              }}
            >
              <X className="h-3 w-3" />
            </Button>
          )}
        </div>
      ))}
    </div>
  );
};
