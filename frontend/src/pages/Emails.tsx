import { useState } from 'react';
import { AnimatePresence, motion } from 'motion/react';
import { ArrowRight, CheckCircle2, Info, Mail, Search, Trash2, XCircle } from 'lucide-react';
import confetti from 'canvas-confetti';
import { GlassCard } from '../components/GlassCard';

// EmailStatus 描述邮箱分发页查询卡片的当前状态。
// 当前页面仍是前端预览，因此这里只表达本地演示流程，不连接真实后端。
type EmailStatus = 'idle' | 'checking' | 'available' | 'taken';

// PreviewEmailRecord 表示邮箱分发页中的一条本地示例记录。
interface PreviewEmailRecord {
  id: number;
  prefix: string;
  target: string;
  updatedAt: string;
}

// reservedPrefixes 表示在预览态下固定不可申请的前缀。
const reservedPrefixes = ['admin', 'root', 'postmaster', 'support'];

// Emails 负责承接新 UI 里的“邮箱分发”页面。
// 当前保留完整视觉结构，但显式说明这是预览能力，避免误导为已上线后端功能。
export function Emails() {
  const [prefix, setPrefix] = useState('');
  const [target, setTarget] = useState('');
  const [status, setStatus] = useState<EmailStatus>('idle');
  const [records, setRecords] = useState<PreviewEmailRecord[]>([
    { id: 1, prefix: 'hello', target: 'contact@example.com', updatedAt: '2026-03-07' },
    { id: 2, prefix: 'infra', target: 'alerts@example.com', updatedAt: '2026-03-07' },
  ]);

  const normalizedPrefix = prefix.trim().toLowerCase();
  const hasConflict = records.some((record) => record.prefix === normalizedPrefix);

  // handleSearch 模拟设计稿中的“前缀可用性查询”反馈。
  function handleSearch(event: React.FormEvent<HTMLFormElement>): void {
    event.preventDefault();

    if (normalizedPrefix === '') {
      setStatus('idle');
      return;
    }

    setStatus('checking');
    window.setTimeout(() => {
      if (reservedPrefixes.includes(normalizedPrefix) || hasConflict) {
        setStatus('taken');
        return;
      }
      setStatus('available');
    }, 700);
  }

  // handleAddPreviewRecord 把当前输入写入本地示例数据，便于继续预览管理表格。
  function handleAddPreviewRecord(): void {
    if (status !== 'available' || target.trim() === '' || normalizedPrefix === '') {
      return;
    }

    setRecords((currentRecords) => [
      {
        id: Date.now(),
        prefix: normalizedPrefix,
        target: target.trim(),
        updatedAt: new Date().toISOString().slice(0, 10),
      },
      ...currentRecords,
    ]);
    setPrefix('');
    setTarget('');
    setStatus('idle');
    confetti({ particleCount: 80, spread: 70, origin: { y: 0.6 }, colors: ['#2dd4bf', '#34d399', '#0ea5e9'] });
  }

  // handleDeleteRecord 删除一条本地预览记录。
  function handleDeleteRecord(recordId: number): void {
    setRecords((currentRecords) => currentRecords.filter((record) => record.id !== recordId));
  }

  return (
    <div className="max-w-6xl mx-auto pt-32 pb-24 px-6">
      <motion.div initial={{ y: 20, opacity: 0 }} animate={{ y: 0, opacity: 1 }} className="mb-10 text-center">
        <div className="inline-flex items-center justify-center p-3 mb-4 rounded-full bg-teal-100 dark:bg-teal-900/30 text-teal-600 dark:text-teal-400">
          <Mail size={32} />
        </div>
        <h1 className="text-3xl md:text-4xl font-extrabold text-gray-900 dark:text-white mb-4">专属邮箱分发</h1>
        <p className="text-lg text-gray-700 dark:text-gray-300 max-w-3xl mx-auto">
          新设计稿页面已接入前端。当前仍为 UI 预览态，真实 Cloudflare Email Routing 后端尚未接入。
        </p>
      </motion.div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-6 items-start">
        <div className="xl:col-span-2 space-y-6">
          <GlassCard>
            <div className="flex items-start gap-3 mb-5">
              <div className="mt-0.5 p-2 rounded-2xl bg-amber-100/80 dark:bg-amber-950/30 text-amber-700 dark:text-amber-300">
                <Info size={18} />
              </div>
              <div>
                <div className="text-base font-bold text-gray-900 dark:text-white">当前为前端预览页</div>
                <div className="mt-1 text-sm text-gray-600 dark:text-gray-300 leading-relaxed">
                  查询、创建和删除都只影响浏览器内的本地示例数据，不会写入真实邮箱路由配置。
                </div>
              </div>
            </div>

            <form onSubmit={handleSearch} className="flex flex-col md:flex-row gap-4">
              <div className="relative flex-1">
                <input
                  type="text"
                  value={prefix}
                  onChange={(event) => {
                    setPrefix(event.target.value.replace(/\s+/g, ''));
                    setStatus('idle');
                  }}
                  placeholder="输入你想要的邮箱前缀"
                  className="w-full pl-4 pr-36 py-4 rounded-2xl bg-white/50 dark:bg-black/50 border border-white/40 dark:border-white/20 focus:outline-none focus:ring-2 focus:ring-teal-500 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 transition-all"
                />
                <div className="absolute right-4 top-1/2 -translate-y-1/2 text-gray-500 font-medium">@linuxdo.space</div>
              </div>
              <button type="submit" className="flex items-center justify-center gap-2 px-8 py-4 rounded-2xl bg-gradient-to-r from-teal-500 to-emerald-500 hover:from-teal-600 hover:to-emerald-600 text-white font-bold shadow-lg transition-all">
                <Search size={20} />
                查询
              </button>
            </form>

            <AnimatePresence mode="wait">
              {status !== 'idle' && (
                <motion.div key={status} initial={{ opacity: 0, y: -10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -10 }} className="mt-6 p-6 rounded-2xl bg-white/40 dark:bg-black/40 border border-white/20">
                  {status === 'checking' && (
                    <div className="flex items-center justify-center gap-3 text-teal-600 dark:text-teal-400 font-medium">
                      <div className="w-5 h-5 border-2 border-teal-500 border-t-transparent rounded-full animate-spin" />
                      正在检查邮箱前缀可用性...
                    </div>
                  )}

                  {status === 'taken' && (
                    <div className="text-center">
                      <div className="inline-flex items-center justify-center gap-2 text-red-600 dark:text-red-400 font-bold text-lg">
                        <XCircle size={20} />
                        当前前缀在预览规则中不可用
                      </div>
                      <p className="mt-3 text-sm text-gray-600 dark:text-gray-300">
                        这可能是保留前缀，或者和当前本地示例记录冲突。真实规则以后端上线版本为准。
                      </p>
                    </div>
                  )}

                  {status === 'available' && (
                    <div className="space-y-4">
                      <div className="text-center">
                        <div className="inline-flex items-center justify-center gap-2 text-emerald-600 dark:text-emerald-400 font-bold text-lg">
                          <CheckCircle2 size={20} />
                          {normalizedPrefix}@linuxdo.space 在预览规则中可用
                        </div>
                      </div>
                      <div className="grid grid-cols-1 md:grid-cols-[1fr_auto] gap-3">
                        <input
                          type="email"
                          value={target}
                          onChange={(event) => setTarget(event.target.value)}
                          placeholder="例如：team@example.com"
                          className="w-full px-4 py-3 rounded-2xl bg-white/50 dark:bg-black/50 border border-white/40 dark:border-white/20 focus:outline-none focus:ring-2 focus:ring-teal-500 text-gray-900 dark:text-white"
                        />
                        <button type="button" onClick={handleAddPreviewRecord} disabled={target.trim() === ''} className="flex items-center justify-center gap-2 px-6 py-3 rounded-2xl bg-gradient-to-r from-emerald-500 to-teal-600 hover:from-emerald-600 hover:to-teal-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium shadow-lg transition-all">
                          <ArrowRight size={18} />
                          加入预览记录
                        </button>
                      </div>
                    </div>
                  )}
                </motion.div>
              )}
            </AnimatePresence>
          </GlassCard>

          <GlassCard className="overflow-hidden p-0">
            <div className="p-6 border-b border-white/20 dark:border-white/10 flex items-center justify-between gap-4">
              <div>
                <h2 className="text-xl font-bold text-gray-900 dark:text-white">邮箱记录预览</h2>
                <p className="mt-1 text-sm text-gray-600 dark:text-gray-300">仅用于承接新 UI 中的表格布局和管理交互。</p>
              </div>
              <div className="rounded-2xl bg-white/35 dark:bg-black/30 border border-white/20 px-4 py-2 text-sm text-gray-700 dark:text-gray-200">当前本地条目：{records.length}</div>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b border-white/20 dark:border-white/10 bg-white/20 dark:bg-black/20">
                    <th className="p-4 font-semibold text-gray-900 dark:text-white">邮箱地址</th>
                    <th className="p-4 font-semibold text-gray-900 dark:text-white">转发目标</th>
                    <th className="p-4 font-semibold text-gray-900 dark:text-white">更新日期</th>
                    <th className="p-4 font-semibold text-gray-900 dark:text-white text-right">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {records.map((record, index) => (
                    <motion.tr key={record.id} initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: index * 0.04 }} className="border-b border-white/10 dark:border-white/5 hover:bg-white/30 dark:hover:bg-white/5 transition-colors">
                      <td className="p-4 font-semibold text-teal-700 dark:text-teal-300">{record.prefix}@linuxdo.space</td>
                      <td className="p-4 text-gray-700 dark:text-gray-200 break-all">{record.target}</td>
                      <td className="p-4 text-gray-500 dark:text-gray-400">{record.updatedAt}</td>
                      <td className="p-4 text-right">
                        <button type="button" onClick={() => handleDeleteRecord(record.id)} className="p-2 rounded-lg hover:bg-white/50 dark:hover:bg-white/10 text-red-600 dark:text-red-400 transition-colors" aria-label={`删除 ${record.prefix}`}>
                          <Trash2 size={16} />
                        </button>
                      </td>
                    </motion.tr>
                  ))}
                </tbody>
              </table>
            </div>

            {records.length === 0 && <div className="p-12 text-center text-gray-500 dark:text-gray-400">暂无本地预览记录。</div>}
          </GlassCard>
        </div>

        <div className="space-y-6">
          <GlassCard>
            <h2 className="text-xl font-bold text-gray-900 dark:text-white">计划中的真实能力</h2>
            <div className="mt-4 space-y-3 text-sm text-gray-600 dark:text-gray-300">
              <div className="rounded-2xl bg-white/35 dark:bg-black/25 border border-white/20 px-4 py-3">1. 真实检查邮箱前缀占用情况。</div>
              <div className="rounded-2xl bg-white/35 dark:bg-black/25 border border-white/20 px-4 py-3">2. 对接 Cloudflare Email Routing 的创建、更新和删除。</div>
              <div className="rounded-2xl bg-white/35 dark:bg-black/25 border border-white/20 px-4 py-3">3. 在配置中心展示真实路由状态与审计信息。</div>
            </div>
          </GlassCard>
        </div>
      </div>
    </div>
  );
}
