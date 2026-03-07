import { useState } from 'react';
import { AnimatePresence, motion } from 'motion/react';
import {
  ArrowLeft,
  CheckCircle,
  Clock,
  Key,
  List,
  Send,
  ShieldPlus,
  Ticket,
  Trash2,
  XCircle,
} from 'lucide-react';
import confetti from 'canvas-confetti';
import { GlassCard } from '../components/GlassCard';
import { GlassSelect } from '../components/GlassSelect';

// PermissionViewMode 控制权限页在“申请主页”和“记录列表”之间切换。
type PermissionViewMode = 'main' | 'list';

// PreviewPermission 表示一条本地已获得权限记录。
interface PreviewPermission {
  id: number;
  type: string;
  target: string;
  acquiredAt: string;
}

// PreviewApplication 表示一条本地权限申请记录。
interface PreviewApplication {
  id: number;
  type: string;
  target: string;
  status: 'approved' | 'pending' | 'rejected';
  appliedAt: string;
}

// permissionOptions 对应设计稿里可申请的权限类型。
const permissionOptions = [
  { value: 'single', label: '某个特定二级域名' },
  { value: 'multiple', label: '某个域名的任意 X 次注册' },
  { value: 'wildcard', label: '某个二级域名及其全部子域名 (泛解析)' },
];

