import { useEffect, useState, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { AnimatePresence, motion } from 'motion/react';
import { X } from 'lucide-react';

// GlassModalProps 描述统一玻璃态弹窗的输入契约。
// 这个组件抽离自 DNS 记录弹窗，供高级功能和常规表单共用，避免再次出现交互风格分裂。
interface GlassModalProps {
  open: boolean;
  title: string;
  onClose: () => void;
  children: ReactNode;
  footer?: ReactNode;
  maxWidthClassName?: string;
}

// GlassModal 统一承载站内主操作弹窗的遮罩、面板、动效与关闭按钮。
// 任何“需要单独聚焦完成一次操作”的交互都应优先复用它，而不是再写折叠表单。
export function GlassModal({
  open,
  title,
  onClose,
  children,
  footer,
  maxWidthClassName = 'max-w-md',
}: GlassModalProps) {
  // mounted 保证 portal 只在浏览器环境挂载，避免访问 document 时报错。
  const [mounted, setMounted] = useState(false);

  // 组件挂载后再允许弹窗渲染到 document.body。
  useEffect(() => {
    setMounted(true);
    return () => setMounted(false);
  }, []);

  // 弹窗打开时锁定 body 滚动，避免背景内容和遮罩一起滚动。
  useEffect(() => {
    if (!mounted || !open || typeof document === 'undefined') {
      return;
    }

    const originalOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = originalOverflow;
    };
  }, [mounted, open]);

  if (!mounted || typeof document === 'undefined') {
    return null;
  }

  return createPortal(
    <AnimatePresence>
      {open ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center px-4">
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0 bg-black/40 backdrop-blur-sm"
            onClick={onClose}
          />

          <motion.div
            initial={{ scale: 0.9, opacity: 0, y: 20 }}
            animate={{ scale: 1, opacity: 1, y: 0 }}
            exit={{ scale: 0.9, opacity: 0, y: 20 }}
            onClick={(event) => event.stopPropagation()}
            className={`relative w-full ${maxWidthClassName} rounded-3xl border border-white/20 bg-white/80 p-6 shadow-2xl backdrop-blur-xl dark:border-white/10 dark:bg-gray-900/80`}
          >
            <button
              onClick={onClose}
              className="absolute top-4 right-4 p-2 text-gray-500 transition-colors hover:text-gray-900 dark:hover:text-white"
            >
              <X size={20} />
            </button>

            <h2 className="mb-6 text-2xl font-bold text-gray-900 dark:text-white">{title}</h2>

            <div className="space-y-4">{children}</div>

            {footer ? <div className="mt-8 flex gap-3">{footer}</div> : null}
          </motion.div>
        </div>
      ) : null}
    </AnimatePresence>,
    document.body,
  );
}
