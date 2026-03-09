import { useEffect, useMemo, useState, type FormEvent } from 'react';
import { motion } from 'motion/react';
import { AlertCircle, ArrowRight, CheckCircle2, LoaderCircle, Mail, Search, ShieldCheck, XCircle } from 'lucide-react';
import { GlassCard } from '../components/GlassCard';
import { APIError, checkPublicEmailRouteAvailability, listMyEmailRoutes, listMyPermissions, submitPermissionApplication, upsertCatchAllEmailRoute, upsertDefaultEmailRoute } from '../lib/api';
import type { EmailRouteAvailabilityResult, User, UserEmailRoute, UserPermission } from '../types/api';

interface EmailsProps {
  authenticated: boolean;
  sessionLoading: boolean;
  user?: User;
  csrfToken?: string;
  onLogin: () => void;
}

type SearchStatus = 'idle' | 'checking' | 'available' | 'taken' | 'error';

const catchAllPermissionKey = 'email_catch_all';
const fallbackRootDomain = 'linuxdo.space';

export function Emails({ authenticated, sessionLoading, user, csrfToken, onLogin }: EmailsProps) {
  const [permission, setPermission] = useState<UserPermission | null>(null);
  const [routes, setRoutes] = useState<UserEmailRoute[]>([]);
  const [loading, setLoading] = useState(false);
  const [savingDefault, setSavingDefault] = useState(false);
  const [savingCatchAll, setSavingCatchAll] = useState(false);
  const [applying, setApplying] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [searchPrefix, setSearchPrefix] = useState('');
  const [searchStatus, setSearchStatus] = useState<SearchStatus>('idle');
  const [searchResult, setSearchResult] = useState<EmailRouteAvailabilityResult | null>(null);
  const [searchMessage, setSearchMessage] = useState('');
  const [defaultTarget, setDefaultTarget] = useState('');
  const [defaultEnabled, setDefaultEnabled] = useState(false);
  const [catchAllTarget, setCatchAllTarget] = useState('');
  const [catchAllEnabled, setCatchAllEnabled] = useState(false);

  const defaultRoute = useMemo(() => routes.find((item) => item.kind === 'default') ?? null, [routes]);
  const catchAllRoute = useMemo(() => routes.find((item) => item.kind === 'catch_all') ?? null, [routes]);
  const searchRootDomain = defaultRoute?.root_domain ?? searchResult?.root_domain ?? fallbackRootDomain;
  const normalizedUsername = normalizePrefix(user?.username ?? '');
  const tableRows = useMemo(() => {
    const items: UserEmailRoute[] = [];
    if (defaultRoute) items.push(defaultRoute);
    items.push(...routes.filter((item) => item.kind === 'custom'));
    if (catchAllRoute) items.push(catchAllRoute);
    return items;
  }, [catchAllRoute, defaultRoute, routes]);

  useEffect(() => {
    if (!authenticated) {
      setPermission(null);
      setRoutes([]);
      setDefaultTarget('');
      setDefaultEnabled(false);
      setCatchAllTarget('');
      setCatchAllEnabled(false);
      return;
    }
    void loadData();
  }, [authenticated]);

  useEffect(() => {
    setDefaultTarget(defaultRoute?.target_email ?? '');
    setDefaultEnabled(defaultRoute?.enabled ?? false);
  }, [defaultRoute?.id, defaultRoute?.target_email, defaultRoute?.enabled]);

  useEffect(() => {
    setCatchAllTarget(catchAllRoute?.target_email ?? '');
    setCatchAllEnabled(catchAllRoute?.enabled ?? false);
  }, [catchAllRoute?.id, catchAllRoute?.target_email, catchAllRoute?.enabled]);

  async function loadData(): Promise<void> {
    try {
      setLoading(true);
      const [permissions, nextRoutes] = await Promise.all([listMyPermissions(), listMyEmailRoutes()]);
      setPermission(permissions.find((item) => item.key === catchAllPermissionKey) ?? null);
      setRoutes(nextRoutes);
      setError('');
    } catch (loadError) {
      setError(readableErrorMessage(loadError, '无法加载邮箱分发数据。'));
    } finally {
      setLoading(false);
    }
  }

  async function handleSearch(event: FormEvent): Promise<void> {
    event.preventDefault();
    if (!searchPrefix.trim()) return;
    try {
      setSearchStatus('checking');
      const result = await checkPublicEmailRouteAvailability(searchRootDomain, searchPrefix);
      setSearchResult(result);
      setSearchStatus(result.available ? 'available' : 'taken');
      setSearchMessage(buildSearchMessage(result, normalizedUsername, authenticated));
    } catch (searchError) {
      setSearchResult(null);
      setSearchStatus('error');
      setSearchMessage(readableErrorMessage(searchError, '邮箱前缀可用性检查失败。'));
    }
  }

  async function saveDefault(targetEmail: string, enabled: boolean): Promise<void> {
    if (!csrfToken) return setError('当前会话缺少 CSRF Token，请刷新后重试。');
    try {
      setSavingDefault(true);
      const nextRoute = await upsertDefaultEmailRoute({ target_email: targetEmail, enabled }, csrfToken);
      setRoutes((current) => upsertRoute(current, nextRoute));
      setNotice(nextRoute.configured ? '默认邮箱转发设置已保存。' : '默认邮箱转发已清空。');
      setError('');
    } catch (saveError) {
      setError(readableErrorMessage(saveError, '保存默认邮箱转发失败。'));
    } finally {
      setSavingDefault(false);
    }
  }

  async function handleApplyCatchAll(): Promise<void> {
    if (!csrfToken) return setError('当前会话缺少 CSRF Token，请刷新后重试。');
    try {
      setApplying(true);
      const nextPermission = await submitPermissionApplication({ key: catchAllPermissionKey }, csrfToken);
      setPermission(nextPermission);
      await loadData();
      setNotice(nextPermission.status === 'approved' ? '邮箱泛解析权限已自动通过。' : '邮箱泛解析权限申请已提交。');
      setError('');
    } catch (applyError) {
      setError(readableErrorMessage(applyError, '提交邮箱泛解析申请失败。'));
    } finally {
      setApplying(false);
    }
  }

  async function handleSaveCatchAll(): Promise<void> {
    if (!csrfToken) return setError('当前会话缺少 CSRF Token，请刷新后重试。');
    try {
      setSavingCatchAll(true);
      const nextRoute = await upsertCatchAllEmailRoute({ target_email: catchAllTarget, enabled: catchAllEnabled }, csrfToken);
      setRoutes((current) => upsertRoute(current, nextRoute));
      setNotice('邮箱泛解析转发设置已保存。');
      setError('');
    } catch (saveError) {
      setError(readableErrorMessage(saveError, '保存邮箱泛解析转发失败。'));
    } finally {
      setSavingCatchAll(false);
    }
  }

  return (
    <div className="mx-auto max-w-6xl px-6 pb-24 pt-32">
      <motion.div initial={{ y: 20, opacity: 0 }} animate={{ y: 0, opacity: 1 }} className="mb-12 text-center">
        <div className="mb-4 inline-flex rounded-full bg-teal-100 p-3 text-teal-600 dark:bg-teal-900/30 dark:text-teal-300"><Mail size={32} /></div>
        <h1 className="mb-4 text-4xl font-extrabold text-gray-900 dark:text-white md:text-5xl">邮箱分发</h1>
        <p className="mx-auto max-w-3xl text-lg text-gray-700 dark:text-gray-300">默认邮箱是主功能，每位用户自动拥有一个与用户名同名的邮箱；邮箱泛解析只是附加权限。</p>
      </motion.div>
      {error ? <NoticeCard tone="error" message={error} /> : null}
      {notice ? <NoticeCard tone="success" message={notice} /> : null}
      <GlassCard className="mb-8">
        <div className="mb-5 flex items-center gap-3"><Search className="text-teal-500" size={22} /><div><h2 className="text-2xl font-bold text-gray-900 dark:text-white">邮箱搜索</h2><p className="text-sm text-gray-600 dark:text-gray-300">搜索功能持续开放，可直接检查邮箱前缀是否已被占用。</p></div></div>
        <form onSubmit={handleSearch} className="flex flex-col gap-4 sm:flex-row"><div className="relative flex-1"><input value={searchPrefix} onChange={(event) => { setSearchPrefix(event.target.value); setSearchStatus('idle'); setSearchResult(null); setSearchMessage(''); }} placeholder="输入你想查询的邮箱前缀" className="w-full rounded-2xl border border-white/40 bg-white/50 px-4 py-4 pr-40 text-gray-900 outline-none focus:ring-2 focus:ring-teal-500 dark:border-white/20 dark:bg-black/50 dark:text-white" /><div className="absolute right-4 top-1/2 -translate-y-1/2 text-sm text-gray-500">@{searchRootDomain}</div></div><button type="submit" disabled={searchStatus === 'checking' || searchPrefix.trim() === ''} className="rounded-2xl bg-gradient-to-r from-teal-500 to-emerald-500 px-8 py-4 font-bold text-white shadow-lg disabled:cursor-not-allowed disabled:opacity-60">{searchStatus === 'checking' ? '查询中...' : '查询'}</button></form>
        {searchStatus !== 'idle' ? <div className="mt-6 rounded-3xl border border-white/20 bg-white/35 p-5 dark:bg-black/25"><div className="flex items-start gap-3">{searchStatus === 'available' ? <CheckCircle2 className="mt-0.5 text-emerald-500" size={20} /> : searchStatus === 'taken' ? <XCircle className="mt-0.5 text-red-500" size={20} /> : searchStatus === 'error' ? <AlertCircle className="mt-0.5 text-amber-500" size={20} /> : <LoaderCircle className="mt-0.5 animate-spin text-teal-500" size={20} />}<div><div className="font-mono text-teal-700 dark:text-teal-300">{searchResult?.address ?? `${searchPrefix.trim()}@${searchRootDomain}`}</div><div className="mt-1 text-sm text-gray-600 dark:text-gray-300">{searchMessage}</div></div></div></div> : null}
      </GlassCard>
      {!authenticated && !sessionLoading ? <GlassCard className="mb-8 border-amber-300/35 bg-amber-100/70 dark:border-amber-500/20 dark:bg-amber-950/30"><div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between"><div><h2 className="text-2xl font-bold text-amber-950 dark:text-amber-100">登录后管理你的邮箱分发</h2><p className="mt-2 text-sm leading-7 text-amber-900 dark:text-amber-200">登录后可直接配置默认邮箱 <span className="font-semibold">{user?.username ?? '<你的用户名>'}@{searchRootDomain}</span> 的转发目标，并申请邮箱泛解析权限。</p></div><button type="button" onClick={onLogin} className="rounded-2xl bg-gradient-to-r from-amber-500 to-orange-500 px-6 py-3 font-semibold text-white shadow-lg">使用 Linux Do 登录</button></div></GlassCard> : null}
      {sessionLoading ? <GlassCard><div className="flex items-center gap-3 text-gray-600 dark:text-gray-300"><LoaderCircle size={20} className="animate-spin" />正在确认你的登录状态与邮箱数据...</div></GlassCard> : null}
      {authenticated ? <div className="grid gap-6 xl:grid-cols-[1.8fr_1fr]"><div className="space-y-6"><GlassCard><div className="mb-5 flex items-center justify-between gap-4"><div><h2 className="text-2xl font-bold text-gray-900 dark:text-white">我的邮箱</h2><p className="mt-1 text-sm text-gray-600 dark:text-gray-300">默认邮箱会自动保留；数据库里已有的附加邮箱也会一并展示。</p></div>{loading ? <LoaderCircle size={20} className="animate-spin text-teal-500" /> : null}</div><div className="overflow-hidden rounded-3xl border border-white/20"><div className="overflow-x-auto"><table className="min-w-full text-sm"><thead className="bg-white/45 text-left text-gray-600 dark:bg-white/5 dark:text-gray-300"><tr><th className="px-5 py-4 font-semibold">邮箱地址</th><th className="px-5 py-4 font-semibold">类型</th><th className="px-5 py-4 font-semibold">转发到</th><th className="px-5 py-4 font-semibold">状态</th><th className="px-5 py-4 font-semibold">说明</th></tr></thead><tbody>{tableRows.map((route) => { const state = routeBadge(route); return <tr key={`${route.kind}-${route.address}`} className="border-t border-white/10 text-gray-700 dark:text-gray-200"><td className="px-5 py-4 font-mono text-teal-700 dark:text-teal-300">{route.address}</td><td className="px-5 py-4">{route.display_name}</td><td className="px-5 py-4">{route.target_email ? <span className="inline-flex items-center gap-2 break-all"><ArrowRight size={15} /> {route.target_email}</span> : <span className="text-gray-500 dark:text-gray-400">尚未设置</span>}</td><td className="px-5 py-4"><span className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${state.className}`}>{state.label}</span></td><td className="px-5 py-4 text-gray-600 dark:text-gray-300"><div>{route.description}</div>{route.updated_at ? <div className="mt-1 text-xs text-gray-500 dark:text-gray-400">最近更新：{formatDate(route.updated_at)}</div> : null}</td></tr>; })}</tbody></table></div></div></GlassCard><GlassCard><div className="mb-5 flex items-center gap-3"><Mail className="text-teal-500" size={22} /><div><h2 className="text-2xl font-bold text-gray-900 dark:text-white">默认邮箱设置</h2><p className="text-sm text-gray-600 dark:text-gray-300">你的默认邮箱始终保留为 <span className="font-mono">{defaultRoute?.address ?? `${user?.username ?? 'username'}@${searchRootDomain}`}</span>。</p></div></div><div className="space-y-4"><input type="email" value={defaultTarget} onChange={(event) => setDefaultTarget(event.target.value)} placeholder="例如：yourname@gmail.com" className="w-full rounded-2xl border border-slate-200 bg-white/70 px-4 py-3 outline-none focus:border-teal-400 focus:ring-2 focus:ring-teal-400/20 dark:border-slate-700 dark:bg-black/35 dark:text-white" /><label className="flex items-center gap-3 rounded-2xl border border-slate-200 bg-white/70 px-4 py-3 text-sm text-slate-700 dark:border-slate-700 dark:bg-black/35 dark:text-slate-200"><input type="checkbox" checked={defaultEnabled} onChange={(event) => setDefaultEnabled(event.target.checked)} disabled={defaultTarget.trim() === ''} />启用默认邮箱转发</label><div className="flex flex-wrap gap-3"><button type="button" onClick={() => void saveDefault(defaultTarget, defaultTarget.trim() !== '' ? defaultEnabled : false)} disabled={savingDefault} className="rounded-2xl bg-gradient-to-r from-teal-500 to-emerald-500 px-5 py-3 font-semibold text-white shadow-lg disabled:cursor-not-allowed disabled:opacity-60">{savingDefault ? '保存中...' : '保存默认邮箱'}</button><button type="button" onClick={() => void saveDefault('', false)} disabled={savingDefault || (!defaultRoute?.configured && defaultTarget.trim() === '')} className="rounded-2xl bg-slate-100 px-5 py-3 font-semibold text-slate-700 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-slate-800 dark:text-slate-100">清空转发</button></div></div></GlassCard></div><GlassCard><div className="mb-5 flex items-center gap-3"><ShieldCheck className="text-amber-500" size={22} /><div><h2 className="text-2xl font-bold text-gray-900 dark:text-white">邮箱泛解析</h2><p className="text-sm text-gray-600 dark:text-gray-300">附加权限功能，不影响默认邮箱分发。</p></div></div><div className="mb-4 flex items-center justify-between gap-3"><span className={`inline-flex rounded-full px-3 py-1 text-sm font-semibold ${permissionBadge(permission?.status ?? 'not_requested').className}`}>{permissionBadge(permission?.status ?? 'not_requested').label}</span><span className="font-mono text-sm text-teal-700 dark:text-teal-300">{catchAllRoute?.address ?? permission?.target ?? 'catch-all@<username>.linuxdo.space'}</span></div>{permission?.eligibility_reasons?.length ? <div className="mb-4 rounded-2xl border border-amber-300/40 bg-amber-100/70 p-4 text-sm text-amber-900 dark:border-amber-500/20 dark:bg-amber-950/30 dark:text-amber-100">{permission.eligibility_reasons.map((item) => <div key={item} className="mb-2 last:mb-0">{item}</div>)}</div> : null}{!permission?.can_manage_route ? <div className="space-y-4"><div className="rounded-2xl border border-white/20 bg-white/35 p-4 text-sm leading-7 text-gray-600 dark:bg-black/25 dark:text-gray-300">提交申请前请阅读承诺书。系统会记录你的申请痕迹，权限通过后才可配置 catch-all 转发。</div><div className="rounded-2xl border border-amber-300/40 bg-amber-50/80 p-4 text-sm leading-7 text-amber-900 dark:border-amber-500/20 dark:bg-amber-950/25 dark:text-amber-100">{permission?.pledge_text ?? '当前暂无法加载承诺书。'}</div><button type="button" onClick={() => void handleApplyCatchAll()} disabled={!permission?.can_apply || applying} className="w-full rounded-2xl bg-gradient-to-r from-amber-500 to-orange-500 px-4 py-3 font-semibold text-white shadow-lg disabled:cursor-not-allowed disabled:opacity-60">{applying ? '提交中...' : permission?.status === 'pending' ? '申请审核中' : '申请邮箱泛解析权限'}</button></div> : <div className="space-y-4"><input type="email" value={catchAllTarget} onChange={(event) => setCatchAllTarget(event.target.value)} placeholder="例如：yourname@gmail.com" className="w-full rounded-2xl border border-slate-200 bg-white/70 px-4 py-3 outline-none focus:border-amber-400 focus:ring-2 focus:ring-amber-400/20 dark:border-slate-700 dark:bg-black/35 dark:text-white" /><label className="flex items-center gap-3 rounded-2xl border border-slate-200 bg-white/70 px-4 py-3 text-sm text-slate-700 dark:border-slate-700 dark:bg-black/35 dark:text-slate-200"><input type="checkbox" checked={catchAllEnabled} onChange={(event) => setCatchAllEnabled(event.target.checked)} />启用邮箱泛解析转发</label><button type="button" onClick={() => void handleSaveCatchAll()} disabled={savingCatchAll} className="w-full rounded-2xl bg-gradient-to-r from-amber-500 to-orange-500 px-4 py-3 font-semibold text-white shadow-lg disabled:cursor-not-allowed disabled:opacity-60">{savingCatchAll ? '保存中...' : '保存邮箱泛解析'}</button></div>}</GlassCard></div> : null}
    </div>
  );
}

function NoticeCard({ tone, message }: { tone: 'error' | 'success'; message: string }) {
  const toneClassName = tone === 'error'
    ? 'mb-6 border-red-300/35 bg-red-100/70 text-red-900 dark:border-red-500/20 dark:bg-red-950/35 dark:text-red-100'
    : 'mb-6 border-emerald-300/35 bg-emerald-100/70 text-emerald-900 dark:border-emerald-500/20 dark:bg-emerald-950/35 dark:text-emerald-100';
  const Icon = tone === 'error' ? AlertCircle : CheckCircle2;
  return <GlassCard className={toneClassName}><div className="flex items-start gap-3"><Icon className="mt-0.5" size={18} /><div>{message}</div></div></GlassCard>;
}

function upsertRoute(routes: UserEmailRoute[], nextRoute: UserEmailRoute): UserEmailRoute[] {
  const remainingRoutes = routes.filter((item) => item.kind !== nextRoute.kind);
  if (nextRoute.kind === 'default') {
    return [nextRoute, ...remainingRoutes];
  }
  const defaultRoute = remainingRoutes.find((item) => item.kind === 'default');
  const others = remainingRoutes.filter((item) => item.kind !== 'default');
  return defaultRoute ? [defaultRoute, ...others, nextRoute] : [...others, nextRoute];
}

function permissionBadge(status: UserPermission['status']) {
  switch (status) {
    case 'approved':
      return { label: '已通过', className: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/25 dark:text-emerald-300' };
    case 'pending':
      return { label: '待审核', className: 'bg-amber-100 text-amber-700 dark:bg-amber-900/25 dark:text-amber-300' };
    case 'rejected':
      return { label: '未通过', className: 'bg-red-100 text-red-700 dark:bg-red-900/25 dark:text-red-300' };
    default:
      return { label: '尚未申请', className: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300' };
  }
}

function routeBadge(route: UserEmailRoute) {
  if (route.kind === 'catch_all' && route.permission_status && route.permission_status !== 'approved') {
    return permissionBadge(route.permission_status);
  }
  if (!route.configured) {
    return { label: '未配置', className: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300' };
  }
  if (!route.enabled) {
    return { label: '已暂停', className: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300' };
  }
  return { label: '已启用', className: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/25 dark:text-emerald-300' };
}

function buildSearchMessage(result: EmailRouteAvailabilityResult, normalizedUsername: string, authenticated: boolean): string {
  if (result.available) {
    if (authenticated && result.normalized_prefix === normalizedUsername) {
      return '这是你默认保留的邮箱地址，直接在下方配置转发即可。';
    }
    return '当前前缀未被占用。搜索功能已恢复，默认邮箱和既有邮箱分发已可正常管理。';
  }
  if (authenticated && result.normalized_prefix === normalizedUsername) {
    return '这是你的默认邮箱地址，搜索结果显示为已占用是正常现象。';
  }
  if (result.reasons.includes('reserved_by_existing_user')) {
    return '该邮箱前缀已经被现有用户的默认邮箱保留。';
  }
  if (result.reasons.includes('existing_email_route')) {
    return '该邮箱地址已经存在转发配置。';
  }
  if (result.reasons.includes('reserved_system_prefix')) {
    return '该邮箱前缀属于系统保留地址，无法使用。';
  }
  return '该邮箱前缀当前不可用。';
}

function readableErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof APIError) {
    return error.message;
  }
  if (error instanceof Error && error.message.trim() !== '') {
    return error.message;
  }
  return fallback;
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value));
}

function normalizePrefix(value: string): string {
  return value.trim().toLowerCase().replace(/[^a-z0-9-]+/g, '-').replace(/^-+|-+$/g, '');
}
