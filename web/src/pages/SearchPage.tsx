import { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Search, Sparkles, Loader2, AlertTriangle } from 'lucide-react'; // 👈 引入 AlertTriangle
import api from '@/lib/axios';
import { NoteCard } from '@/components/NoteCard';
import { cn } from '@/lib/utils';
import type {Note} from '@/types';

export default function SearchPage() {
    const [query, setQuery] = useState('');
    const [mode, setMode] = useState<'normal' | 'ai'>('normal');
    const [debouncedQuery, setDebouncedQuery] = useState('');

    // 防抖逻辑：用户停止输入 500ms 后再触发搜索
    useEffect(() => {
        const timer = setTimeout(() => {
            setDebouncedQuery(query);
        }, 500);
        return () => clearTimeout(timer);
    }, [query]);

    // 根据模式选择 API 端点
    const searchApi = mode === 'normal' ? '/notes/search' : '/notes/smartsearch';

    const { data, isLoading, isError, error, refetch } = useQuery({
        queryKey: ['search', mode, debouncedQuery],
        queryFn: async () => {
            if (!debouncedQuery.trim()) return { list: [] };

            const res = await api.get<any, any>(`${searchApi}?q=${debouncedQuery}`);

            // 数据标准化处理（兼容后端两种不同的返回结构）
            // 普通搜索: res.data.notes
            // AI搜索: res.data (直接是数组)
            if (mode === 'normal') {
                return { list: res.data?.notes || [] };
            } else {
                return { list: Array.isArray(res.data) ? res.data : [] };
            }
        },
        enabled: !!debouncedQuery.trim(), // 只有有关键词时才搜索
        retry: 1, // 失败后重试 1 次
    });

    const notes = (data as any)?.list || [];

    return (
        <div className="max-w-2xl mx-auto pb-20">
            <h2 className="text-xl font-bold text-slate-800 mb-6 flex items-center gap-2">
                <Search className="text-slate-800" />
                全文搜索
            </h2>

            {/* 搜索框与切换栏 (Sticky Top) */}
            <div className="bg-white p-4 rounded-2xl shadow-sm border border-slate-200 mb-6 sticky top-4 z-10">
                <div className="relative mb-4">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={20} />
                    <input
                        className="w-full pl-10 pr-4 py-3 bg-slate-50 border-none rounded-xl focus:ring-2 focus:ring-indigo-500 outline-none transition-all"
                        placeholder={mode === 'ai' ? "描述你想找的内容 (例如: 关于Go语言的学习笔记)..." : "搜索关键词..."}
                        value={query}
                        onChange={e => setQuery(e.target.value)}
                        autoFocus
                    />
                </div>

                <div className="flex bg-slate-100 p-1 rounded-lg">
                    <button
                        onClick={() => setMode('normal')}
                        className={cn(
                            "flex-1 flex items-center justify-center gap-2 py-2 rounded-md text-sm font-medium transition-all",
                            mode === 'normal' ? "bg-white text-slate-900 shadow-sm" : "text-slate-500 hover:text-slate-700"
                        )}
                    >
                        <Search size={16} /> 关键词精确搜索
                    </button>
                    <button
                        onClick={() => setMode('ai')}
                        className={cn(
                            "flex-1 flex items-center justify-center gap-2 py-2 rounded-md text-sm font-medium transition-all",
                            mode === 'ai' ? "bg-white text-indigo-600 shadow-sm" : "text-slate-500 hover:text-slate-700"
                        )}
                    >
                        <Sparkles size={16} /> AI 语义搜索
                    </button>
                </div>
            </div>

            {/* 结果展示区域 */}
            <div className="space-y-4">

                {/* 1. 加载状态 */}
                {isLoading && (
                    <div className="flex flex-col items-center justify-center py-10 text-slate-400">
                        <Loader2 className="animate-spin mb-2" size={32} />
                        <span className="text-sm">正在挖掘你的记忆...</span>
                    </div>
                )}

                {/* 2. 错误状态 (使用了 isError) */}
                {isError && (
                    <div className="flex flex-col items-center justify-center py-10 text-rose-500 bg-rose-50 rounded-xl border border-rose-100 p-6 text-center">
                        <AlertTriangle className="mb-3" size={40} />
                        <h3 className="font-bold text-lg mb-1">搜索遇到了一点问题</h3>
                        <p className="text-sm opacity-80 mb-4 max-w-xs mx-auto">
                            {(error as Error)?.message || "无法连接到服务器，请检查网络或后端服务状态。"}
                        </p>
                        <button
                            onClick={() => refetch()}
                            className="px-5 py-2 bg-white text-rose-600 text-sm font-medium rounded-lg border border-rose-200 hover:bg-rose-100 transition-colors shadow-sm"
                        >
                            重试
                        </button>
                    </div>
                )}

                {/* 3. 空状态 (非加载、非错误、有关键词但无结果) */}
                {!isLoading && !isError && debouncedQuery && notes.length === 0 && (
                    <div className="text-center text-slate-400 py-10">
                        <p>没有找到与 "{debouncedQuery}" 相关的笔记。</p>
                        {mode === 'normal' && (
                            <button
                                onClick={() => setMode('ai')}
                                className="text-indigo-500 hover:underline text-sm mt-2"
                            >
                                试试 AI 语义搜索？
                            </button>
                        )}
                    </div>
                )}

                {/* 4. 笔记列表 */}
                {!isLoading && !isError && notes.map((note: Note) => (
                    <NoteCard key={note.id} note={note} />
                ))}
            </div>
        </div>
    );
}