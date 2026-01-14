import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Construction } from "lucide-react";

interface PlaceholderPageProps {
  title: string;
  description?: string;
}

export default function PlaceholderPage({ title, description }: PlaceholderPageProps) {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">{title}</h1>
        {description && <p className="text-gray-600 mt-2">{description}</p>}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Construction className="w-5 h-5 text-orange-500" />
            功能开发中
          </CardTitle>
          <CardDescription>
            此功能正在开发中，敬请期待
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            该页面将包含 {title} 相关的所有功能和数据管理。
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