// Permissions 负责承接新 UI 里的“权限申请”页面。
// 当前仍是前端预览，兑换和申请不会触发真实后端动作。
export function Permissions() {
  const [viewMode, setViewMode] = useState<PermissionViewMode>('main');
  const [redeemCode, setRedeemCode] = useState('');
  const [permissionType, setPermissionType] = useState('single');
  const [targetDomain, setTargetDomain] = useState('');
  const [applyReason, setApplyReason] = useState('');
  const [isRedeeming, setIsRedeeming] = useState(false);
  const [isApplying, setIsApplying] = useState(false);
  const [redeemedPermissions, setRedeemedPermissions] = useState<PreviewPermission[]>([
    { id: 1, type: 'single', target: 'api.linuxdo.space', acquiredAt: '2026-03-01' },
    { id: 2, type: 'multiple', target: '5 次注册额度', acquiredAt: '2026-03-02' },
  ]);
  const [applications, setApplications] = useState<PreviewApplication[]>([
    { id: 1, type: 'wildcard', target: '*.dev.linuxdo.space', status: 'approved', appliedAt: '2026-03-05' },
    { id: 2, type: 'single', target: 'test.linuxdo.space', status: 'pending', appliedAt: '2026-03-06' },
    { id: 3, type: 'multiple', target: '10 次注册额度', status: 'rejected', appliedAt: '2026-03-04' },
  ]);

  // triggerCelebration 在本地预览动作完成后给出统一的成功反馈。
  function triggerCelebration(): void {
    confetti({ particleCount: 100, spread: 70, origin: { y: 0.6 }, colors: ['#2dd4bf', '#34d399', '#0ea5e9'] });
  }

  // handleRedeem 模拟兑换码兑换流程，并写入本地记录列表。
  function handleRedeem(event: React.FormEvent<HTMLFormElement>): void {
    event.preventDefault();
    if (redeemCode.trim() === '') {
      return;
    }

    setIsRedeeming(true);
    window.setTimeout(() => {
      setRedeemedPermissions((currentRecords) => [
        {
          id: Date.now(),
          type: 'single',
          target: `预览兑换：${redeemCode.trim().toUpperCase()}`,
          acquiredAt: new Date().toISOString().slice(0, 10),
        },
        ...currentRecords,
      ]);
      setRedeemCode('');
      setIsRedeeming(false);
      triggerCelebration();
    }, 700);
  }

  // handleApply 模拟高级权限申请流程，并跳转到本地记录页。
  function handleApply(event: React.FormEvent<HTMLFormElement>): void {
    event.preventDefault();
    if (targetDomain.trim() === '' || applyReason.trim().length < 30) {
      return;
    }

    setIsApplying(true);
    window.setTimeout(() => {
      setApplications((currentRecords) => [
        {
          id: Date.now(),
          type: permissionType,
          target: targetDomain.trim(),
          status: 'pending',
          appliedAt: new Date().toISOString().slice(0, 10),
        },
        ...currentRecords,
      ]);
      setTargetDomain('');
      setApplyReason('');
      setIsApplying(false);
      triggerCelebration();
      setViewMode('list');
    }, 900);
  }

  // getTypeLabel 把内部权限类型转换成可读文字。
  function getTypeLabel(type: string): string {
    switch (type) {
      case 'single':
        return '特定二级域名';
      case 'multiple':
        return '多次注册额度';
      case 'wildcard':
        return '泛解析';
      default:
        return type;
    }
  }

  // getStatusMeta 返回当前申请状态的图标和颜色信息。
  function getStatusMeta(status: PreviewApplication['status']) {
    if (status === 'approved') {
      return { label: '已通过', icon: <CheckCircle size={14} />, className: 'bg-emerald-100/80 dark:bg-emerald-950/40 text-emerald-700 dark:text-emerald-300' };
    }
    if (status === 'rejected') {
      return { label: '已拒绝', icon: <XCircle size={14} />, className: 'bg-red-100/80 dark:bg-red-950/40 text-red-700 dark:text-red-300' };
    }
    return { label: '审核中', icon: <Clock size={14} />, className: 'bg-amber-100/80 dark:bg-amber-950/40 text-amber-700 dark:text-amber-300' };
  }

  if (viewMode === 'list') {
    return (
      <div className="max-w-6xl mx-auto pt-32 pb-24 px-6">
        <motion.div initial={{ y: 20, opacity: 0 }} animate={{ y: 0, opacity: 1 }} className="mb-8">
          <button type="button" onClick={() => setViewMode('main')} className="flex items-center gap-2 text-teal-600 dark:text-teal-400 hover:text-teal-700 dark:hover:text-teal-300 transition-colors mb-6 font-medium">
            <ArrowLeft size={20} />
            返回申请页
          </button>
          <div className="flex items-center gap-3">
            <div className="p-3 rounded-xl bg-teal-100 dark:bg-teal-900/30 text-teal-600 dark:text-teal-400">
              <List size={28} />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">我的权限记录</h1>
              <p className="text-gray-500 dark:text-gray-400 text-sm">当前为前端预览列表，用于确认新 UI 的分组和信息密度</p>
            </div>
          </div>
        </motion.div>

        <div className="space-y-8">
          <div>
            <h2 className="text-xl font-bold text-gray-900 dark:text-white mb-4">已获得权限</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {redeemedPermissions.map((item) => (
                <GlassCard key={item.id}>
                  <div className="inline-flex items-center gap-2 rounded-full px-3 py-1 text-xs font-semibold bg-teal-100/80 dark:bg-teal-950/40 text-teal-700 dark:text-teal-300">
                    <Ticket size={13} />
                    已兑换
                  </div>
                  <h3 className="mt-4 text-lg font-bold text-gray-900 dark:text-white">{getTypeLabel(item.type)}</h3>
                  <p className="mt-2 text-sm text-gray-600 dark:text-gray-300 break-all">{item.target}</p>
                  <div className="mt-3 text-xs text-gray-500 dark:text-gray-400">获得时间：{item.acquiredAt}</div>
                </GlassCard>
              ))}
            </div>
          </div>

          <div>
            <h2 className="text-xl font-bold text-gray-900 dark:text-white mb-4">申请进度</h2>
            <div className="space-y-4">
              <AnimatePresence>
                {applications.map((application) => {
                  const statusMeta = getStatusMeta(application.status);
                  return (
                    <motion.div key={application.id} initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -10 }}>
                      <GlassCard>
                        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
                          <div>
                            <div className="inline-flex items-center gap-2 rounded-full px-3 py-1 text-xs font-semibold bg-white/45 dark:bg-black/35 text-gray-700 dark:text-gray-300">
                              <ShieldPlus size={13} />
                              {getTypeLabel(application.type)}
                            </div>
                            <h3 className="mt-3 text-lg font-bold text-gray-900 dark:text-white break-all">{application.target}</h3>
                            <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">申请时间：{application.appliedAt}</p>
                          </div>
                          <div className="flex items-center gap-3 justify-between md:justify-end">
                            <div className={`inline-flex items-center gap-2 rounded-full px-3 py-1 text-xs font-semibold ${statusMeta.className}`}>
                              {statusMeta.icon}
                              {statusMeta.label}
                            </div>
                            <button type="button" onClick={() => setApplications((currentRecords) => currentRecords.filter((record) => record.id !== application.id))} className="p-2 rounded-lg hover:bg-white/50 dark:hover:bg-white/10 text-red-600 dark:text-red-400 transition-colors" aria-label={`删除 ${application.target}`}>
                              <Trash2 size={16} />
                            </button>
                          </div>
                        </div>
                      </GlassCard>
                    </motion.div>
                  );
                })}
              </AnimatePresence>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-6xl mx-auto pt-32 pb-24 px-6">
      <motion.div initial={{ y: 20, opacity: 0 }} animate={{ y: 0, opacity: 1 }} className="mb-10 text-center">
        <div className="inline-flex items-center justify-center p-3 mb-4 rounded-full bg-teal-100 dark:bg-teal-900/30 text-teal-600 dark:text-teal-400">
          <ShieldPlus size={32} />
        </div>
        <h1 className="text-3xl md:text-4xl font-extrabold text-gray-900 dark:text-white mb-4">权限申请</h1>
        <p className="text-lg text-gray-700 dark:text-gray-300 max-w-3xl mx-auto">新设计稿页面已接入前端，兑换码和高级权限申请暂时只提供本地预览，不会触发真实审批。</p>
      </motion.div>

      <div className="grid grid-cols-1 lg:grid-cols-5 gap-6 items-start">
        <div className="lg:col-span-2 space-y-6">
          <GlassCard>
            <div className="flex items-start gap-3">
              <div className="mt-0.5 p-2 rounded-2xl bg-amber-100/80 dark:bg-amber-950/30 text-amber-700 dark:text-amber-300">
                <Ticket size={18} />
              </div>
              <div>
                <div className="text-base font-bold text-gray-900 dark:text-white">当前为前端预览页</div>
                <div className="mt-1 text-sm text-gray-600 dark:text-gray-300 leading-relaxed">当前页面用于确认新 UI 下的兑换、申请、记录查看三块结构是否合理。后端接口将在后续版本接入。</div>
              </div>
            </div>
            <button type="button" onClick={() => setViewMode('list')} className="mt-5 w-full flex items-center justify-center gap-2 px-5 py-3 rounded-2xl bg-white/45 dark:bg-black/35 hover:bg-white/60 dark:hover:bg-black/50 text-gray-900 dark:text-white font-medium transition-all">
              <List size={18} />
              查看我的预览记录
            </button>
          </GlassCard>

          <GlassCard>
            <div className="flex items-center gap-3 mb-6">
              <div className="p-2 rounded-xl bg-teal-100 dark:bg-teal-900/30 text-teal-600 dark:text-teal-400">
                <Key size={24} />
              </div>
              <h2 className="text-xl font-bold text-gray-900 dark:text-white">兑换权限码</h2>
            </div>
            <form onSubmit={handleRedeem} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">兑换码</label>
                <input
                  type="text"
                  value={redeemCode}
                  onChange={(event) => setRedeemCode(event.target.value)}
                  placeholder="输入兑换码，例如：LINUXDO-2026"
                  className="w-full px-4 py-3 rounded-xl bg-white/50 dark:bg-black/50 border border-gray-200 dark:border-gray-700 focus:outline-none focus:ring-2 focus:ring-teal-500 text-gray-900 dark:text-white font-mono"
                />
              </div>
              <button type="submit" disabled={redeemCode.trim() === '' || isRedeeming} className="w-full flex items-center justify-center gap-2 px-6 py-3 rounded-xl bg-gradient-to-r from-teal-500 to-emerald-600 hover:from-teal-600 hover:to-emerald-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium shadow-lg transition-all">
                <Key size={18} />
                {isRedeeming ? '兑换中...' : '立即兑换'}
              </button>
            </form>
          </GlassCard>
        </div>

        <div className="lg:col-span-3">
          <GlassCard>
            <div className="flex items-center gap-3 mb-6">
              <div className="p-2 rounded-xl bg-emerald-100 dark:bg-emerald-900/50 text-emerald-600 dark:text-emerald-400">
                <Send size={24} />
              </div>
              <h2 className="text-xl font-bold text-gray-900 dark:text-white">申请高级权限</h2>
            </div>

            <form onSubmit={handleApply} className="space-y-5">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-5">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">申请权限类型</label>
                  <GlassSelect options={permissionOptions} value={permissionType} onChange={setPermissionType} />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">目标域名/前缀</label>
                  <input
                    type="text"
                    value={targetDomain}
                    onChange={(event) => setTargetDomain(event.target.value)}
                    placeholder="例如：api.linuxdo.space"
                    className="w-full px-4 py-3 rounded-xl bg-white/50 dark:bg-black/50 border border-gray-200 dark:border-gray-700 focus:outline-none focus:ring-2 focus:ring-teal-500 text-gray-900 dark:text-white"
                    required
                  />
                </div>
              </div>

              <div>
                <div className="flex justify-between items-end mb-2 gap-3">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">申请原因 (不少于 30 字)</label>
                  <span className={`text-xs font-mono ${applyReason.length >= 30 ? 'text-teal-600 dark:text-teal-400' : 'text-red-500'}`}>{applyReason.length} / 30</span>
                </div>
                <textarea
                  value={applyReason}
                  onChange={(event) => setApplyReason(event.target.value)}
                  placeholder="请描述申请用途、项目背景和预期使用方式，便于后续接入真实审核流程时复用这块 UI。"
                  rows={5}
                  className="w-full px-4 py-3 rounded-xl bg-white/50 dark:bg-black/50 border border-gray-200 dark:border-gray-700 focus:outline-none focus:ring-2 focus:ring-teal-500 text-gray-900 dark:text-white resize-none"
                />
              </div>

              <button type="submit" disabled={applyReason.trim().length < 30 || targetDomain.trim() === '' || isApplying} className="w-full flex items-center justify-center gap-2 px-6 py-3 rounded-xl bg-gradient-to-r from-emerald-500 to-teal-600 hover:from-emerald-600 hover:to-teal-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium shadow-lg transition-all">
                <Send size={18} />
                {isApplying ? '提交中...' : '提交申请'}
              </button>
            </form>
          </GlassCard>
        </div>
      </div>
    </div>
  );
}
