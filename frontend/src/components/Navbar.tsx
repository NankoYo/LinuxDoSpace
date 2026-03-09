import { useEffect, useRef, useState } from 'react';
import { AnimatePresence, motion } from 'motion/react';
import {
  Cloud,
  Github,
  Home,
  Key,
  LogIn,
  Mail,
  Menu,
  Moon,
  Settings,
  ShieldAlert,
  SlidersHorizontal,
  Sun,
  X,
} from 'lucide-react';

// NavbarTab constrains the navigation targets supported by the public shell.
type NavbarTab = 'home' | 'domains' | 'emails' | 'settings' | 'permissions' | 'supervision' | 'login';

interface NavbarProps {
  activeTab: string;
  setActiveTab: (tab: NavbarTab) => void;
  isDark: boolean;
  toggleTheme: () => void;
  animeBackgroundEnabled: boolean;
  onAnimeBackgroundEnabledChange: (enabled: boolean) => void;
  authenticated: boolean;
  displayName?: string;
  onAuthAction: () => void;
}

// navItems centralizes the main page links rendered in both desktop and mobile navigation.
const navItems = [
  { id: 'home' as const, label: '首页', icon: Home },
  { id: 'domains' as const, label: '域名分发', icon: Cloud },
  { id: 'emails' as const, label: '邮箱分发', icon: Mail },
  { id: 'settings' as const, label: '配置中心', icon: Settings },
  { id: 'permissions' as const, label: '权限申请', icon: Key },
  { id: 'supervision' as const, label: '共同监督', icon: ShieldAlert },
];

