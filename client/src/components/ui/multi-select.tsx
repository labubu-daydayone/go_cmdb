import * as React from 'react';
import { X, Check, ChevronDown } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';

export interface MultiSelectOption {
  id: string;
  name: string;
  description?: string;
}

interface MultiSelectProps {
  options: MultiSelectOption[];
  selected: string[];
  onChange: (selected: string[]) => void;
  placeholder?: string;
  searchPlaceholder?: string;
  getOptionColor?: (index: number) => string;
}

export function MultiSelect({
  options,
  selected,
  onChange,
  placeholder = '请选择...',
  searchPlaceholder = '搜索...',
  getOptionColor,
}: MultiSelectProps) {
  const [open, setOpen] = React.useState(false);
  const [searchTerm, setSearchTerm] = React.useState('');
  const containerRef = React.useRef<HTMLDivElement>(null);

  // 点击外部关闭下拉框
  React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // 过滤选项
  const filteredOptions = React.useMemo(() => {
    if (!searchTerm) return options;
    const term = searchTerm.toLowerCase();
    return options.filter(
      (opt) =>
        opt.name.toLowerCase().includes(term) ||
        opt.description?.toLowerCase().includes(term)
    );
  }, [options, searchTerm]);

  // 获取已选择的选项
  const selectedOptions = React.useMemo(() => {
    return options.filter((opt) => selected.includes(opt.id));
  }, [options, selected]);

  // 切换选择
  const toggleOption = (optionId: string) => {
    if (selected.includes(optionId)) {
      onChange(selected.filter((id) => id !== optionId));
    } else {
      onChange([...selected, optionId]);
    }
  };

  // 移除选项
  const removeOption = (optionId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    onChange(selected.filter((id) => id !== optionId));
  };

  return (
    <div ref={containerRef} className="relative w-full">
      {/* 输入框区域 */}
      <div
        className={cn(
          'flex min-h-[40px] w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm cursor-pointer',
          open && 'ring-2 ring-ring ring-offset-2'
        )}
        onClick={() => setOpen(!open)}
      >
        <div className="flex flex-wrap gap-1 flex-1">
          {selectedOptions.length > 0 ? (
            selectedOptions.map((opt, index) => (
              <Badge
                key={opt.id}
                className={cn(
                  'cursor-pointer hover:opacity-80',
                  getOptionColor ? getOptionColor(index) : ''
                )}
                onClick={(e) => removeOption(opt.id, e)}
              >
                {opt.name}
                <X className="ml-1 h-3 w-3" />
              </Badge>
            ))
          ) : (
            <span className="text-muted-foreground">{placeholder}</span>
          )}
        </div>
        <ChevronDown
          className={cn(
            'h-4 w-4 opacity-50 transition-transform',
            open && 'transform rotate-180'
          )}
        />
      </div>

      {/* 下拉面板 */}
      {open && (
        <div className="absolute z-50 w-full mt-2 rounded-md border bg-popover text-popover-foreground shadow-md animate-in fade-in-0 zoom-in-95">
          <div className="p-2">
            <Input
              placeholder={searchPlaceholder}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="h-9"
              onClick={(e) => e.stopPropagation()}
            />
          </div>
          <div className="max-h-[300px] overflow-y-auto p-1">
            {filteredOptions.length > 0 ? (
              filteredOptions.map((option) => {
                const isSelected = selected.includes(option.id);
                return (
                  <div
                    key={option.id}
                    className={cn(
                      'relative flex cursor-pointer select-none items-center rounded-sm px-2 py-2 text-sm outline-none transition-colors hover:bg-accent hover:text-accent-foreground',
                      isSelected && 'bg-accent/50'
                    )}
                    onClick={(e) => {
                      e.stopPropagation();
                      toggleOption(option.id);
                    }}
                  >
                    <div
                      className={cn(
                        'mr-2 flex h-4 w-4 items-center justify-center rounded-sm border border-primary',
                        isSelected
                          ? 'bg-primary text-primary-foreground'
                          : 'opacity-50'
                      )}
                    >
                      {isSelected && <Check className="h-3 w-3" />}
                    </div>
                    <div className="flex-1">
                      <div className="font-medium">{option.name}</div>
                      {option.description && (
                        <div className="text-xs text-muted-foreground">
                          {option.description}
                        </div>
                      )}
                    </div>
                  </div>
                );
              })
            ) : (
              <div className="py-6 text-center text-sm text-muted-foreground">
                未找到匹配项
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
