import React, { useState } from 'react';
import { useLocation } from 'wouter';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useAuth } from '@/contexts/AuthContext';
import { toast } from 'sonner';

export default function Login() {
  const [, setLocation] = useLocation();
  const { login } = useAuth();
  const [isLoading, setIsLoading] = useState(false);
  const [isRegister, setIsRegister] = useState(false);
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    try {
      if (isRegister) {
        if (formData.password !== formData.confirmPassword) {
          toast.error('密码不一致');
          return;
        }
        await login(formData.username, formData.password);
        toast.success('注册成功！');
      } else {
        await login(formData.username, formData.password);
        toast.success('登录成功！');
      }
      setLocation('/');
    } catch (error: any) {
      toast.error(error?.message || '操作失败');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100">
      <Card className="w-full max-w-md shadow-lg">
        <CardHeader className="space-y-2 text-center">
          <CardTitle className="text-2xl font-bold">CMDB管理系统</CardTitle>
          <CardDescription>
            {isRegister ? '创建新账号' : '登录您的账号'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">用户名</label>
              <Input
                name="username"
                placeholder="输入用户名"
                value={formData.username}
                onChange={handleChange}
                required
              />
            </div>

            {isRegister && (
              <div className="space-y-2">
                <label className="text-sm font-medium">邮箱</label>
                <Input
                  name="email"
                  type="email"
                  placeholder="输入邮箱"
                  value={formData.email}
                  onChange={handleChange}
                  required
                />
              </div>
            )}

            <div className="space-y-2">
              <label className="text-sm font-medium">密码</label>
              <Input
                name="password"
                type="password"
                placeholder="输入密码"
                value={formData.password}
                onChange={handleChange}
                required
              />
            </div>

            {isRegister && (
              <div className="space-y-2">
                <label className="text-sm font-medium">确认密码</label>
                <Input
                  name="confirmPassword"
                  type="password"
                  placeholder="确认密码"
                  value={formData.confirmPassword}
                  onChange={handleChange}
                  required
                />
              </div>
            )}

            <Button
              type="submit"
              className="w-full"
              disabled={isLoading}
            >
              {isLoading ? '处理中...' : isRegister ? '注册' : '登录'}
            </Button>
          </form>

          <div className="mt-4 text-center text-sm">
            {isRegister ? (
              <>
                已有账号？{' '}
                <button
                  onClick={() => setIsRegister(false)}
                  className="text-blue-600 hover:underline font-medium"
                >
                  返回登录
                </button>
              </>
            ) : (
              <>
                没有账号？{' '}
                <button
                  onClick={() => setIsRegister(true)}
                  className="text-blue-600 hover:underline font-medium"
                >
                  创建账号
                </button>
              </>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
