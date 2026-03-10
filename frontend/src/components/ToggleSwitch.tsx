// ToggleSwitch 统一承载公共前端里的开关样式。
// 这样可以避免每个页面各自拼装一套“轨道 + 圆点”，从而减少渲染不一致问题。
interface ToggleSwitchProps {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  title: string;
  description?: string;
  disabled?: boolean;
  className?: string;
}

// ToggleSwitch 使用 button + role="switch" 实现可访问的开关控件。
// 相比直接渲染原生 checkbox，这种写法能完全受控于当前站点的玻璃态视觉风格。
export function ToggleSwitch({
  checked,
  onCheckedChange,
  title,
  description = '',
  disabled = false,
  className = '',
}: ToggleSwitchProps) {
  return (
    <div
      className={[
        'flex items-center justify-between gap-4 rounded-2xl border border-white/15 bg-white/35 p-4 dark:border-white/10 dark:bg-black/20',
        disabled ? 'opacity-70' : '',
        className,
      ].join(' ').trim()}
    >
      <div className="min-w-0 flex-1">
        <div className="text-sm font-semibold text-gray-900 dark:text-white">{title}</div>
        {description ? <div className="mt-1 text-sm text-gray-600 dark:text-gray-300">{description}</div> : null}
      </div>

      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={title}
        disabled={disabled}
        onClick={() => onCheckedChange(!checked)}
        className={[
          'relative inline-flex h-8 w-14 shrink-0 items-center rounded-full border p-0.5 transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-teal-400/40',
          checked
            ? 'border-teal-300/70 bg-teal-500 shadow-[0_8px_20px_rgba(20,184,166,0.28)]'
            : 'border-white/35 bg-slate-300/90 shadow-inner dark:border-white/15 dark:bg-slate-700/90',
          disabled ? 'cursor-not-allowed opacity-70' : 'hover:scale-[1.02]',
        ].join(' ').trim()}
      >
        <span
          className={[
            'absolute left-0.5 top-0.5 h-6 w-6 rounded-full bg-white shadow-[0_4px_12px_rgba(15,23,42,0.18)] transition-transform duration-200 will-change-transform',
            checked ? 'translate-x-7' : 'translate-x-0',
          ].join(' ').trim()}
        />
        <span className="sr-only">{checked ? '已开启' : '已关闭'}</span>
      </button>
    </div>
  );
}