// Navbar renders the shared public-site navigation, top-right action buttons,
// and the mobile drawer while keeping the rest of the page shell unchanged.
export function Navbar({
  activeTab,
  setActiveTab,
  isDark,
  toggleTheme,
  animeBackgroundEnabled,
  onAnimeBackgroundEnabledChange,
  authenticated,
  displayName,
  onAuthAction,
}: NavbarProps) {
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const [isAppearancePanelOpen, setIsAppearancePanelOpen] = useState(false);
  const appearancePanelRef = useRef<HTMLDivElement | null>(null);
  const appearancePanelId = 'site-appearance-panel';

  useEffect(() => {
    setIsMobileMenuOpen(false);
    setIsAppearancePanelOpen(false);
  }, [activeTab]);

  useEffect(() => {
    if (!isAppearancePanelOpen) {
      return;
    }

    const handlePointerDown = (event: MouseEvent) => {
      if (appearancePanelRef.current?.contains(event.target as Node)) {
        return;
      }

      setIsAppearancePanelOpen(false);
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsAppearancePanelOpen(false);
      }
    };

    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [isAppearancePanelOpen]);

  function handleBrandClick(): void {
    setActiveTab('home');
  }

  function handleAuthClick(): void {
    setIsMobileMenuOpen(false);
    setIsAppearancePanelOpen(false);
    onAuthAction();
  }

  function toggleAppearancePanel(): void {
    setIsMobileMenuOpen(false);
    setIsAppearancePanelOpen((currentValue) => !currentValue);
  }

  return (
    <>
      <motion.nav
        initial={{ y: -50, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        className="fixed left-0 right-0 top-0 z-40 px-4 py-4 sm:px-6"
      >
        <div className="relative mx-auto flex max-w-[85rem] items-center justify-between gap-3 rounded-full border border-white/20 bg-white/30 px-4 py-3 shadow-lg backdrop-blur-xl dark:border-white/10 dark:bg-black/30 sm:px-6">
          <div className="flex min-w-0 cursor-pointer items-center gap-3" onClick={handleBrandClick}>
            <div className="h-11 w-11 shrink-0 overflow-hidden rounded-2xl bg-white/85 shadow-lg ring-1 ring-white/40 dark:bg-white/10 dark:ring-white/10">
              <img src="/logo.png" alt="LinuxDoSpace Logo" className="h-full w-full bg-white object-contain" />
            </div>
            <span className="truncate bg-gradient-to-r from-teal-500 to-emerald-600 bg-clip-text text-xl font-bold text-transparent dark:from-teal-400 dark:to-emerald-400">
              LinuxDoSpace
            </span>
          </div>

          <div className="hidden items-center gap-2 xl:flex">
            {navItems.map((item) => {
              const Icon = item.icon;
              const isActive = activeTab === item.id;
              return (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => setActiveTab(item.id)}
                  className={`flex items-center gap-2 whitespace-nowrap rounded-full px-4 py-2 transition-all duration-300 ${
                    isActive
                      ? 'bg-white/50 text-teal-600 shadow-sm dark:bg-white/10 dark:text-teal-300'
                      : 'text-gray-700 hover:bg-white/30 dark:text-gray-300 dark:hover:bg-white/5'
                  }`}
                >
                  <Icon size={18} className="shrink-0" />
                  <span className="font-medium">{item.label}</span>
                </button>
              );
            })}
          </div>

          <div className="hidden shrink-0 items-center gap-3 xl:flex">
            <button
              type="button"
              onClick={toggleTheme}
              className="rounded-full bg-white/30 p-2 text-gray-700 transition-all hover:bg-white/50 dark:bg-white/10 dark:text-gray-300 dark:hover:bg-white/20"
            >
              {isDark ? <Sun size={20} /> : <Moon size={20} />}
            </button>
            <a
              href="https://github.com/MoYeRanqianzhi/LinuxDoSpace"
              target="_blank"
              rel="noreferrer"
              title="查看 GitHub 仓库"
              aria-label="查看 GitHub 仓库"
              className="rounded-full bg-white/30 p-2 text-gray-700 transition-all hover:bg-white/50 dark:bg-white/10 dark:text-gray-300 dark:hover:bg-white/20"
            >
              <Github size={18} />
            </a>
            <button
              type="button"
              onClick={toggleAppearancePanel}
              aria-expanded={isAppearancePanelOpen}
              aria-controls={appearancePanelId}
              title="页面偏好设置"
              aria-label="页面偏好设置"
              className={`rounded-full p-2 transition-all ${
                isAppearancePanelOpen
                  ? 'bg-white/60 text-teal-700 dark:bg-white/20 dark:text-teal-200'
                  : 'bg-white/30 text-gray-700 hover:bg-white/50 dark:bg-white/10 dark:text-gray-300 dark:hover:bg-white/20'
              }`}
            >
              <SlidersHorizontal size={18} />
            </button>
            <button
              type="button"
              onClick={handleAuthClick}
              className="flex max-w-[11rem] items-center gap-2 whitespace-nowrap rounded-full bg-gradient-to-r from-teal-500 to-emerald-500 px-5 py-2 font-medium text-white shadow-lg transition-all hover:-translate-y-0.5 hover:from-teal-600 hover:to-emerald-600 hover:shadow-xl"
            >
              {authenticated ? <Settings size={18} className="shrink-0" /> : <LogIn size={18} className="shrink-0" />}
              <span className="truncate">{authenticated ? displayName ?? '控制台' : '登录'}</span>
            </button>
          </div>

          <div className="flex shrink-0 items-center gap-2 xl:hidden">
            <button
              type="button"
              onClick={toggleTheme}
              className="rounded-full bg-white/30 p-2 text-gray-700 transition-all hover:bg-white/50 dark:bg-white/10 dark:text-gray-300 dark:hover:bg-white/20"
            >
              {isDark ? <Sun size={20} /> : <Moon size={20} />}
            </button>
            <a
              href="https://github.com/MoYeRanqianzhi/LinuxDoSpace"
              target="_blank"
              rel="noreferrer"
              title="查看 GitHub 仓库"
              aria-label="查看 GitHub 仓库"
              className="rounded-full bg-white/30 p-2 text-gray-700 transition-all hover:bg-white/50 dark:bg-white/10 dark:text-gray-300 dark:hover:bg-white/20"
            >
              <Github size={18} />
            </a>
            <button
              type="button"
              onClick={toggleAppearancePanel}
              aria-expanded={isAppearancePanelOpen}
              aria-controls={appearancePanelId}
              title="页面偏好设置"
              aria-label="页面偏好设置"
              className={`rounded-full p-2 transition-all ${
                isAppearancePanelOpen
                  ? 'bg-white/60 text-teal-700 dark:bg-white/20 dark:text-teal-200'
                  : 'bg-white/30 text-gray-700 hover:bg-white/50 dark:bg-white/10 dark:text-gray-300 dark:hover:bg-white/20'
              }`}
            >
              <SlidersHorizontal size={18} />
            </button>
            <button
              type="button"
              onClick={() => {
                setIsAppearancePanelOpen(false);
                setIsMobileMenuOpen(true);
              }}
              className="rounded-full bg-white/30 p-2 text-gray-700 transition-all hover:bg-white/50 dark:bg-white/10 dark:text-gray-300 dark:hover:bg-white/20"
            >
              <Menu size={20} />
            </button>
          </div>

          <AnimatePresence>
            {isAppearancePanelOpen && (
              <motion.div
                id={appearancePanelId}
                ref={appearancePanelRef}
                initial={{ opacity: 0, y: -10, scale: 0.98 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: -10, scale: 0.98 }}
                transition={{ duration: 0.18, ease: 'easeOut' }}
                className="absolute right-3 top-full z-50 mt-3 w-[22rem] max-w-[calc(100vw-2rem)] rounded-3xl border border-white/25 bg-white/80 p-5 shadow-2xl backdrop-blur-2xl dark:border-white/10 dark:bg-gray-950/80 sm:right-6"
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-sm font-semibold text-gray-900 dark:text-white">页面偏好</div>
                    <p className="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-300">
                      默认开启二次元随机背景，关闭后会回到当前纯本地背景。
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => setIsAppearancePanelOpen(false)}
                    className="rounded-full p-2 text-gray-500 transition-all hover:bg-white/60 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-white/10 dark:hover:text-white"
                  >
                    <X size={16} />
                  </button>
                </div>

                <label className="mt-4 flex cursor-pointer items-center gap-4 rounded-2xl border border-white/25 bg-white/45 px-4 py-4 dark:border-white/10 dark:bg-white/5">
                  <div className="min-w-0 flex-1">
                    <div className="text-sm font-medium text-gray-900 dark:text-white">二次元随机背景</div>
                    <p className="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-300">
                      开启时浏览器会请求第三方动漫图；关闭后仅保留本站本地背景层。
                    </p>
                  </div>
                  <span
                    className={`relative inline-flex h-7 w-12 shrink-0 items-center rounded-full transition-colors ${
                      animeBackgroundEnabled ? 'bg-teal-500' : 'bg-gray-300 dark:bg-gray-700'
                    }`}
                  >
                    <span
                      className={`absolute left-1 h-5 w-5 rounded-full bg-white shadow-sm transition-transform ${
                        animeBackgroundEnabled ? 'translate-x-5' : 'translate-x-0'
                      }`}
                    />
                  </span>
                  <input
                    type="checkbox"
                    className="sr-only"
                    checked={animeBackgroundEnabled}
                    onChange={(event) => onAnimeBackgroundEnabledChange(event.target.checked)}
                  />
                </label>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </motion.nav>

      <AnimatePresence>
        {isMobileMenuOpen && (
          <>
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm xl:hidden"
              onClick={() => setIsMobileMenuOpen(false)}
            />

            <motion.aside
              initial={{ x: '100%' }}
              animate={{ x: 0 }}
              exit={{ x: '100%' }}
              transition={{ type: 'spring', stiffness: 300, damping: 30 }}
              className="fixed bottom-0 right-0 top-0 z-50 flex w-[20rem] max-w-[86vw] flex-col border-l border-white/20 bg-white/75 p-5 shadow-2xl backdrop-blur-2xl dark:border-white/10 dark:bg-gray-950/75 xl:hidden"
            >
              <div className="mb-6 flex items-center justify-between">
                <div>
                  <div className="text-lg font-bold text-gray-900 dark:text-white">页面导航</div>
                  <div className="text-sm text-gray-500 dark:text-gray-400">新 UI 结构与旧功能并行适配中</div>
                </div>
                <button
                  type="button"
                  onClick={() => setIsMobileMenuOpen(false)}
                  className="rounded-full bg-white/30 p-2 text-gray-700 transition-all hover:bg-white/50 dark:bg-white/10 dark:text-gray-300 dark:hover:bg-white/20"
                >
                  <X size={18} />
                </button>
              </div>

              <div className="space-y-2">
                {navItems.map((item) => {
                  const Icon = item.icon;
                  const isActive = activeTab === item.id;
                  return (
                    <button
                      key={item.id}
                      type="button"
                      onClick={() => setActiveTab(item.id)}
                      className={`flex w-full items-center gap-3 rounded-2xl px-4 py-3 text-left transition-all ${
                        isActive
                          ? 'bg-teal-500 text-white shadow-lg'
                          : 'bg-white/35 text-gray-700 hover:bg-white/50 dark:bg-white/5 dark:text-gray-300 dark:hover:bg-white/10'
                      }`}
                    >
                      <Icon size={18} className="shrink-0" />
                      <span className="font-medium">{item.label}</span>
                    </button>
                  );
                })}
              </div>

              <div className="mt-6 rounded-2xl border border-white/20 bg-white/30 p-4 text-sm text-gray-600 dark:border-white/10 dark:bg-white/5 dark:text-gray-300">
                新 UI 已接入更多页面。当前仍以后端已上线功能为准，未接后端的页面先展示前端结构与交互预览。
              </div>

              <div className="mt-auto pt-6">
                <button
                  type="button"
                  onClick={handleAuthClick}
                  className="flex w-full items-center justify-center gap-2 rounded-2xl bg-gradient-to-r from-teal-500 to-emerald-500 px-5 py-3 font-medium text-white shadow-lg transition-all hover:from-teal-600 hover:to-emerald-600"
                >
                  {authenticated ? <Settings size={18} /> : <LogIn size={18} />}
                  <span>{authenticated ? displayName ?? '进入控制台' : '登录 Linux Do'}</span>
                </button>
              </div>
            </motion.aside>
          </>
        )}
      </AnimatePresence>
    </>
  );
}
