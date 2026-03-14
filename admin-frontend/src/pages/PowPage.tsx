import { useEffect, useMemo, useState } from 'react';
import { Cpu, LoaderCircle, Pickaxe, Settings2, Sparkles } from 'lucide-react';

import { APIError, getAdminPOWSettings, updateAdminPOWBenefit, updateAdminPOWDifficulty, updateAdminPOWGlobalSettings } from '../lib/api';
import { AdminSwitch } from '../components/AdminSwitch';
import { GlassCard } from '../components/GlassCard';
import type { AdminPOWBenefitSettings, AdminPOWDifficultySettings, AdminPOWGlobalSettings, AdminPOWSettings } from '../types/admin';

interface PowPageProps {
  csrfToken: string;
}

interface GlobalDraft {
  enabled: boolean;
  default_daily_completion_limit: number;
  base_reward_min: number;
  base_reward_max: number;
}

function formatDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
}

// PowPage renders the administrator-facing proof-of-work configuration center.
export function PowPage({ csrfToken }: PowPageProps) {
  const [settings, setSettings] = useState<AdminPOWSettings | null>(null);
  const [globalDraft, setGlobalDraft] = useState<GlobalDraft | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [savingGlobal, setSavingGlobal] = useState(false);
  const [savingBenefitKey, setSavingBenefitKey] = useState('');
  const [savingDifficultyValue, setSavingDifficultyValue] = useState<number | null>(null);

  const enabledBenefitCount = useMemo(
    () => settings?.benefits.filter((item) => item.enabled).length ?? 0,
    [settings?.benefits],
  );
  const enabledDifficultyCount = useMemo(
    () => settings?.difficulties.filter((item) => item.enabled).length ?? 0,
    [settings?.difficulties],
  );

  useEffect(() => {
    void loadSettings();
  }, []);

  async function loadSettings(): Promise<void> {
    try {
      setLoading(true);
      const nextSettings = await getAdminPOWSettings();
      setSettings(nextSettings);
      setGlobalDraft({
        enabled: nextSettings.global.enabled,
        default_daily_completion_limit: nextSettings.global.default_daily_completion_limit,
        base_reward_min: nextSettings.global.base_reward_min,
        base_reward_max: nextSettings.global.base_reward_max,
      });
      setError('');
    } catch (loadError) {
      setError(loadError instanceof APIError ? loadError.message : '加载 PoW 配置失败。');
    } finally {
      setLoading(false);
    }
  }

  async function saveGlobalSettings(): Promise<void> {
    if (!globalDraft) {
      return;
    }

    try {
      setSavingGlobal(true);
      const updated = await updateAdminPOWGlobalSettings(globalDraft, csrfToken);
      setSettings((current) => current ? { ...current, global: updated } : current);
      setGlobalDraft({
        enabled: updated.enabled,
        default_daily_completion_limit: updated.default_daily_completion_limit,
        base_reward_min: updated.base_reward_min,
        base_reward_max: updated.base_reward_max,
      });
      setError('');
    } catch (saveError) {
      setError(saveError instanceof APIError ? saveError.message : '保存 PoW 全局设置失败。');
    } finally {
      setSavingGlobal(false);
    }
  }

  async function toggleBenefit(benefit: AdminPOWBenefitSettings): Promise<void> {
    try {
      setSavingBenefitKey(benefit.key);
      const updated = await updateAdminPOWBenefit(benefit.key, { enabled: !benefit.enabled }, csrfToken);
      setSettings((current) => current ? {
        ...current,
        benefits: current.benefits.map((item) => (item.key === updated.key ? updated : item)),
      } : current);
      setError('');
    } catch (toggleError) {
      setError(toggleError instanceof APIError ? toggleError.message : '保存福利开关失败。');
    } finally {
      setSavingBenefitKey('');
    }
  }

  async function toggleDifficulty(item: AdminPOWDifficultySettings): Promise<void> {
    try {
      setSavingDifficultyValue(item.difficulty);
      const updated = await updateAdminPOWDifficulty(item.difficulty, { enabled: !item.enabled }, csrfToken);
      setSettings((current) => current ? {
        ...current,
        difficulties: current.difficulties.map((entry) => (entry.difficulty === updated.difficulty ? updated : entry)),
      } : current);
      setError('');
    } catch (toggleError) {
      setError(toggleError instanceof APIError ? toggleError.message : '保存难度开关失败。');
    } finally {
      setSavingDifficultyValue(null);
    }
  }

  return (
    <div className="mx-auto max-w-7xl">
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div className="flex items-center gap-3">
          <div className="rounded-2xl bg-cyan-100 p-3 text-cyan-600 dark:bg-cyan-900/30 dark:text-cyan-300">
            <Pickaxe size={28} />
          </div>
          <div>
            <h1 className="text-3xl font-bold text-slate-900 dark:text-white">PoW 福利</h1>
            <p className="mt-1 text-sm text-slate-500 dark:text-slate-300">统一管理 PoW 功能总开关、基础奖励范围、福利项目开关与可用难度级别。</p>
          </div>
        </div>

        <div className="grid gap-3 sm:grid-cols-3">
          <SummaryPill label="功能状态" value={settings?.global.enabled ? '已启用' : '已关闭'} accent={settings?.global.enabled ? 'cyan' : 'slate'} />
          <SummaryPill label="已启用福利" value={`${enabledBenefitCount} 项`} accent="emerald" />
          <SummaryPill label="已启用难度" value={`${enabledDifficultyCount} 档`} accent="amber" />
        </div>
      </div>

      {error ? (
        <div className="mb-5 rounded-2xl border border-red-300/50 bg-red-50/80 px-4 py-3 text-sm text-red-700 dark:border-red-500/20 dark:bg-red-950/30 dark:text-red-200">
          {error}
        </div>
      ) : null}

      {loading ? (
        <GlassCard className="p-8 text-center text-sm text-slate-500 dark:text-slate-300">正在加载 PoW 配置...</GlassCard>
      ) : null}

      {!loading && settings && globalDraft ? (
        <>
          <div className="mb-8 grid gap-5 xl:grid-cols-[minmax(0,1.05fr)_minmax(320px,0.95fr)]">
            <GlassCard className="p-6">
              <div className="mb-5 flex items-start justify-between gap-4">
                <div>
                  <div className="mb-2 inline-flex items-center gap-2 rounded-full bg-white/45 px-3 py-1 text-xs font-semibold text-slate-600 dark:bg-white/10 dark:text-slate-300">
                    <Settings2 size={14} />
                    全局配置
                  </div>
                  <h2 className="text-xl font-bold text-slate-900 dark:text-white">PoW 总控</h2>
                  <p className="mt-2 text-sm leading-7 text-slate-600 dark:text-slate-300">
                    控制整套 PoW 福利是否开放、每位用户默认每日最多可成功领取几次，以及解题完成后基础奖励的随机区间。
                  </p>
                </div>
                <span className={`rounded-full px-3 py-1 text-xs font-semibold ${globalDraft.enabled ? 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-300' : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300'}`}>
                  {globalDraft.enabled ? '功能已开启' : '功能已关闭'}
                </span>
              </div>

              <AdminSwitch
                checked={globalDraft.enabled}
                onCheckedChange={(checked) => setGlobalDraft((current) => current ? { ...current, enabled: checked } : current)}
                label="启用 PoW 功能"
                description="关闭后，前台将不能继续开始新的 PoW 福利解题流程。"
                accent="cyan"
                className="border-white/20 bg-white/40 dark:border-white/10 dark:bg-black/20"
              />

              <div className="mt-4 grid gap-4 md:grid-cols-3">
                <NumericCard
                  label="默认每日成功次数"
                  value={globalDraft.default_daily_completion_limit}
                  min={1}
                  onChange={(value) => setGlobalDraft((current) => current ? { ...current, default_daily_completion_limit: value } : current)}
                />
                <NumericCard
                  label="基础奖励最小值"
                  value={globalDraft.base_reward_min}
                  min={1}
                  onChange={(value) => setGlobalDraft((current) => current ? { ...current, base_reward_min: value } : current)}
                />
                <NumericCard
                  label="基础奖励最大值"
                  value={globalDraft.base_reward_max}
                  min={1}
                  onChange={(value) => setGlobalDraft((current) => current ? { ...current, base_reward_max: value } : current)}
                />
              </div>

              <div className="mt-5 flex items-center justify-between gap-4">
                <div className="text-xs text-slate-500 dark:text-slate-400">最近更新：{formatDate(settings.global.updated_at)}</div>
                <button
                  type="button"
                  onClick={() => void saveGlobalSettings()}
                  disabled={savingGlobal}
                  className="inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-cyan-500 to-blue-600 px-4 py-2 text-sm font-medium text-white shadow-lg transition hover:from-cyan-600 hover:to-blue-700 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {savingGlobal ? <LoaderCircle size={16} className="animate-spin" /> : <Sparkles size={16} />}
                  {savingGlobal ? '保存中...' : '保存全局配置'}
                </button>
              </div>
            </GlassCard>

            <GlassCard className="p-6">
              <div className="mb-4 inline-flex items-center gap-2 rounded-full bg-white/45 px-3 py-1 text-xs font-semibold text-slate-600 dark:bg-white/10 dark:text-slate-300">
                <Cpu size={14} />
                当前解释
              </div>
              <div className="space-y-4 text-sm leading-7 text-slate-600 dark:text-slate-300">
                <div>
                  前台成功一次 PoW 福利后，最终奖励 = `随机基础奖励 × 难度倍率`。
                </div>
                <div>
                  目前难度倍率固定等于难度值本身，所以只需要控制难度是否开放，不需要再单独设置倍率。
                </div>
                <div>
                  用户级每日次数设置会覆盖这里的默认值。未设置覆盖时，自动继承“默认每日成功次数”。
                </div>
              </div>
            </GlassCard>
          </div>

          <div className="mb-8 grid gap-5 xl:grid-cols-2">
            {settings.benefits.map((benefit) => (
              <GlassCard key={benefit.key} className="p-6">
                <div className="mb-5 flex items-start justify-between gap-4">
                  <div>
                    <div className="mb-2 inline-flex items-center gap-2 rounded-full bg-white/45 px-3 py-1 text-xs font-semibold text-slate-600 dark:bg-white/10 dark:text-slate-300">
                      <Sparkles size={14} />
                      福利项目
                    </div>
                    <h2 className="text-xl font-bold text-slate-900 dark:text-white">{benefit.display_name}</h2>
                    <p className="mt-2 text-sm leading-7 text-slate-600 dark:text-slate-300">{benefit.description}</p>
                  </div>
                  <span className={`rounded-full px-3 py-1 text-xs font-semibold ${benefit.enabled ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/25 dark:text-emerald-300' : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300'}`}>
                    {benefit.enabled ? '已启用' : '已关闭'}
                  </span>
                </div>

                <AdminSwitch
                  checked={benefit.enabled}
                  onCheckedChange={() => void toggleBenefit(benefit)}
                  disabled={savingBenefitKey === benefit.key}
                  label="允许前台选择该福利"
                  description={`关闭后，前台不会再展示或允许申请 ${benefit.display_name}。当前奖励单位：${benefit.reward_unit}。`}
                  accent="cyan"
                  className="border-white/20 bg-white/40 dark:border-white/10 dark:bg-black/20"
                />

                <div className="mt-5 text-xs text-slate-500 dark:text-slate-400">最近更新：{formatDate(benefit.updated_at)}</div>
              </GlassCard>
            ))}
          </div>

          <div className="grid gap-5 md:grid-cols-2 xl:grid-cols-4">
            {settings.difficulties.map((item) => (
              <GlassCard key={item.difficulty} className="p-6">
                <div className="mb-4 flex items-center justify-between gap-3">
                  <div>
                    <div className="text-lg font-bold text-slate-900 dark:text-white">{item.label}</div>
                    <div className="mt-1 text-xs text-slate-500 dark:text-slate-400">奖励倍率 {item.reward_multiplier}x</div>
                  </div>
                  <span className={`rounded-full px-3 py-1 text-xs font-semibold ${item.enabled ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/25 dark:text-amber-300' : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300'}`}>
                    {item.enabled ? '可选' : '关闭'}
                  </span>
                </div>

                <div className="mb-4 text-sm leading-7 text-slate-600 dark:text-slate-300">{item.description}</div>
                <AdminSwitch
                  checked={item.enabled}
                  onCheckedChange={() => void toggleDifficulty(item)}
                  disabled={savingDifficultyValue === item.difficulty}
                  label="允许前台使用该难度"
                  accent="amber"
                  className="border-white/20 bg-white/40 dark:border-white/10 dark:bg-black/20"
                />
                <div className="mt-4 text-xs text-slate-500 dark:text-slate-400">最近更新：{formatDate(item.updated_at)}</div>
              </GlassCard>
            ))}
          </div>
        </>
      ) : null}
    </div>
  );
}

