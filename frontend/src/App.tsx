import { startTransition, useEffect, useState } from 'react';
import { Footer } from './components/Footer';
import { Navbar } from './components/Navbar';
import {
  APIError,
  getAuthLoginURL,
  getCurrentSession,
  listPublicDomains,
  logout,
} from './lib/api';
import { Domains } from './pages/Domains';
import { Emails } from './pages/Emails';
import { Home } from './pages/Home';
import { Login } from './pages/Login';
import { Permissions } from './pages/Permissions';
import { Settings } from './pages/Settings';
import { Supervision } from './pages/Supervision';
import type { Allocation, ManagedDomain, MeResponse, User } from './types/api';

// TabKey enumerates every client-side page supported by the public frontend.
type TabKey = 'home' | 'domains' | 'emails' | 'settings' | 'permissions' | 'supervision' | 'login';

// SessionState mirrors the normalized `/v1/me` payload consumed across pages.
interface SessionState {
  authenticated: boolean;
  oauthConfigured: boolean;
  user?: User;
  csrfToken?: string;
  sessionExpiresAt?: string;
  allocations: Allocation[];
}

// tabPathMap keeps URL updates centralized without bringing in a full router.
const tabPathMap: Record<TabKey, string> = {
  home: '/',
  domains: '/domains',
  emails: '/emails',
  settings: '/settings',
  permissions: '/permissions',
  supervision: '/supervision',
  login: '/login',
};

// pathToTab converts the current pathname into the matching view key.
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

// normalizeSessionResponse converts the backend response into the smaller shape
// shared by the public-site pages.
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

// SiteBackground renders a fully local background so the public site no longer
// leaks visitor metadata to an external image host.
function SiteBackground() {
  return (
    <>
      <div className="fixed inset-0 z-[-3] bg-[radial-gradient(circle_at_top_left,_rgba(250,204,21,0.26),_transparent_38%),radial-gradient(circle_at_top_right,_rgba(56,189,248,0.18),_transparent_34%),linear-gradient(160deg,_#f8fafc_0%,_#e2e8f0_46%,_#cbd5e1_100%)] transition-colors duration-500 dark:bg-[radial-gradient(circle_at_top_left,_rgba(234,179,8,0.16),_transparent_38%),radial-gradient(circle_at_top_right,_rgba(34,211,238,0.12),_transparent_34%),linear-gradient(160deg,_#020617_0%,_#0f172a_54%,_#111827_100%)]" />
      <div className="fixed inset-0 z-[-2] overflow-hidden">
        <div className="absolute -left-20 top-12 h-64 w-64 rounded-full bg-amber-300/35 blur-3xl dark:bg-amber-500/18" />
        <div className="absolute right-[-5rem] top-1/4 h-72 w-72 rounded-full bg-sky-300/30 blur-3xl dark:bg-cyan-400/16" />
        <div className="absolute bottom-[-6rem] left-1/3 h-80 w-80 rounded-full bg-emerald-300/18 blur-3xl dark:bg-emerald-400/12" />
      </div>
      <div className="fixed inset-0 z-[-1] bg-white/40 backdrop-blur-[2px] transition-colors duration-500 dark:bg-black/40" />
    </>
  );
}

// readableErrorMessage turns browser-side failures into one stable user-facing message.
function readableErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof APIError) {
    return error.message;
  }
  if (error instanceof Error && error.message.trim() !== '') {
    return error.message;
  }
  return fallback;
}

// App coordinates page switching, session refresh, and the shared public layout.
export default function App() {
  const [activeTab, setActiveTab] = useState<TabKey>(() => pathToTab(window.location.pathname));
  const [isDark, setIsDark] = useState(false);
  const [session, setSession] = useState<SessionState>({
    authenticated: false,
    oauthConfigured: false,
    allocations: [],
  });
  const [sessionLoading, setSessionLoading] = useState(true);
  const [sessionError, setSessionError] = useState('');
  const [publicDomains, setPublicDomains] = useState<ManagedDomain[]>([]);
  const [domainsLoading, setDomainsLoading] = useState(true);
  const [domainsError, setDomainsError] = useState('');

  useEffect(() => {
    if (isDark) {
      document.documentElement.classList.add('dark');
      return;
    }
    document.documentElement.classList.remove('dark');
  }, [isDark]);

  useEffect(() => {
    const handlePopState = () => {
      startTransition(() => {
        setActiveTab(pathToTab(window.location.pathname));
      });
    };

    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  useEffect(() => {
    const expectedPath = tabPathMap[activeTab];
    if (window.location.pathname !== expectedPath) {
      window.history.pushState({}, '', expectedPath);
    }
  }, [activeTab]);

  useEffect(() => {
    void refreshSession();
    void refreshPublicDomains();
  }, []);

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

  function navigateToTab(tab: TabKey): void {
    startTransition(() => {
      setActiveTab(tab);
    });
  }

  function beginLogin(nextTab: TabKey): void {
    window.location.assign(getAuthLoginURL(tabPathMap[nextTab]));
  }

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

  async function handleAllocationCreated(): Promise<void> {
    await refreshSession({ silent: true });
    navigateToTab('settings');
  }

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

  const bannerMessage = sessionError || domainsError;

  return (
    <div className="relative min-h-screen overflow-x-hidden font-sans text-gray-900 transition-colors duration-500 dark:text-white">
      <SiteBackground />

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

      {bannerMessage && (
        <div className="relative z-20 px-6 pt-24">
          <div className="mx-auto max-w-5xl rounded-2xl border border-amber-300/40 bg-amber-100/70 px-4 py-3 text-sm text-amber-900 shadow-lg backdrop-blur-md dark:border-amber-700/40 dark:bg-amber-950/40 dark:text-amber-200">
            {bannerMessage}
          </div>
        </div>
      )}

      <main className="relative z-10 min-h-screen">{renderContent()}</main>

      <Footer />
    </div>
  );
}
