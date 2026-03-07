import { startTransition, useEffect, useState } from 'react';
import { Navbar } from './components/Navbar';
import { Footer } from './components/Footer';
import { Home } from './pages/Home';
import { Domains } from './pages/Domains';
import { Emails } from './pages/Emails';
import { Settings } from './pages/Settings';
import { Permissions } from './pages/Permissions';
import { Login } from './pages/Login';
import { Supervision } from './pages/Supervision';
import {
  APIError,
  getAuthLoginURL,
  getCurrentSession,
  listPublicDomains,
  logout,
} from './lib/api';
import type { Allocation, ManagedDomain, MeResponse, User } from './types/api';

// TabKey 定义当前前端主应用支持的全部页面标签。
// 当前除了已经接好后端的页面外，还额外承接新 UI 中的“邮箱分发”和“权限申请”占位页。
type TabKey = 'home' | 'domains' | 'emails' | 'settings' | 'permissions' | 'supervision' | 'login';

// SessionState 表示浏览器端缓存的当前登录态。
// 该状态来自后端 `/v1/me`，并被多个真实业务页面共同消费。
interface SessionState {
  authenticated: boolean;
  oauthConfigured: boolean;
  user?: User;
  csrfToken?: string;
  sessionExpiresAt?: string;
  allocations: Allocation[];
}

// tabPathMap 把前端页面状态映射到 URL。
// 这样做可以在不接入 React Router 的前提下，仍然支持直接访问和前进后退。
const tabPathMap: Record<TabKey, string> = {
  home: '/',
  domains: '/domains',
  emails: '/emails',
  settings: '/settings',
  permissions: '/permissions',
  supervision: '/supervision',
  login: '/login',
};

// pathToTab 根据浏览器路径反推出应该展示的页面标签。
function pathToTab(pathname: string): TabKey {
  switch (pathname.toLowerCase()) {
    case '/domains':
      return 'domains';
    case '/emails':
      return 'emails';
    case '/settings':
      return 'settings';
    case '/permissions':
      return 'permissions';
    case '/supervision':
      return 'supervision';
    case '/login':
      return 'login';
    default:
      return 'home';
  }
}

// normalizeSessionResponse 把后端 `/v1/me` 响应收敛为前端统一结构。
function normalizeSessionResponse(response: MeResponse): SessionState {
  return {
    authenticated: response.authenticated,
    oauthConfigured: response.oauth_configured ?? response.authenticated,
    user: response.user,
    csrfToken: response.csrf_token,
    sessionExpiresAt: response.session_expires_at,
    allocations: response.allocations ?? [],
  };
}

