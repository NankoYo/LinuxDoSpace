import type { ReactNode } from 'react';

// GlassTooltipProps 描述玻璃态悬浮提示层的通用输入契约。
// content 使用换行分隔多行文本，children 则是触发悬浮层显示的卡片内容。
interface GlassTooltipProps {
  content?: string;
  children: ReactNode;
}

// GlassTooltip 为站内统计卡片提供统一的半透明圆角悬浮提示层。
// 当前实现只负责展示只读说明，不承担点击交互，因此保留 pointer-events-none
// 以避免提示层遮挡原有布局和鼠标移动路径。
export function GlassTooltip({ content, children }: GlassTooltipProps) {
  if (!content || content.trim() === '') {
    return <>{children}</>;
  }

  const lines = content
    .split('\n')
    .map((item) => item.trim())
    .filter((item) => item !== '');

  return (
    <div className="group relative">
      {children}

      <div className="pointer-events-none invisible absolute left-1/2 top-full z-40 mt-3 w-max max-w-[calc(100vw-2rem)] -translate-x-1/2 translate-y-1 opacity-0 transition-all duration-200 group-hover:visible group-hover:translate-y-0 group-hover:opacity-100 group-focus-within:visible group-focus-within:translate-y-0 group-focus-within:opacity-100">
        <div className="absolute left-1/2 top-0 h-3 w-3 -translate-x-1/2 -translate-y-1/2 rotate-45 border-l border-t border-white/25 bg-white/80 backdrop-blur-2xl dark:border-white/10 dark:bg-gray-950/80" />

        <div className="relative max-w-[20rem] rounded-2xl border border-white/25 bg-white/80 px-4 py-3 text-sm leading-7 text-gray-700 shadow-2xl backdrop-blur-2xl dark:border-white/10 dark:bg-gray-950/80 dark:text-gray-200">
          {lines.map((line, index) => (
            <div key={`${index}-${line}`}>{line}</div>
          ))}
        </div>
      </div>
    </div>
  );
}
