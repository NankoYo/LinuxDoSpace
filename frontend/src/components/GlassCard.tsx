import { ReactNode } from 'react';
import { motion } from 'motion/react';

interface GlassCardProps {
  children: ReactNode;
  className?: string;
  delay?: number;
}

export function GlassCard({ children, className = '', delay = 0 }: GlassCardProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5, delay }}
      className={`backdrop-blur-xl bg-white/40 dark:bg-black/40 border border-white/30 dark:border-white/10 shadow-xl rounded-3xl p-6 ${className}`}
    >
      {children}
    </motion.div>
  );
}
