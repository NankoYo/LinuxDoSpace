import { motion } from 'motion/react';
import { GlassCard } from '../components/GlassCard';
import { Rocket, Shield, Zap, Globe } from 'lucide-react';

export function Home() {
  const features = [
    {
      icon: <Globe className="w-8 h-8 text-teal-500" />,
      title: '专属二级域名',
      description: '提供 linuxdo.space 等优质后缀，彰显佬友尊贵身份，让你的项目拥有个性化入口。',
    },
    {
      icon: <Zap className="w-8 h-8 text-emerald-500" />,
      title: '极速解析生效',
      description: '基于 Cloudflare 强大的全球网络，DNS 记录秒级同步，告别漫长等待。',
    },
    {
      icon: <Shield className="w-8 h-8 text-cyan-500" />,
      title: '安全稳定可靠',
      description: '享受企业级 DDoS 防护与免费 SSL 证书，为你的网站保驾护航。',
    },
    {
      icon: <Rocket className="w-8 h-8 text-sky-500" />,
      title: '极简配置体验',
      description: '专为 Linux Do 佬友打造的现代化控制台，一键添加解析，小白也能轻松上手。',
    },
  ];

  return (
    <div className="max-w-6xl mx-auto pt-32 pb-24 px-6">
      <div className="text-center mb-16">
        <motion.div
          initial={{ scale: 0.8, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ duration: 0.6, type: 'spring' }}
          className="inline-block mb-4 px-4 py-1.5 rounded-full bg-white/30 dark:bg-black/30 border border-white/40 dark:border-white/10 text-teal-600 dark:text-teal-300 font-medium text-sm backdrop-blur-md"
        >
          ✨ 欢迎来到二次元的奇妙世界
        </motion.div>
        
        <motion.h1
          initial={{ y: 20, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          transition={{ duration: 0.6, delay: 0.1 }}
          className="text-5xl md:text-7xl font-extrabold mb-6 text-gray-900 dark:text-white tracking-tight"
        >
          <span className="bg-clip-text text-transparent bg-gradient-to-r from-teal-500 via-emerald-500 to-cyan-500">
            佬友空间
          </span>
          <br />
          连接你的无限可能
        </motion.h1>
        
        <motion.p
          initial={{ y: 20, opacity: 0 }}
          animate={{ y: 0, opacity: 1 }}
          transition={{ duration: 0.6, delay: 0.2 }}
          className="text-lg md:text-xl text-gray-700 dark:text-gray-200 max-w-2xl mx-auto leading-relaxed backdrop-blur-sm bg-white/10 dark:bg-black/10 p-4 rounded-2xl"
        >
          LinuxDoSpace 是专为 Linux Do 社区成员打造的免费二级域名分发平台。
          在这里，你可以轻松获取专属域名，开启你的极客之旅。
        </motion.p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {features.map((feature, index) => (
          <div key={index}>
            <GlassCard delay={0.3 + index * 0.1} className="hover:bg-white/50 dark:hover:bg-black/50 transition-colors group">
              <div className="flex items-start gap-4">
                <div className="p-3 rounded-2xl bg-white/50 dark:bg-white/10 group-hover:scale-110 transition-transform duration-300">
                  {feature.icon}
                </div>
                <div>
                  <h3 className="text-xl font-bold mb-2 text-gray-900 dark:text-white">{feature.title}</h3>
                  <p className="text-gray-600 dark:text-gray-300 leading-relaxed">{feature.description}</p>
                </div>
              </div>
            </GlassCard>
          </div>
        ))}
      </div>
    </div>
  );
}
