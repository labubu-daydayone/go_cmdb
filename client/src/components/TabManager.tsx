import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { useLocation } from 'wouter';
import { X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  horizontalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

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
  reorderTabs: (oldIndex: number, newIndex: number) => void;
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

const MAX_TABS = 10;
const STORAGE_KEY = 'cmdb_tabs';
const ACTIVE_TAB_KEY = 'cmdb_active_tab';

export const TabProvider: React.FC<TabProviderProps> = ({ children }) => {
  const [, setLocation] = useLocation();
  
  // 从 localStorage 加载标签页
  const loadTabs = (): Tab[] => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved) {
        return JSON.parse(saved);
      }
    } catch (error) {
      console.error('Failed to load tabs from localStorage:', error);
    }
    return [{ id: 'dashboard', title: '仪表板', path: '/', closable: false }];
  };

  const loadActiveTabId = (): string => {
    try {
      const saved = localStorage.getItem(ACTIVE_TAB_KEY);
      if (saved) {
        return saved;
      }
    } catch (error) {
      console.error('Failed to load active tab from localStorage:', error);
    }
    return 'dashboard';
  };

  const [tabs, setTabs] = useState<Tab[]>(loadTabs);
  const [activeTabId, setActiveTabId] = useState(loadActiveTabId);

  // 保存标签页到 localStorage
  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(tabs));
    } catch (error) {
      console.error('Failed to save tabs to localStorage:', error);
    }
  }, [tabs]);

  useEffect(() => {
    try {
      localStorage.setItem(ACTIVE_TAB_KEY, activeTabId);
    } catch (error) {
      console.error('Failed to save active tab to localStorage:', error);
    }
  }, [activeTabId]);

  const addTab = (tab: Omit<Tab, 'id'>) => {
    // 特殊处理：首页路径 '/' 使用固定ID 'dashboard'
    const id = tab.path === '/' ? 'dashboard' : tab.path.replace(/\//g, '-').substring(1);
    const existingTab = tabs.find(t => t.id === id);
    
    if (existingTab) {
      setActiveTabId(id);
    } else {
      // 检查标签页数量限制
      if (tabs.length >= MAX_TABS) {
        toast.error(`最多只能打开${MAX_TABS}个标签页，请关闭一些标签后再试`);
        return;
      }
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

  const reorderTabs = (oldIndex: number, newIndex: number) => {
    setTabs(prev => arrayMove(prev, oldIndex, newIndex));
  };

  return (
    <TabContext.Provider value={{ tabs, activeTabId, addTab, removeTab, setActiveTab, reorderTabs }}>
      {children}
    </TabContext.Provider>
  );
};

// 可拖拽的标签项组件
interface SortableTabProps {
  tab: Tab;
  isActive: boolean;
  onRemove: (id: string) => void;
  onActivate: (id: string) => void;
}

const SortableTab: React.FC<SortableTabProps> = ({ tab, isActive, onRemove, onActivate }) => {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: tab.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`
        flex items-center gap-2 px-3 py-1.5 rounded-t-md
        transition-colors min-w-[100px] max-w-[200px]
        ${isActive 
          ? 'bg-primary text-primary-foreground' 
          : 'bg-muted hover:bg-muted/80'
        }
      `}
    >
      <span 
        {...attributes}
        {...listeners}
        className="text-sm truncate flex-1 cursor-move"
        onClick={(e) => {
          // 点击切换标签，dnd-kit会自动区分点击和拖拽
          if (!isDragging) {
            onActivate(tab.id);
          }
        }}
      >
        {tab.title}
      </span>
      {tab.closable && (
        <Button
          variant="ghost"
          size="sm"
          className="h-4 w-4 p-0 hover:bg-transparent cursor-pointer"
          onClick={(e) => {
            e.stopPropagation();
            onRemove(tab.id);
          }}
        >
          <X className="h-3 w-3" />
        </Button>
      )}
    </div>
  );
};

export const TabBar: React.FC = () => {
  const { tabs, activeTabId, removeTab, setActiveTab, reorderTabs } = useTabManager();
  
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8, // 鼠标移动8px才触发拖拽，小于8px认为是点击
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;

    if (over && active.id !== over.id) {
      const oldIndex = tabs.findIndex(tab => tab.id === active.id);
      const newIndex = tabs.findIndex(tab => tab.id === over.id);
      reorderTabs(oldIndex, newIndex);
    }
  };

  return (
    <div className="flex items-center gap-1 px-4 py-2 bg-background border-b overflow-x-auto">
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={handleDragEnd}
      >
        <SortableContext
          items={tabs.map(tab => tab.id)}
          strategy={horizontalListSortingStrategy}
        >
          {tabs.map(tab => (
            <SortableTab
              key={tab.id}
              tab={tab}
              isActive={activeTabId === tab.id}
              onRemove={removeTab}
              onActivate={setActiveTab}
            />
          ))}
        </SortableContext>
      </DndContext>
    </div>
  );
};
