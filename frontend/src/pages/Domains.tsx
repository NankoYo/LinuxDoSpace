import { useState } from 'react';
import { motion } from 'motion/react';
import { GlassCard } from '../components/GlassCard';
import { Search, CheckCircle, XCircle } from 'lucide-react';

export function Domains() {
  const [domain, setDomain] = useState('');
  const [status, setStatus] = useState<'idle' | 'available' | 'taken'>('idle');

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (!domain) return;
    // Mock check
    setStatus(Math.random() > 0.5 ? 'available' : 'taken');
  };

  return (
    <div className="max-w-4xl mx-auto pt-32 pb-24 px-6">
      <motion.div
        initial={{ y: 20, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        className="text-center mb-12"
      >
        <h1 className="text-4xl md:text-5xl font-extrabold mb-4 text-gray-900 dark:text-white">
          寻找你的专属域名
        </h1>
        <p className="text-lg text-gray-700 dark:text-gray-300">
          目前支持 <span className="font-bold text-teal-500">linuxdo.space</span> 后缀
        </p>
      </motion.div>

      <GlassCard className="mb-8">
        <form onSubmit={handleSearch} className="flex flex-col sm:flex-row gap-4">
          <div className="relative flex-1">
            <input
              type="text"
              value={domain}
              onChange={(e) => {
                setDomain(e.target.value);
                setStatus('idle');
              }}
              placeholder="输入你想要的域名前缀"
              className="w-full pl-4 pr-32 py-4 rounded-2xl bg-white/50 dark:bg-black/50 border border-white/40 dark:border-white/20 focus:outline-none focus:ring-2 focus:ring-teal-500 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 transition-all"
            />
            <div className="absolute right-4 top-1/2 -translate-y-1/2 text-gray-500 font-medium">
              .linuxdo.space
            </div>
          </div>
          <button
            type="submit"
            className="flex items-center justify-center gap-2 px-8 py-4 rounded-2xl bg-gradient-to-r from-teal-500 to-emerald-500 hover:from-teal-600 hover:to-emerald-600 text-white font-bold shadow-lg transition-all transform hover:scale-105"
          >
            <Search size={20} />
            查询
          </button>
        </form>

        {status !== 'idle' && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            className="mt-6 p-4 rounded-2xl bg-white/30 dark:bg-black/30 border border-white/20 flex items-center justify-between"
          >
            <div className="flex items-center gap-3">
              {status === 'available' ? (
                <CheckCircle className="text-green-500 w-6 h-6" />
              ) : (
                <XCircle className="text-red-500 w-6 h-6" />
              )}
              <span className="text-lg font-medium text-gray-900 dark:text-white">
                {domain}.linuxdo.space
              </span>
            </div>
            {status === 'available' ? (
              <button className="px-6 py-2 rounded-xl bg-green-500 hover:bg-green-600 text-white font-medium transition-colors">
                立即注册
              </button>
            ) : (
              <span className="text-red-500 font-medium">已被注册</span>
            )}
          </motion.div>
        )}
      </GlassCard>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {[1, 2, 3].map((i) => (
          <GlassCard key={i} delay={0.2 + i * 0.1} className="text-center">
            <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-gradient-to-br from-teal-400 to-emerald-400 flex items-center justify-center text-white font-bold text-xl shadow-lg">
              {i}
            </div>
            <h3 className="text-lg font-bold mb-2 text-gray-900 dark:text-white">
              {i === 1 ? '选择前缀' : i === 2 ? '验证身份' : '配置解析'}
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              {i === 1
                ? '输入心仪的域名前缀进行查询'
                : i === 2
                ? '通过 Linux Do 账号授权登录'
                : '一键添加 A/CNAME 记录生效'}
            </p>
          </GlassCard>
        ))}
      </div>
    </div>
  );
}
