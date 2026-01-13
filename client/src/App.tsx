import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import NotFound from "@/pages/NotFound";
import { Route, Switch, useLocation } from "wouter";
import ErrorBoundary from "./components/ErrorBoundary";
import { ThemeProvider } from "./contexts/ThemeContext";
import { AuthProvider, useAuth } from "./contexts/AuthContext";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";
import Users from "./pages/Users";
import Permissions from "./pages/Permissions";
import Layout from "./components/Layout";

// 受保护的路由组件
function ProtectedRoute({ path, component: Component }: { path: string; component: React.ComponentType }) {
  const { isAuthenticated, isLoading } = useAuth();
  const [location, setLocation] = useLocation();

  if (isLoading) {
    return <div className="flex items-center justify-center h-screen">加载中...</div>;
  }

  if (!isAuthenticated) {
    setLocation('/login');
    return null;
  }

  return (
    <Route path={path}>
      <Layout>
        <Component />
      </Layout>
    </Route>
  );
}
function Router() {
  const { isAuthenticated, isLoading } = useAuth();
  const [location] = useLocation();

  if (isLoading) {
    return <div className="flex items-center justify-center h-screen">加载中...</div>;
  }

  return (
    <Switch>
      <Route path="/login" component={Login} />
      {isAuthenticated && (
        <>
          <Route path="/" component={() => <Layout><Dashboard /></Layout>} />
          <Route path="/users" component={() => <Layout><Users /></Layout>} />
          <Route path="/permissions" component={() => <Layout><Permissions /></Layout>} />
        </>
      )}
      {!isAuthenticated && location !== '/login' && <Route component={() => <Login />} />}
      <Route path="/404" component={NotFound} />
      <Route component={NotFound} />
    </Switch>
  );
}

function App() {
  return (
    <ErrorBoundary>
      <ThemeProvider defaultTheme="light">
        <TooltipProvider>
          <AuthProvider>
            <Toaster />
            <Router />
          </AuthProvider>
        </TooltipProvider>
      </ThemeProvider>
    </ErrorBoundary>
  );
}

export default App;
