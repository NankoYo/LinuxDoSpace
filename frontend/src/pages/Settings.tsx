import { useState } from 'react';
import { motion, AnimatePresence } from 'motion/react';
import { GlassCard } from '../components/GlassCard';
import { Plus, Trash2, Edit2, X } from 'lucide-react';
import confetti from 'canvas-confetti';

interface Record {
  id: number;
  type: string;
  name: string;
  content: string;
  proxied: boolean;
}

export function Settings() {
  const [records, setRecords] = useState<Record[]>([
    { id: 1, type: 'A', name: '@', content: '192.168.1.1', proxied: true },
    { id: 2, type: 'CNAME', name: 'www', content: 'cname.vercel-dns.com', proxied: false },
  ]);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<Record | null>(null);
  const [formData, setFormData] = useState({ type: 'A', name: '', content: '', proxied: false });

  const triggerFireworks = () => {
    const duration = 3 * 1000;
    const end = Date.now() + duration;

    const frame = () => {
      confetti({
        particleCount: 5,
        angle: 60,
        spread: 55,
        origin: { x: 0 },
        colors: ['#2dd4bf', '#34d399', '#0ea5e9']
      });
      confetti({
        particleCount: 5,
        angle: 120,
        spread: 55,
        origin: { x: 1 },
        colors: ['#2dd4bf', '#34d399', '#0ea5e9']
      });

      if (Date.now() < end) {
        requestAnimationFrame(frame);
      }
    };
    frame();
  };

  const handleSave = () => {
    if (!formData.name || !formData.content) return;
    
    if (editingRecord) {
      setRecords(records.map(r => r.id === editingRecord.id ? { ...formData, id: r.id } : r));
    } else {
      setRecords([...records, { ...formData, id: Date.now() }]);
    }
    
    setIsModalOpen(false);
    setEditingRecord(null);
    setFormData({ type: 'A', name: '', content: '', proxied: false });
    triggerFireworks();
  };

  const handleDelete = (id: number) => {
    setRecords(records.filter(r => r.id !== id));
  };

  const openModal = (record?: Record) => {
    if (record) {
      setEditingRecord(record);
      setFormData(record);
    } else {
      setEditingRecord(null);
      setFormData({ type: 'A', name: '', content: '', proxied: false });
    }
    setIsModalOpen(true);
  };

  return (
    <div className="max-w-5xl mx-auto pt-32 pb-24 px-6 relative">
      <motion.div
        initial={{ y: 20, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        className="mb-8 flex flex-col sm:flex-row sm:items-center justify-between gap-4"
      >
        <div>
          <h1 className="text-3xl font-extrabold text-gray-900 dark:text-white">解析配置中心</h1>
          <p className="text-gray-700 dark:text-gray-300 mt-2">管理你的 linuxdo.space 域名记录</p>
        </div>
        <button 
          onClick={() => openModal()}
          className="flex items-center justify-center gap-2 px-6 py-3 rounded-xl bg-gradient-to-r from-teal-500 to-emerald-600 hover:from-teal-600 hover:to-emerald-700 text-white font-medium shadow-lg transition-all transform hover:scale-105"
        >
          <Plus size={18} />
          添加记录
        </button>
      </motion.div>

      <GlassCard className="overflow-hidden p-0">
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-white/20 dark:border-white/10 bg-white/20 dark:bg-black/20">
                <th className="p-4 font-semibold text-gray-900 dark:text-white">类型</th>
                <th className="p-4 font-semibold text-gray-900 dark:text-white">名称</th>
                <th className="p-4 font-semibold text-gray-900 dark:text-white">内容</th>
                <th className="p-4 font-semibold text-gray-900 dark:text-white">代理状态</th>
                <th className="p-4 font-semibold text-gray-900 dark:text-white text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <AnimatePresence>
                {records.map((record) => (
                  <motion.tr
                    key={record.id}
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: 'auto' }}
                    exit={{ opacity: 0, x: -50, backgroundColor: 'rgba(239, 68, 68, 0.2)' }}
                    transition={{ duration: 0.3 }}
                    className="border-b border-white/10 dark:border-white/5 hover:bg-white/30 dark:hover:bg-white/5 transition-colors"
                  >
                    <td className="p-4">
                      <span className="px-2 py-1 rounded-md bg-teal-100 dark:bg-teal-900/50 text-teal-700 dark:text-teal-300 text-sm font-bold">
                        {record.type}
                      </span>
                    </td>
                    <td className="p-4 font-medium text-gray-800 dark:text-gray-200">{record.name}</td>
                    <td className="p-4 text-gray-600 dark:text-gray-400 font-mono text-sm">{record.content}</td>
                    <td className="p-4">
                      <div className="flex items-center gap-2">
                        <div
                          className={`w-3 h-3 rounded-full ${
                            record.proxied ? 'bg-orange-500 shadow-[0_0_8px_#f97316]' : 'bg-gray-400'
                          }`}
                        />
                        <span className="text-sm text-gray-700 dark:text-gray-300">
                          {record.proxied ? '已代理' : '仅 DNS'}
                        </span>
                      </div>
                    </td>
                    <td className="p-4 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <button 
                          onClick={() => openModal(record)}
                          className="p-2 rounded-lg hover:bg-white/50 dark:hover:bg-white/10 text-blue-600 dark:text-blue-400 transition-colors"
                        >
                          <Edit2 size={16} />
                        </button>
                        <button 
                          onClick={() => handleDelete(record.id)}
                          className="p-2 rounded-lg hover:bg-white/50 dark:hover:bg-white/10 text-red-600 dark:text-red-400 transition-colors"
                        >
                          <Trash2 size={16} />
                        </button>
                      </div>
                    </td>
                  </motion.tr>
                ))}
              </AnimatePresence>
            </tbody>
          </table>
        </div>
        {records.length === 0 && (
          <div className="p-12 text-center text-gray-500 dark:text-gray-400">
            暂无解析记录，点击上方按钮添加。
          </div>
        )}
      </GlassCard>

      <AnimatePresence>
        {isModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center px-4">
            <motion.div 
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="absolute inset-0 bg-black/40 backdrop-blur-sm"
              onClick={() => setIsModalOpen(false)}
            />
            <motion.div
              initial={{ scale: 0.9, opacity: 0, y: 20 }}
              animate={{ scale: 1, opacity: 1, y: 0 }}
              exit={{ scale: 0.9, opacity: 0, y: 20 }}
              className="relative w-full max-w-md bg-white/80 dark:bg-gray-900/80 backdrop-blur-xl border border-white/20 dark:border-white/10 rounded-3xl p-6 shadow-2xl"
            >
              <button 
                onClick={() => setIsModalOpen(false)}
                className="absolute top-4 right-4 p-2 text-gray-500 hover:text-gray-900 dark:hover:text-white transition-colors"
              >
                <X size={20} />
              </button>
              
              <h2 className="text-2xl font-bold mb-6 text-gray-900 dark:text-white">
                {editingRecord ? '修改记录' : '添加记录'}
              </h2>
              
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">记录类型</label>
                  <select 
                    value={formData.type}
                    onChange={e => setFormData({...formData, type: e.target.value})}
                    className="w-full px-4 py-2 rounded-xl bg-white/50 dark:bg-black/50 border border-gray-200 dark:border-gray-700 focus:outline-none focus:ring-2 focus:ring-teal-500 text-gray-900 dark:text-white"
                  >
                    <option value="A">A</option>
                    <option value="CNAME">CNAME</option>
                    <option value="TXT">TXT</option>
                    <option value="MX">MX</option>
                  </select>
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">名称</label>
                  <div className="relative">
                    <input 
                      type="text"
                      value={formData.name}
                      onChange={e => setFormData({...formData, name: e.target.value})}
                      placeholder="@ 或 www"
                      className="w-full px-4 py-2 rounded-xl bg-white/50 dark:bg-black/50 border border-gray-200 dark:border-gray-700 focus:outline-none focus:ring-2 focus:ring-teal-500 text-gray-900 dark:text-white"
                    />
                  </div>
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">内容</label>
                  <input 
                    type="text"
                    value={formData.content}
                    onChange={e => setFormData({...formData, content: e.target.value})}
                    placeholder="IPv4 地址或域名"
                    className="w-full px-4 py-2 rounded-xl bg-white/50 dark:bg-black/50 border border-gray-200 dark:border-gray-700 focus:outline-none focus:ring-2 focus:ring-teal-500 text-gray-900 dark:text-white"
                  />
                </div>
                
                <div className="flex items-center gap-3 pt-2">
                  <button 
                    onClick={() => setFormData({...formData, proxied: !formData.proxied})}
                    className={`relative w-12 h-6 rounded-full transition-colors ${formData.proxied ? 'bg-orange-500' : 'bg-gray-300 dark:bg-gray-600'}`}
                  >
                    <div className={`absolute top-1 left-1 w-4 h-4 rounded-full bg-white transition-transform ${formData.proxied ? 'translate-x-6' : 'translate-x-0'}`} />
                  </button>
                  <span className="text-sm text-gray-700 dark:text-gray-300">
                    代理状态 (Cloudflare)
                  </span>
                </div>
              </div>
              
              <div className="mt-8 flex gap-3">
                <button 
                  onClick={() => setIsModalOpen(false)}
                  className="flex-1 px-4 py-2 rounded-xl bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-900 dark:text-white font-medium transition-colors"
                >
                  取消
                </button>
                <button 
                  onClick={handleSave}
                  className="flex-1 px-4 py-2 rounded-xl bg-gradient-to-r from-teal-500 to-emerald-600 hover:from-teal-600 hover:to-emerald-700 text-white font-medium shadow-lg transition-all"
                >
                  保存
                </button>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>
    </div>
  );
}
