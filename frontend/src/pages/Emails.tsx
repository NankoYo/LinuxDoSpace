import { useEffect, useMemo, useState } from 'react';
import { AnimatePresence, motion } from 'motion/react';
import {
  AlertCircle,
  ArrowRight,
  CheckCircle2,
  LoaderCircle,
  Mail,
  ShieldCheck,
  ShieldQuestion,
  XCircle,
} from 'lucide-react';
import { GlassCard } from '../components/GlassCard';
import {
  APIError,
  listMyEmailRoutes,
  listMyPermissions,
  submitPermissionApplication,
  upsertCatchAllEmailRoute,
} from '../lib/api';
import type { User, UserEmailRoute, UserPermission } from '../types/api';

// EmailsProps describes the authenticated session state required by the real
// user-facing email page.
interface EmailsProps {
  authenticated: boolean;
  sessionLoading: boolean;
  user?: User;
  csrfToken?: string;
  onLogin: () => void;
}

const emailCatchAllPermissionKey = 'email_catch_all';

// Emails renders the catch-all mailbox application flow and the forwarding
// target configuration form without changing the existing site shell.
export function Emails({ authenticated, sessionLoading, user, csrfToken, onLogin }: EmailsProps) {
  const [permission, setPermission] = useState<UserPermission | null>(null);
  const [route, setRoute] = useState<UserEmailRoute | null>(null);
  const [targetEmail, setTargetEmail] = useState('');
  const [enabled, setEnabled] = useState(false);
  const [loading, setLoading] = useState(false);
  const [applying, setApplying] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [isPledgeModalOpen, setIsPledgeModalOpen] = useState(false);

  // canEditRoute centralizes the condition that decides whether the settings
  // form should be writable for the current session.
  const canEditRoute = Boolean(route?.can_manage && permission?.can_manage_route);

  // statusBadge keeps the status wording and tone consistent across all cards.
  const statusBadge = useMemo(() => describePermissionStatus(permission?.status ?? 'not_requested'), [permission?.status]);

  useEffect(() => {
    if (!authenticated) {
      setPermission(null);
      setRoute(null);
      setTargetEmail('');
      setEnabled(false);
      setError('');
      setNotice('');
      setIsPledgeModalOpen(false);
      return;
    }

    void loadData();
  }, [authenticated]);

  useEffect(() => {
    setTargetEmail(route?.target_email ?? '');
    setEnabled(route?.enabled ?? false);
  }, [route?.id, route?.target_email, route?.enabled]);

  // loadData refreshes both the permission card and the route placeholder so the
  // page stays consistent after apply/save operations.
  async function loadData(): Promise<void> {
    try {
      setLoading(true);
      const [permissions, routes] = await Promise.all([listMyPermissions(), listMyEmailRoutes()]);
      setPermission(permissions.find((item) => item.key === emailCatchAllPermissionKey) ?? null);
      setRoute(routes.find((item) => item.permission_key === emailCatchAllPermissionKey) ?? null);
      setError('');
    } catch (loadError) {
      setError(readableErrorMessage(loadError, '无法加载邮箱权限与转发配置。'));
    } finally {
      setLoading(false);
    }
  }

  // handleApply submits the permission application after the user confirms the pledge text.
  async function handleApply(): Promise<void> {
    if (!csrfToken) {
      setError('当前会话缺少 CSRF Token，请刷新后重试。');
      return;
    }

    try {
      setApplying(true);
      const nextPermission = await submitPermissionApplication({ key: emailCatchAllPermissionKey }, csrfToken);
      setPermission(nextPermission);
      await loadData();
      setNotice(nextPermission.status === 'approved' ? '权限申请已自动通过，现在可以设置转发邮箱。' : '权限申请已提交，等待管理员审核。');
      setIsPledgeModalOpen(false);
      setError('');
    } catch (applyError) {
      setError(readableErrorMessage(applyError, '提交权限申请失败。'));
    } finally {
      setApplying(false);
    }
  }

  // handleSave stores the current forwarding target after the permission has been approved.
  async function handleSave(): Promise<void> {
    if (!csrfToken) {
      setError('当前会话缺少 CSRF Token，请刷新后重试。');
      return;
    }

    try {
      setSaving(true);
      const nextRoute = await upsertCatchAllEmailRoute(
        {
          target_email: targetEmail,
          enabled,
        },
        csrfToken,
      );
      setRoute(nextRoute);
      setNotice('邮箱泛解析转发设置已保存。');
      setError('');
    } catch (saveError) {
      setError(readableErrorMessage(saveError, '保存邮箱转发设置失败。'));
    } finally {
      setSaving(false);
    }
  }

  if (!authenticated) {
    return (
      <div className="mx-auto max-w-6xl px-6 pb-24 pt-32">
        <motion.div initial={{ y: 18, opacity: 0 }} animate={{ y: 0, opacity: 1 }} className="mb-10 text-center">
          <div className="mb-4 inline-flex items-center justify-center rounded-full bg-teal-100 p-3 text-teal-600 dark:bg-teal-900/30 dark:text-teal-300">
            <Mail size={32} />
          </div>
          <h1 className="mb-4 text-3xl font-extrabold text-gray-900 dark:text-white md:text-4xl">邮箱泛解析</h1>
          <p className="mx-auto max-w-3xl text-lg text-gray-700 dark:text-gray-300">
            登录后即可查看你自己的邮箱泛解析权限状态，并在获批后把 `catch-all@用户名.linuxdo.space` 转发到指定邮箱。
          </p>
        </motion.div>

        <GlassCard className="mx-auto max-w-3xl text-center">
          <div className="mb-4 inline-flex items-center justify-center rounded-full bg-white/60 p-3 text-teal-600 dark:bg-white/10 dark:text-teal-300">
            <ShieldQuestion size={28} />
          </div>
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white">先登录，再申请邮箱泛解析权限</h2>
          <p className="mx-auto mt-4 max-w-2xl text-sm leading-7 text-gray-600 dark:text-gray-300">
            该功能绑定到你与用户名同名的默认二级域名。系统会先核验当前账号的 Linux Do 信任等级与命名空间归属，再决定是否允许自动通过。
          </p>
          <button
            type="button"
            onClick={onLogin}
            className="mt-6 inline-flex items-center gap-2 rounded-2xl bg-[#1a1a1a] px-6 py-3 font-bold text-white shadow-lg transition-all hover:bg-black dark:bg-white dark:text-black dark:hover:bg-gray-100"
          >
            <ArrowRight size={18} />
            使用 Linux Do 登录
          </button>
        </GlassCard>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-6xl px-6 pb-24 pt-32">
      <motion.div initial={{ y: 18, opacity: 0 }} animate={{ y: 0, opacity: 1 }} className="mb-10 text-center">
        <div className="mb-4 inline-flex items-center justify-center rounded-full bg-teal-100 p-3 text-teal-600 dark:bg-teal-900/30 dark:text-teal-300">
          <Mail size={32} />
        </div>
        <h1 className="mb-4 text-3xl font-extrabold text-gray-900 dark:text-white md:text-4xl">邮箱泛解析</h1>
        <p className="mx-auto max-w-3xl text-lg text-gray-700 dark:text-gray-300">
          这里用于申请并配置 `catch-all@{user?.username ?? '你的用户名'}.linuxdo.space` 的转发目标。权限申请记录会同步进入管理员后台，便于后续监督与审计。
        </p>
      </motion.div>

      {sessionLoading || loading ? (
        <GlassCard className="mb-6 flex items-center gap-3 text-sm text-gray-600 dark:text-gray-300">
          <LoaderCircle size={18} className="animate-spin" />
          正在加载你的邮箱权限状态...
        </GlassCard>
      ) : null}

      {error ? (
        <div className="mb-6 rounded-2xl border border-red-300/50 bg-red-50/80 px-4 py-3 text-sm text-red-700 dark:border-red-500/20 dark:bg-red-950/30 dark:text-red-200">
          {error}
        </div>
      ) : null}

      {notice ? (
        <div className="mb-6 rounded-2xl border border-emerald-300/50 bg-emerald-50/80 px-4 py-3 text-sm text-emerald-700 dark:border-emerald-500/20 dark:bg-emerald-950/30 dark:text-emerald-200">
          {notice}
        </div>
      ) : null}

      <div className="grid grid-cols-1 gap-6 xl:grid-cols-3">
        <GlassCard className="xl:col-span-2">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <div className="mb-3 flex flex-wrap items-center gap-3">
                <span className="rounded-full bg-white/55 px-3 py-1 text-xs font-semibold text-gray-600 dark:bg-white/10 dark:text-gray-300">邮箱权限</span>
                <span className={`rounded-full px-3 py-1 text-xs font-semibold ${statusBadge.className}`}>{statusBadge.label}</span>
              </div>
              <h2 className="text-2xl font-bold text-gray-900 dark:text-white">{permission?.display_name ?? 'catch-all@<username>.linuxdo.space'}</h2>
              <div className="mt-3 font-mono text-sm text-teal-700 dark:text-teal-300">{permission?.target ?? route?.address ?? 'catch-all@<username>.linuxdo.space'}</div>
              <p className="mt-4 text-sm leading-7 text-gray-600 dark:text-gray-300">
                {permission?.description ?? '为与你用户名同名的默认二级域名开启一个 catch-all 邮箱转发入口。'}
              </p>
            </div>

            <div className="shrink-0 rounded-3xl border border-white/20 bg-white/35 p-5 text-sm text-gray-700 dark:border-white/10 dark:bg-black/20 dark:text-gray-200 lg:w-72">
              <div className="mb-3 text-xs uppercase tracking-[0.24em] text-gray-400">当前要求</div>
              <div className="space-y-3">
                <div className="rounded-2xl bg-white/50 px-4 py-3 dark:bg-white/5">
                  Linux Do 等级需大于等于 <span className="font-bold text-teal-600 dark:text-teal-300">{permission?.min_trust_level ?? 2}</span>
                </div>
                <div className="rounded-2xl bg-white/50 px-4 py-3 dark:bg-white/5">
                  需要先持有与用户名同名的默认二级域名
                </div>
                <div className="rounded-2xl bg-white/50 px-4 py-3 dark:bg-white/5">
                  {permission?.auto_approve ? '符合条件后会自动通过' : '符合条件后仍需管理员审核'}
                </div>
              </div>
            </div>
          </div>

          {permission?.eligibility_reasons?.length ? (
            <div className="mt-6 rounded-2xl border border-amber-300/40 bg-amber-50/80 p-4 text-sm text-amber-800 dark:border-amber-500/20 dark:bg-amber-950/25 dark:text-amber-200">
              <div className="mb-2 flex items-center gap-2 font-semibold">
                <AlertCircle size={16} />
                当前暂不满足申请条件
              </div>
              <div className="space-y-2 leading-6">
                {permission.eligibility_reasons.map((reason) => (
                  <div key={reason}>- {reason}</div>
                ))}
              </div>
            </div>
          ) : null}

          <div className="mt-6 flex flex-wrap gap-3">
            {permission?.can_apply ? (
              <button
                type="button"
                onClick={() => setIsPledgeModalOpen(true)}
                className="inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-emerald-500 to-teal-600 px-5 py-3 font-medium text-white shadow-lg transition-all hover:from-emerald-600 hover:to-teal-700"
              >
                <ShieldCheck size={18} />
                申请邮箱泛解析权限
              </button>
            ) : null}

            {permission?.status === 'approved' ? (
              <div className="inline-flex items-center gap-2 rounded-2xl bg-emerald-100 px-4 py-3 text-sm font-medium text-emerald-700 dark:bg-emerald-900/25 dark:text-emerald-300">
                <CheckCircle2 size={18} />
                当前权限已生效
              </div>
            ) : null}

            {permission?.status === 'pending' ? (
              <div className="inline-flex items-center gap-2 rounded-2xl bg-amber-100 px-4 py-3 text-sm font-medium text-amber-700 dark:bg-amber-900/25 dark:text-amber-300">
                <LoaderCircle size={18} className="animate-spin" />
                已提交申请，等待审核
              </div>
            ) : null}

            {permission?.status === 'rejected' ? (
              <div className="inline-flex items-center gap-2 rounded-2xl bg-red-100 px-4 py-3 text-sm font-medium text-red-700 dark:bg-red-900/25 dark:text-red-300">
                <XCircle size={18} />
                上次申请未通过，可在满足条件后重新申请
              </div>
            ) : null}
          </div>
        </GlassCard>

        <GlassCard>
          <h2 className="text-xl font-bold text-gray-900 dark:text-white">申请与转发说明</h2>
          <div className="mt-4 space-y-3 text-sm leading-7 text-gray-600 dark:text-gray-300">
            <div className="rounded-2xl bg-white/40 px-4 py-3 dark:bg-white/5">1. 在本页手动提交权限申请，系统会写入管理员后台申请记录。</div>
            <div className="rounded-2xl bg-white/40 px-4 py-3 dark:bg-white/5">2. 申请通过后，你可以把 catch-all 邮件转发到任意合法邮箱。</div>
            <div className="rounded-2xl bg-white/40 px-4 py-3 dark:bg-white/5">3. 若管理员后续将权限改为待审核或拒绝，现有转发会自动停用。</div>
          </div>
        </GlassCard>

        <GlassCard className="xl:col-span-2">
          <div className="mb-5 flex items-center justify-between gap-4">
            <div>
              <h2 className="text-xl font-bold text-gray-900 dark:text-white">转发设置</h2>
              <p className="mt-1 text-sm text-gray-600 dark:text-gray-300">权限通过后，这里的设置会直接写入后端的邮箱转发配置。</p>
            </div>
            <div className={`rounded-full px-3 py-1 text-xs font-semibold ${canEditRoute ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/25 dark:text-emerald-300' : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300'}`}>
              {canEditRoute ? '可编辑' : '等待权限通过'}
            </div>
          </div>

          <div className="space-y-5">
            <div>
              <label className="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">邮箱地址</label>
              <div className="rounded-2xl border border-white/20 bg-white/50 px-4 py-3 font-mono text-sm text-teal-700 dark:border-white/10 dark:bg-black/20 dark:text-teal-300">
                {route?.address ?? permission?.target ?? `catch-all@${user?.username ?? 'username'}.linuxdo.space`}
              </div>
            </div>

            <div>
              <label className="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">转发到</label>
              <input
                type="email"
                value={targetEmail}
                onChange={(event) => setTargetEmail(event.target.value)}
                disabled={!canEditRoute}
                placeholder="例如：team@example.com"
                className="w-full rounded-2xl border border-white/20 bg-white/55 px-4 py-3 text-gray-900 outline-none transition focus:border-teal-400 focus:ring-2 focus:ring-teal-400/20 disabled:cursor-not-allowed disabled:opacity-60 dark:border-white/10 dark:bg-black/20 dark:text-white"
              />
            </div>

            <label className="flex items-center justify-between gap-4 rounded-2xl border border-white/20 bg-white/45 px-4 py-3 text-sm text-gray-700 dark:border-white/10 dark:bg-black/20 dark:text-gray-200">
              <div>
                <div className="font-medium">启用转发</div>
                <div className="mt-1 text-xs text-gray-500 dark:text-gray-400">关闭后保留目标邮箱，但 catch-all 转发将暂停。</div>
              </div>
              <input type="checkbox" checked={enabled} onChange={(event) => setEnabled(event.target.checked)} disabled={!canEditRoute} />
            </label>

            <button
              type="button"
              onClick={() => void handleSave()}
              disabled={!canEditRoute || targetEmail.trim() === '' || saving}
              className="inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-fuchsia-500 to-pink-500 px-5 py-3 font-medium text-white shadow-lg transition-all hover:from-fuchsia-600 hover:to-pink-600 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {saving ? <LoaderCircle size={18} className="animate-spin" /> : <ArrowRight size={18} />}
              {saving ? '保存中...' : '保存邮箱转发'}
            </button>
          </div>
        </GlassCard>

        <GlassCard className="overflow-hidden p-0 xl:col-span-1">
          <div className="border-b border-white/20 px-6 py-5 dark:border-white/10">
            <h2 className="text-xl font-bold text-gray-900 dark:text-white">当前记录</h2>
            <p className="mt-1 text-sm text-gray-600 dark:text-gray-300">这里只展示邮箱地址、状态与转发目标，不涉及其他敏感配置。</p>
          </div>

          <div className="space-y-4 p-6 text-sm">
            <div>
              <div className="mb-1 text-xs uppercase tracking-[0.24em] text-gray-400">邮箱地址</div>
              <div className="font-mono text-teal-700 dark:text-teal-300">{route?.address ?? permission?.target ?? '-'}</div>
            </div>
            <div>
              <div className="mb-1 text-xs uppercase tracking-[0.24em] text-gray-400">转发目标</div>
              <div className="break-all text-gray-700 dark:text-gray-200">{route?.target_email || '尚未设置'}</div>
            </div>
            <div>
              <div className="mb-1 text-xs uppercase tracking-[0.24em] text-gray-400">开关状态</div>
              <div className={`${route?.enabled ? 'text-emerald-700 dark:text-emerald-300' : 'text-slate-500 dark:text-slate-400'}`}>{route?.enabled ? '已启用' : '未启用'}</div>
            </div>
            <div>
              <div className="mb-1 text-xs uppercase tracking-[0.24em] text-gray-400">最近更新</div>
              <div className="text-gray-500 dark:text-gray-400">{route?.updated_at ? formatDate(route.updated_at) : '尚未写入配置'}</div>
            </div>
          </div>
        </GlassCard>
      </div>

      <AnimatePresence>
        {isPledgeModalOpen && permission ? (
          <div className="fixed inset-0 z-50 flex items-center justify-center px-4">
            <button type="button" className="absolute inset-0 bg-black/45 backdrop-blur-sm" onClick={() => setIsPledgeModalOpen(false)} aria-label="关闭承诺书弹窗" />
            <motion.div initial={{ opacity: 0, y: 16, scale: 0.97 }} animate={{ opacity: 1, y: 0, scale: 1 }} exit={{ opacity: 0, y: 16, scale: 0.97 }} className="relative z-10 w-full max-w-2xl">
              <GlassCard className="border-white/35 bg-white/85 p-6 dark:bg-slate-950/80">
                <h2 className="text-2xl font-bold text-gray-900 dark:text-white">提交邮箱泛解析申请</h2>
                <p className="mt-3 text-sm leading-7 text-gray-600 dark:text-gray-300">
                  你即将申请 {permission.target} 的 catch-all 邮箱权限。确认提交后，系统会将下面这段承诺作为申请理由直接写入后台申请记录。
                </p>
                <div className="mt-5 rounded-3xl border border-amber-300/40 bg-amber-50/80 p-5 text-sm leading-7 text-amber-900 dark:border-amber-500/20 dark:bg-amber-950/25 dark:text-amber-100">
                  {permission.pledge_text}
                </div>
                <div className="mt-6 flex gap-3">
                  <button type="button" onClick={() => setIsPledgeModalOpen(false)} className="flex-1 rounded-2xl bg-slate-100 px-4 py-3 font-medium text-slate-700 dark:bg-slate-800 dark:text-slate-100">
                    取消
                  </button>
                  <button type="button" onClick={() => void handleApply()} disabled={applying} className="flex-1 rounded-2xl bg-gradient-to-r from-emerald-500 to-teal-600 px-4 py-3 font-medium text-white shadow-lg disabled:cursor-not-allowed disabled:opacity-60">
                    {applying ? '提交中...' : '确认提交申请'}
                  </button>
                </div>
              </GlassCard>
            </motion.div>
          </div>
        ) : null}
      </AnimatePresence>
    </div>
  );
}

// describePermissionStatus maps the backend status to a presentational label.
function describePermissionStatus(status: UserPermission['status']) {
  switch (status) {
    case 'approved':
      return {
        label: '已通过',
        className: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/25 dark:text-emerald-300',
      };
    case 'pending':
      return {
        label: '待审核',
        className: 'bg-amber-100 text-amber-700 dark:bg-amber-900/25 dark:text-amber-300',
      };
    case 'rejected':
      return {
        label: '未通过',
        className: 'bg-red-100 text-red-700 dark:bg-red-900/25 dark:text-red-300',
      };
    default:
      return {
        label: '尚未申请',
        className: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300',
      };
  }
}

// readableErrorMessage extracts the most useful message from one thrown request error.
function readableErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof APIError) {
    return error.message;
  }
  if (error instanceof Error && error.message.trim() !== '') {
    return error.message;
  }
  return fallback;
}

// formatDate renders timestamps with a single locale-aware formatter.
function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value));
}
