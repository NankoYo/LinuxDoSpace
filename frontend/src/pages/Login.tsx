import { motion } from 'motion/react';
import { GlassCard } from '../components/GlassCard';
import { LogIn } from 'lucide-react';

export function Login() {
  return (
    <div className="min-h-screen flex items-center justify-center px-4 pt-20">
      <GlassCard className="w-full max-w-md text-center relative overflow-hidden">
        {/* Decorative background elements */}
        <div className="absolute -top-20 -right-20 w-40 h-40 bg-teal-400 rounded-full mix-blend-multiply filter blur-2xl opacity-50 animate-blob" />
        <div className="absolute -bottom-20 -left-20 w-40 h-40 bg-emerald-400 rounded-full mix-blend-multiply filter blur-2xl opacity-50 animate-blob animation-delay-2000" />
        
        <motion.div
          initial={{ scale: 0.9, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ duration: 0.5 }}
          className="relative z-10"
        >
          <div className="w-20 h-20 mx-auto mb-6 rounded-full bg-gradient-to-tr from-teal-500 to-emerald-600 flex items-center justify-center text-white font-bold text-4xl shadow-2xl shadow-emerald-500/30">
            L
          </div>
          
          <h2 className="text-3xl font-extrabold mb-2 text-gray-900 dark:text-white">
            欢迎来到佬友空间
          </h2>
          <p className="text-gray-600 dark:text-gray-300 mb-8">
            请使用 Linux Do 账号授权登录
          </p>

          <button className="w-full flex items-center justify-center gap-3 px-6 py-4 rounded-2xl bg-[#1a1a1a] dark:bg-white hover:bg-black dark:hover:bg-gray-100 text-white dark:text-black font-bold text-lg shadow-xl transition-all transform hover:scale-[1.02] active:scale-95">
            <LogIn size={24} />
            <span>Login with Linux Do</span>
          </button>
          
          <div className="mt-8 text-sm text-gray-500 dark:text-gray-400">
            登录即代表同意服务条款与隐私政策
          </div>
        </motion.div>
      </GlassCard>
    </div>
  );
}
