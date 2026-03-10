import { useEffect, useRef, useState } from 'react';
import { AnimatePresence, motion } from 'motion/react';
import { ChevronDown } from 'lucide-react';

// GlassSelectOption 描述玻璃态下拉框中的一个可选项。
// 该类型保持极简，只承载后续页面渲染当前所需的值和展示文本。
export interface GlassSelectOption {
  value: string;
  label: string;
}

// GlassSelectProps 描述自定义下拉组件的输入契约。
// 当前组件用于权限页的前端预览表单，因此不额外引入复杂的受控/非受控分支。
interface GlassSelectProps {
  options: GlassSelectOption[];
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
}

// GlassSelect 提供和当前站点玻璃态风格一致的下拉选择器。
// 设计稿中有这个组件，而现有前端源码缺失，所以需要补回正式实现。
export function GlassSelect({
  options,
  value,
  onChange,
  placeholder = '请选择',
  disabled = false,
}: GlassSelectProps) {
  // isOpen 控制下拉列表是否展开。
  const [isOpen, setIsOpen] = useState(false);

  // containerRef 用于判断点击是否发生在组件外部，从而自动关闭面板。
  const containerRef = useRef<HTMLDivElement>(null);

  // selectedOption 表示当前 value 对应的选项，便于按钮区域展示标签文本。
  const selectedOption = options.find((option) => option.value === value);

  // 点击组件外部时关闭下拉层，避免移动端和桌面端都出现遮挡残留。
  useEffect(() => {
    function handlePointerDown(event: MouseEvent): void {
      if (!containerRef.current?.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }

    function handleEscape(event: KeyboardEvent): void {
      if (event.key === 'Escape') {
        setIsOpen(false);
      }
    }

    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('keydown', handleEscape);

    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('keydown', handleEscape);
    };
  }, []);

  return (
    <div ref={containerRef} className="relative w-full">
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        disabled={disabled}
        onClick={() => setIsOpen((current) => !current)}
        className={[
          'w-full flex items-center justify-between gap-3 px-4 py-3 rounded-xl bg-white/50 dark:bg-black/50 border border-gray-200 dark:border-gray-700 focus:outline-none focus:ring-2 focus:ring-teal-500 text-gray-900 dark:text-white transition-all backdrop-blur-md',
          disabled ? 'cursor-not-allowed opacity-70' : '',
        ].join(' ').trim()}
      >
        <span className="truncate text-left">{selectedOption?.label ?? placeholder}</span>
        <motion.span animate={{ rotate: isOpen ? 180 : 0 }} transition={{ duration: 0.2 }}>
          <ChevronDown size={18} className="text-gray-500 dark:text-gray-400" />
        </motion.span>
      </button>

      <AnimatePresence>
        {isOpen && (
          <motion.div
            initial={{ opacity: 0, y: -8, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -8, scale: 0.98 }}
            transition={{ duration: 0.2 }}
            className="absolute z-50 mt-2 w-full rounded-2xl bg-white/85 dark:bg-gray-950/80 backdrop-blur-xl border border-white/40 dark:border-white/10 shadow-2xl overflow-hidden"
          >
            <div className="max-h-60 overflow-y-auto py-1 custom-scrollbar" role="listbox">
              {options.map((option) => {
                const isSelected = option.value === value;

                return (
                  <button
                    key={option.value}
                    type="button"
                    role="option"
                    aria-selected={isSelected}
                    onClick={() => {
                      onChange(option.value);
                      setIsOpen(false);
                    }}
                    className={`w-full text-left px-4 py-3 transition-colors ${
                      isSelected
                        ? 'bg-teal-500/10 dark:bg-teal-500/20 text-teal-700 dark:text-teal-300 font-medium'
                        : 'text-gray-700 dark:text-gray-300 hover:bg-black/5 dark:hover:bg-white/10'
                    }`}
                  >
                    {option.label}
                  </button>
                );
              })}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