function SummaryPill({ label, value, accent }: { label: string; value: string; accent: 'cyan' | 'emerald' | 'amber' | 'slate' }) {
  const className = accent === 'cyan'
    ? 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/25 dark:text-cyan-300'
    : accent === 'emerald'
      ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/25 dark:text-emerald-300'
      : accent === 'amber'
        ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/25 dark:text-amber-300'
        : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300';

  return (
    <div className={`rounded-2xl px-4 py-3 ${className}`}>
      <div className="text-[11px] font-semibold uppercase tracking-[0.24em] opacity-75">{label}</div>
      <div className="mt-1 text-base font-semibold">{value}</div>
    </div>
  );
}

function NumericCard({
  label,
  value,
  min,
  onChange,
}: {
  label: string;
  value: number;
  min: number;
  onChange: (value: number) => void;
}) {
  return (
    <div className="rounded-2xl border border-white/20 bg-white/40 px-4 py-4 dark:border-white/10 dark:bg-black/20">
      <label className="mb-2 block text-sm font-medium text-slate-700 dark:text-slate-200">{label}</label>
      <input
        type="number"
        min={min}
        step={1}
        value={value}
        onChange={(event) => onChange(Math.max(min, Number(event.target.value) || min))}
        className="w-full rounded-2xl border border-slate-200 bg-white/70 px-4 py-3 outline-none focus:border-cyan-400 focus:ring-2 focus:ring-cyan-400/20 dark:border-slate-700 dark:bg-black/35 dark:text-white"
      />
    </div>
  );
}