// App 负责维护主应用的页面切换、登录态刷新和全局视觉外壳。
export default function App() {
  // activeTab 保存当前展示中的主页面。
  const [activeTab, setActiveTab] = useState<TabKey>(() => pathToTab(window.location.pathname));

  // isDark 控制当前主题是否启用深色模式。
  const [isDark, setIsDark] = useState(false);

  // session 保存当前浏览器对应的登录态与 allocation 列表。
  const [session, setSession] = useState<SessionState>({
    authenticated: false,
    oauthConfigured: false,
    allocations: [],
  });

  // sessionLoading 控制登录态读取过程中的加载提示。
  const [sessionLoading, setSessionLoading] = useState(true);

  // sessionError 保存登录态读取失败时的提示文本。
  const [sessionError, setSessionError] = useState('');

  // publicDomains 保存域名分发页可选的公开根域名列表。
  const [publicDomains, setPublicDomains] = useState<ManagedDomain[]>([]);

  // domainsLoading 控制公开根域名列表的加载态。
  const [domainsLoading, setDomainsLoading] = useState(true);

  // domainsError 保存公开根域名列表加载失败时的错误提示。
  const [domainsError, setDomainsError] = useState('');

  // 主题切换继续沿用 DOM 根元素上的 `.dark` 类，不改变现有样式体系。
  useEffect(() => {
    if (isDark) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, [isDark]);

  // 监听浏览器前进后退，把地址栏变化同步回前端状态。
  useEffect(() => {
    const handlePopState = () => {
      startTransition(() => {
        setActiveTab(pathToTab(window.location.pathname));
      });
    };

    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  // 当页面标签变化时，把对应路径写回地址栏。
  useEffect(() => {
    const expectedPath = tabPathMap[activeTab];
    if (window.location.pathname !== expectedPath) {
      window.history.pushState({}, '', expectedPath);
    }
  }, [activeTab]);

  // 首次进入页面时，读取登录态与公开根域名列表。
  useEffect(() => {
    void refreshSession();
    void refreshPublicDomains();
  }, []);

  // refreshSession 刷新当前浏览器登录态。
  // `silent=true` 时仅静默刷新，不改变页面主加载提示。
  async function refreshSession(options: { silent?: boolean } = {}): Promise<void> {
    const { silent = false } = options;
    if (!silent) {
      setSessionLoading(true);
    }

    try {
      const response = await getCurrentSession();
      setSession(normalizeSessionResponse(response));
      setSessionError('');
    } catch (error) {
      setSession({
        authenticated: false,
        oauthConfigured: false,
        allocations: [],
      });
      setSessionError(readableErrorMessage(error, '无法加载当前登录状态'));
    } finally {
      if (!silent) {
        setSessionLoading(false);
      }
    }
  }

  // refreshPublicDomains 刷新域名分发页用到的公开根域名数据。
  async function refreshPublicDomains(): Promise<void> {
    setDomainsLoading(true);

    try {
      const domains = await listPublicDomains();
      setPublicDomains(domains);
      setDomainsError('');
    } catch (error) {
      setPublicDomains([]);
      setDomainsError(readableErrorMessage(error, '无法加载可分发域名列表'));
    } finally {
      setDomainsLoading(false);
    }
  }

  // navigateToTab 统一封装主页面切换逻辑。
  function navigateToTab(tab: TabKey): void {
    startTransition(() => {
      setActiveTab(tab);
    });
  }

  // beginLogin 把用户带到后端 Linux Do OAuth 登录入口。
  function beginLogin(nextTab: TabKey): void {
    window.location.assign(getAuthLoginURL(tabPathMap[nextTab]));
  }

  // handleLogout 请求后端销毁当前会话，并把前端退回首页。
  async function handleLogout(): Promise<void> {
    if (!session.csrfToken) {
      return;
    }

    try {
      await logout(session.csrfToken);
      setSession({
        authenticated: false,
        oauthConfigured: session.oauthConfigured,
        allocations: [],
      });
      setSessionError('');
      navigateToTab('home');
    } catch (error) {
      setSessionError(readableErrorMessage(error, '退出登录失败'));
    }
  }

  // handleAllocationCreated 在 allocation 创建成功后刷新会话并跳到配置中心。
  async function handleAllocationCreated(): Promise<void> {
    await refreshSession({ silent: true });
    navigateToTab('settings');
  }

  // renderContent 根据当前标签选择页面内容。
  function renderContent() {
    switch (activeTab) {
      case 'home':
        return <Home />;
      case 'domains':
        return (
          <Domains
            publicDomains={publicDomains}
            domainsLoading={domainsLoading}
            domainsError={domainsError}
            authenticated={session.authenticated}
            user={session.user}
            allocations={session.allocations}
            csrfToken={session.csrfToken}
            onLogin={() => beginLogin('domains')}
            onAllocationCreated={handleAllocationCreated}
          />
        );
      case 'emails':
        return <Emails />;
      case 'settings':
        return (
          <Settings
            authenticated={session.authenticated}
            sessionLoading={sessionLoading}
            user={session.user}
            allocations={session.allocations}
            csrfToken={session.csrfToken}
            onLogin={() => beginLogin('settings')}
            onNavigateDomains={() => navigateToTab('domains')}
            onSessionRefresh={() => refreshSession({ silent: true })}
            onLogout={handleLogout}
          />
        );
      case 'permissions':
        return <Permissions />;
      case 'supervision':
        return <Supervision />;
      case 'login':
        return (
          <Login
            authenticated={session.authenticated}
            oauthConfigured={session.oauthConfigured}
            user={session.user}
            onLogin={() => beginLogin('settings')}
            onOpenSettings={() => navigateToTab('settings')}
            onLogout={handleLogout}
          />
        );
      default:
        return <Home />;
    }
  }

  // bannerMessage 统一承接最重要的全局错误提示。
  const bannerMessage = sessionError || domainsError;

  return (
    <div className="min-h-screen font-sans text-gray-900 dark:text-white transition-colors duration-500 overflow-x-hidden relative">
      {/* 背景图继续沿用现有视觉方向，不在这次 UI 迁移里改变内容资源。 */}
      <div
        className="fixed inset-0 z-[-2] bg-cover bg-center bg-no-repeat transition-all duration-1000 dark:brightness-[0.3]"
        style={{ backgroundImage: 'url(https://www.loliapi.com/acg/)' }}
      />

      {/* 轻量遮罩继续保留，确保复杂背景下的文本可读性。 */}
      <div className="fixed inset-0 z-[-1] bg-white/40 dark:bg-black/40 backdrop-blur-[2px] transition-colors duration-500" />

      <Navbar
        activeTab={activeTab}
        setActiveTab={navigateToTab}
        isDark={isDark}
        toggleTheme={() => setIsDark(!isDark)}
        authenticated={session.authenticated}
        displayName={session.user?.display_name || session.user?.username}
        onAuthAction={() => {
          if (session.authenticated) {
            navigateToTab('settings');
            return;
          }
          navigateToTab('login');
        }}
      />

      {/* 顶部横幅继续用于提示全局后端错误，避免被新 UI 隐藏。 */}
      {bannerMessage && (
        <div className="pt-24 px-6 relative z-20">
          <div className="max-w-5xl mx-auto rounded-2xl border border-amber-300/40 bg-amber-100/70 dark:bg-amber-950/40 dark:border-amber-700/40 backdrop-blur-md px-4 py-3 text-sm text-amber-900 dark:text-amber-200 shadow-lg">
            {bannerMessage}
          </div>
        </div>
      )}

      <main className="relative z-10 min-h-screen">{renderContent()}</main>

      <Footer />
    </div>
  );
}

// readableErrorMessage 把浏览器端异常统一转成用户可读的提示文本。
function readableErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof APIError) {
    return error.message;
  }
  if (error instanceof Error && error.message.trim() !== '') {
    return error.message;
  }
  return fallback;
}
