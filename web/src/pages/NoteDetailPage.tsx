import { useParams, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, Loader2 } from 'lucide-react';
import api from '@/lib/axios';
import { NoteCard } from '@/components/NoteCard';
import type {Note} from '@/types';

export default function NoteDetailPage() {
    const { id } = useParams();
    const navigate = useNavigate();

    const { data: note, isLoading, isError } = useQuery({
        queryKey: ['note', id],
        queryFn: async () => {
            const res = await api.get<any, { data: Note }>(`/notes/${id}`);
            return res.data;
        }
    });

    if (isLoading) return <div className="flex justify-center py-20"><Loader2 className="animate-spin text-slate-400" /></div>;
    if (isError || !note) return <div className="text-center py-20 text-slate-400">笔记不存在或已被删除</div>;

    return (
        <div className="max-w-3xl mx-auto pb-20">
            {/* 顶部导航栏 */}
            <button
                onClick={() => navigate(-1)}
                className="flex items-center gap-2 text-slate-500 hover:text-indigo-600 mb-6 transition-colors font-medium"
            >
                <ArrowLeft size={20} /> 返回列表
            </button>

            {/* 详情卡片 (复用 NoteCard，但可以给它传个参数让它展开全部内容，不过目前 NoteCard 已经是全展开模式了，直接用即可) */}
            <NoteCard note={note} />

            {/* 评论区占位 (API 暂时没写评论接口，这里留个 UI) */}
            <div className="mt-8 pt-8 border-t border-slate-200">
                <h3 className="font-bold text-slate-700 mb-4">评论互动</h3>
                <div className="bg-slate-50 rounded-xl p-8 text-center text-slate-400 border border-dashed border-slate-200">
                    评论功能正在开发中...
                </div>
            </div>
        </div>
    );
}