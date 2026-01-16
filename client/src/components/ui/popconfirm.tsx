import * as React from "react";
import { AlertCircle } from "lucide-react";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Button } from "@/components/ui/button";

export interface PopconfirmProps {
  title: string;
  description?: string;
  onConfirm?: () => void | Promise<void>;
  onCancel?: () => void;
  okText?: string;
  cancelText?: string;
  children: React.ReactElement;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}

export function Popconfirm({
  title,
  description,
  onConfirm,
  onCancel,
  okText = "OK",
  cancelText = "取消",
  children,
  open: controlledOpen,
  onOpenChange,
}: PopconfirmProps) {
  const [internalOpen, setInternalOpen] = React.useState(false);
  const [loading, setLoading] = React.useState(false);

  // 使用受控或非受控模式
  const open = controlledOpen !== undefined ? controlledOpen : internalOpen;
  const setOpen = onOpenChange || setInternalOpen;

  const handleConfirm = async () => {
    if (onConfirm) {
      setLoading(true);
      try {
        await onConfirm();
      } finally {
        setLoading(false);
      }
    }
    setOpen(false);
  };

  const handleCancel = () => {
    if (onCancel) {
      onCancel();
    }
    setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent
        className="w-auto p-0"
        side="top"
        align="center"
        sideOffset={8}
      >
        <div className="flex flex-col gap-3 p-3">
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0 mt-0.5">
              <div className="w-6 h-6 rounded-full bg-orange-100 flex items-center justify-center">
                <AlertCircle className="w-4 h-4 text-orange-500" />
              </div>
            </div>
            <div className="flex-1">
              <div className="font-medium text-sm">{title}</div>
              {description && (
                <div className="text-xs text-muted-foreground mt-1">
                  {description}
                </div>
              )}
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleCancel}
              disabled={loading}
            >
              {cancelText}
            </Button>
            <Button size="sm" onClick={handleConfirm} disabled={loading}>
              {loading ? "处理中..." : okText}
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
