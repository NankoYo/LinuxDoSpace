import { motion } from 'motion/react';
import { GlassCard } from '../components/GlassCard';
import { ShieldCheck, ArrowRight, LogOut } from 'lucide-react';
import { LinuxDoIcon } from '../components/LinuxDoIcon';
import type { User } from '../types/api';

// LoginProps 描述登录页和外层应用之间的交互契约。
interface LoginProps {
  authenticated: boolean;
  oauthConfigured: boolean;
  user?: User;
  onLogin: () => void;
  onOpenSettings: () => void;
  onLogout: () => Promise<void>;
}

// Login 负责展示 Linux Do 登录入口，并在用户已登录时展示当前账号状态。
export function Login({
  authenticated,
  oauthConfigured,
  user,
  onLogin,
  onOpenSettings,
  onLogout,
}: LoginProps) {
  return (
    <div className="min-h-screen flex items-center justify-center px-4 pt-20">
      <GlassCard className="w-full max-w-md text-center relative overflow-hidden">
        {/* 装饰性背景光斑保留原有风格，只补真实业务逻辑。 */}
        <div className="absolute -top-20 -right-20 w-40 h-40 bg-teal-400 rounded-full mix-blend-multiply filter blur-2xl opacity-50 animate-blob" />
        <div className="absolute -bottom-20 -left-20 w-40 h-40 bg-emerald-400 rounded-full mix-blend-multiply filter blur-2xl opacity-50 animate-blob animation-delay-2000" />

        <motion.div
          initial={{ scale: 0.9, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ duration: 0.5 }}
          className="relative z-10"
        >
          <div className="w-20 h-20 mx-auto mb-6 rounded-full bg-gradient-to-tr from-teal-500 to-emerald-600 flex items-center justify-center text-white font-bold text-4xl shadow-2xl shadow-emerald-500/30 overflow-hidden">
            {authenticated && user?.avatar_url ? (
              <img src={user.avatar_url} alt={user.display_name} className="w-full h-full object-cover" />
            ) : (
              'L'
            )}
          </div>

          <h2 className="text-3xl font-extrabold mb-2 text-gray-900 dark:text-white">
            {authenticated ? '欢迎回来，佬友' : '欢迎来到佬友空间'}
          </h2>
          <p className="text-gray-600 dark:text-gray-300 mb-8">
            {authenticated
              ? `${user?.display_name || user?.username} 已通过 Linux Do 授权登录`
              : '请使用 Linux Do 账号授权登录'}
          </p>

          {!authenticated ? (
            <button
              onClick={onLogin}
              disabled={!oauthConfigured}
              className="w-full flex items-center justify-center gap-3 px-6 py-4 rounded-2xl bg-[#1a1a1a] dark:bg-white hover:bg-black dark:hover:bg-gray-100 disabled:opacity-60 disabled:cursor-not-allowed text-white dark:text-black font-bold text-lg shadow-xl transition-all transform hover:scale-[1.02] active:scale-95"
            >
              <LinuxDoIcon className="w-6 h-6 shrink-0" />
              <span>{oauthConfigured ? '使用 Linux Do 继续' : 'OAuth 暂未配置'}</span>
            </button>
          ) : (
            <div className="space-y-3">
              <button
                onClick={onOpenSettings}
                className="w-full flex items-center justify-center gap-3 px-6 py-4 rounded-2xl bg-[#1a1a1a] dark:bg-white hover:bg-black dark:hover:bg-gray-100 text-white dark:text-black font-bold text-lg shadow-xl transition-all transform hover:scale-[1.02] active:scale-95"
              >
                <ArrowRight size={24} />
                <span>进入配置中心</span>
              </button>
              <button
                onClick={() => void onLogout()}
                className="w-full flex items-center justify-center gap-3 px-6 py-4 rounded-2xl bg-white/45 dark:bg-black/35 hover:bg-white/60 dark:hover:bg-black/50 text-gray-900 dark:text-white font-bold text-lg shadow-xl transition-all transform hover:scale-[1.02] active:scale-95"
              >
                <LogOut size={22} />
                <span>退出当前登录</span>
              </button>
            </div>
          )}

          <div className="mt-8 text-sm text-gray-500 dark:text-gray-400 flex items-center justify-center gap-2">
            <ShieldCheck size={16} />
            <span>
              {authenticated
                ? '当前会话由后端 Session 与 CSRF 双重保护'
                : '登录即代表同意服务条款与隐私政策'}
            </span>
          </div>
        </motion.div>
      </GlassCard>
    </div>
  );
}
