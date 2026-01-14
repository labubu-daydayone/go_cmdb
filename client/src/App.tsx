import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import NotFound from "@/pages/NotFound";
import { Route, Switch, useLocation } from "wouter";
import ErrorBoundary from "./components/ErrorBoundary";
import { ThemeProvider } from "./contexts/ThemeContext";
import { AuthProvider, useAuth } from "./contexts/AuthContext";
import { TabProvider } from "./components/TabManager";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";
import Users from "./pages/Users";
import Permissions from "./pages/Permissions";
import Layout from "./components/Layout";

// 域名管理模块
import DomainList from "./pages/domain/DomainList";
import ParseTemplate from "./pages/domain/ParseTemplate";
import AccountGroup from "./pages/domain/AccountGroup";
import CertManagement from "./pages/domain/CertManagement";

// 网站管理模块
import WebsiteList from "./pages/website/WebsiteList";
import DnsConfig from "./pages/website/DnsConfig";
import ParseLine from "./pages/website/ParseLine";
import NodeList from "./pages/website/NodeList";
import NodeGroup from "./pages/website/NodeGroup";
import HuaweiCdn from "./pages/website/HuaweiCdn";
import OriginList from "./pages/website/OriginList";

// 系统设置模块
import KeyConfig from "./pages/system/KeyConfig";

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
          
          {/* 域名管理 */}
          <Route path="/domain/list" component={() => <Layout><DomainList /></Layout>} />
          <Route path="/domain/template" component={() => <Layout><ParseTemplate /></Layout>} />
          <Route path="/domain/group" component={() => <Layout><AccountGroup /></Layout>} />
          <Route path="/domain/cert" component={() => <Layout><CertManagement /></Layout>} />
          
          {/* 网站管理 */}
          <Route path="/website/list" component={() => <Layout><WebsiteList /></Layout>} />
          <Route path="/website/dns" component={() => <Layout><DnsConfig /></Layout>} />
          <Route path="/website/line" component={() => <Layout><ParseLine /></Layout>} />
          <Route path="/website/nodes" component={() => <Layout><NodeList /></Layout>} />
          <Route path="/website/node-group" component={() => <Layout><NodeGroup /></Layout>} />
          <Route path="/website/huawei-cdn" component={() => <Layout><HuaweiCdn /></Layout>} />
          <Route path="/website/origin" component={() => <Layout><OriginList /></Layout>} />
          
          {/* 系统设置 */}
          <Route path="/users" component={() => <Layout><Users /></Layout>} />
          <Route path="/permissions" component={() => <Layout><Permissions /></Layout>} />
          <Route path="/system/keys" component={() => <Layout><KeyConfig /></Layout>} />
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
            <TabProvider>
              <Toaster />
              <Router />
            </TabProvider>
          </AuthProvider>
        </TooltipProvider>
      </ThemeProvider>
    </ErrorBoundary>
  );
}

export default App;
