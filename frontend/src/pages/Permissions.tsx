import { useEffect, useMemo, useState } from 'react';
import { motion } from 'motion/react';
import { ArrowRight, CheckCircle2, Clock3, Key, Mail, ShieldAlert, Ticket, XCircle } from 'lucide-react';
import { GlassCard } from '../components/GlassCard';
import { APIError, listMyPermissions } from '../lib/api';
import type { User, UserPermission } from '../types/api';

// PermissionsProps describes the authenticated session state required by the
// public permissions page.
interface PermissionsProps {
  authenticated: boolean;
  sessionLoading: boolean;
  user?: User;
  onLogin: () => void;
  onOpenEmails: () => void;
}

const emailCatchAllPermissionKey = 'email_catch_all';

// Permissions renders the current permission snapshot for the signed-in user.
// The actual application action lives on the Emails page because that is where
// the user also reads the pledge text and configures the forwarding target.
export function Permissions({ authenticated, sessionLoading, user, onLogin, onOpenEmails }: PermissionsProps) {
  const [permission, setPermission] = useState<UserPermission | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // statusBadge keeps the permission-state copy consistent across the page.
  const statusBadge = useMemo(() => describePermissionStatus(permission?.status ?? 'not_requested'), [permission?.status]);

  useEffect(() => {
    if (!authenticated) {
      setPermission(null);
      setError('');
      return;
    }

    void loadPermissions();
  }, [authenticated]);

  // loadPermissions fetches the currently supported permission cards from the backend.
  async function loadPermissions(): Promise<void> {
    try {
      setLoading(true);
      const items = await listMyPermissions();
      setPermission(items.find((item) => item.key === emailCatchAllPermissionKey) ?? null);
      setError('');
    } catch (loadError) {
      setError(readableErrorMessage(loadError, '无法加载权限列表。'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="mx-auto max-w-6xl px-6 pb-24 pt-32">
      <motion.div initial={{ y: 18, opacity: 0 }} animate={{ y: 0, opacity: 1 }} className="mb-10 text-center">
        <div className="mb-4 inline-flex items-center justify-center rounded-full bg-emerald-100 p-3 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-300">
          <Key size={32} />
        </div>
        <h1 className="mb-4 text-3xl font-extrabold text-gray-900 dark:text-white md:text-4xl">权限中心</h1>
        <p className="mx-auto max-w-3xl text-lg text-gray-700 dark:text-gray-300">
          当前页面展示你已经拥有或正在申请中的高级能力。现阶段已接入的真实功能是邮箱泛解析权限，其他权限入口会在后续版本逐步开放。
        </p>
      </motion.div>

      {error ? (
        <div className="mb-6 rounded-2xl border border-red-300/50 bg-red-50/80 px-4 py-3 text-sm text-red-700 dark:border-red-500/20 dark:bg-red-950/30 dark:text-red-200">
          {error}
        </div>
      ) : null}

      {!authenticated ? (
        <GlassCard className="mx-auto max-w-3xl text-center">
          <div className="mb-4 inline-flex items-center justify-center rounded-full bg-white/60 p-3 text-emerald-600 dark:bg-white/10 dark:text-emerald-300">
            <ShieldAlert size={28} />
          </div>
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white">登录后查看你的权限状态</h2>
          <p className="mx-auto mt-4 max-w-2xl text-sm leading-7 text-gray-600 dark:text-gray-300">
            权限申请和审核记录与 Linux Do OAuth 身份绑定，未登录时不会展示任何个人申请信息。
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
      ) : (
        <>
          {(sessionLoading || loading) && (
            <GlassCard className="mb-6 text-sm text-gray-600 dark:text-gray-300">正在加载你的权限状态...</GlassCard>
          )}

          <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
            <GlassCard className="lg:col-span-2">
              <div className="mb-4 flex flex-wrap items-center gap-3">
                <span className="rounded-full bg-white/55 px-3 py-1 text-xs font-semibold text-gray-600 dark:bg-white/10 dark:text-gray-300">当前可用权限</span>
                <span className={`rounded-full px-3 py-1 text-xs font-semibold ${statusBadge.className}`}>{statusBadge.label}</span>
              </div>
              <h2 className="text-2xl font-bold text-gray-900 dark:text-white">{permission?.display_name ?? 'catch-all@<username>.linuxdo.space'}</h2>
              <div className="mt-3 font-mono text-sm text-emerald-700 dark:text-emerald-300">{permission?.target ?? `catch-all@${user?.username ?? 'username'}.linuxdo.space`}</div>
              <p className="mt-4 text-sm leading-7 text-gray-600 dark:text-gray-300">
                {permission?.description ?? '为与你用户名同名的默认二级域名开启一个 catch-all 邮箱转发入口。'}
              </p>

              <div className="mt-6 grid gap-4 md:grid-cols-2">
                <div className="rounded-3xl border border-white/20 bg-white/35 p-5 dark:border-white/10 dark:bg-black/20">
                  <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
                    <Ticket size={18} />
                    自动通过条件
                  </div>
                  <div className="text-sm leading-7 text-gray-600 dark:text-gray-300">
                    Linux Do 等级需至少达到 <span className="font-bold text-emerald-600 dark:text-emerald-300">{permission?.min_trust_level ?? 2}</span>，并且需要已经持有与用户名同名的默认二级域名。
                  </div>
                </div>

                <div className="rounded-3xl border border-white/20 bg-white/35 p-5 dark:border-white/10 dark:bg-black/20">
                  <div className="mb-2 flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
                    <Mail size={18} />
                    申请入口
                  </div>
                  <div className="text-sm leading-7 text-gray-600 dark:text-gray-300">
                    该权限需要在“邮箱泛解析”页面中手动点击申请，并确认承诺书后提交。申请记录会同步保留在管理员后台。
                  </div>
                  <button
                    type="button"
                    onClick={onOpenEmails}
                    className="mt-4 inline-flex items-center gap-2 rounded-2xl bg-gradient-to-r from-emerald-500 to-teal-600 px-5 py-3 font-medium text-white shadow-lg transition-all hover:from-emerald-600 hover:to-teal-700"
                  >
                    <ArrowRight size={18} />
                    前往邮箱页面
                  </button>
                </div>
              </div>

              {permission?.eligibility_reasons?.length ? (
                <div className="mt-6 rounded-2xl border border-amber-300/40 bg-amber-50/80 p-4 text-sm text-amber-800 dark:border-amber-500/20 dark:bg-amber-950/25 dark:text-amber-200">
                  <div className="mb-2 flex items-center gap-2 font-semibold">
                    <ShieldAlert size={16} />
                    当前不可直接申请
                  </div>
                  <div className="space-y-2 leading-6">
                    {permission.eligibility_reasons.map((reason) => (
                      <div key={reason}>- {reason}</div>
                    ))}
                  </div>
                </div>
              ) : null}
            </GlassCard>

            <GlassCard>
              <h2 className="text-xl font-bold text-gray-900 dark:text-white">状态概览</h2>
              <div className="mt-4 space-y-3 text-sm leading-7 text-gray-600 dark:text-gray-300">
                <div className="rounded-2xl bg-white/40 px-4 py-3 dark:bg-white/5">
                  当前状态：<span className="font-semibold text-gray-900 dark:text-white">{statusBadge.label}</span>
                </div>
                <div className="rounded-2xl bg-white/40 px-4 py-3 dark:bg-white/5">
                  自动通过：<span className="font-semibold text-gray-900 dark:text-white">{permission?.auto_approve ? '开启' : '关闭'}</span>
                </div>
                <div className="rounded-2xl bg-white/40 px-4 py-3 dark:bg-white/5">
                  当前账号：<span className="font-semibold text-gray-900 dark:text-white">{user?.username ?? '-'}</span>
                </div>
              </div>
            </GlassCard>
          </div>

          <GlassCard className="mt-6">
            <div className="mb-5 flex items-center gap-3">
              <div className="rounded-2xl bg-amber-100 p-3 text-amber-600 dark:bg-amber-900/30 dark:text-amber-300">
                <Clock3 size={24} />
              </div>
              <div>
                <h2 className="text-xl font-bold text-gray-900 dark:text-white">最新申请记录</h2>
                <p className="mt-1 text-sm text-gray-600 dark:text-gray-300">这里只显示当前权限的最新状态快照，管理员后台会保留审核轨迹。</p>
              </div>
            </div>

            {permission?.application ? (
              <div className="grid gap-4 lg:grid-cols-4">
                <div className="rounded-2xl border border-white/20 bg-white/35 px-4 py-4 dark:border-white/10 dark:bg-black/20">
                  <div className="mb-1 text-xs uppercase tracking-[0.24em] text-gray-400">申请状态</div>
                  <div className="flex items-center gap-2 font-semibold text-gray-900 dark:text-white">
                    {permission.application.status === 'approved' ? <CheckCircle2 size={18} className="text-emerald-500" /> : permission.application.status === 'rejected' ? <XCircle size={18} className="text-red-500" /> : <Clock3 size={18} className="text-amber-500" />}
                    {describePermissionStatus(permission.application.status).label}
                  </div>
                </div>
                <div className="rounded-2xl border border-white/20 bg-white/35 px-4 py-4 dark:border-white/10 dark:bg-black/20">
                  <div className="mb-1 text-xs uppercase tracking-[0.24em] text-gray-400">提交时间</div>
                  <div className="font-semibold text-gray-900 dark:text-white">{formatDate(permission.application.created_at)}</div>
                </div>
                <div className="rounded-2xl border border-white/20 bg-white/35 px-4 py-4 dark:border-white/10 dark:bg-black/20">
                  <div className="mb-1 text-xs uppercase tracking-[0.24em] text-gray-400">最近变更</div>
                  <div className="font-semibold text-gray-900 dark:text-white">{formatDate(permission.application.updated_at)}</div>
                </div>
                <div className="rounded-2xl border border-white/20 bg-white/35 px-4 py-4 dark:border-white/10 dark:bg-black/20">
                  <div className="mb-1 text-xs uppercase tracking-[0.24em] text-gray-400">审核备注</div>
                  <div className="text-sm text-gray-700 dark:text-gray-200">{permission.application.review_note || '暂无审核备注'}</div>
                </div>
              </div>
            ) : (
              <div className="rounded-2xl border border-dashed border-white/25 bg-white/25 px-5 py-8 text-sm text-gray-600 dark:border-white/10 dark:bg-black/15 dark:text-gray-300">
                当前还没有提交过该权限申请。需要时请前往邮箱页面发起申请。
              </div>
            )}
          </GlassCard>
        </>
      )}
    </div>
  );
}

// describePermissionStatus keeps wording and visual tone aligned with the email page.
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

// readableErrorMessage extracts the most relevant message from one request failure.
function readableErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof APIError) {
    return error.message;
  }
  if (error instanceof Error && error.message.trim() !== '') {
    return error.message;
  }
  return fallback;
}

// formatDate renders timestamps with one shared locale-aware formatter.
function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value));
}
