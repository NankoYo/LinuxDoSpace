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
          'relative h-7 w-14 shrink-0 rounded-full transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-teal-400/40',
          checked ? 'bg-teal-500' : 'bg-slate-300 dark:bg-slate-700',
          disabled ? 'cursor-not-allowed' : '',
        ].join(' ').trim()}
      >
        <span
          className={[
            'absolute top-1 h-5 w-5 rounded-full bg-white shadow transition-transform duration-200',
            checked ? 'translate-x-8' : 'translate-x-1',
          ].join(' ').trim()}
        />
      </button>
    </div>
  );
}
