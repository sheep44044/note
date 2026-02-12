import { useQuery } from '@tanstack/react-query';
import { Loader2, UserCheck } from 'lucide-react';
import api from '@/lib/axios';
import { NoteCard } from '@/components/NoteCard';
import type {Note} from '@/types';

export default function FollowingPage() {
    const { data, isLoading } = useQuery({
        queryKey: ['following-notes'],
        queryFn: async () => {
            // 假设后端返回的是 { data: [...] } 或 { data: { list: [] } }
            const res = await api.get<any, any>('/notes/follow');
            return Array.isArray(res.data) ? res.data : (res.data?.list || []);
        },
    });

    const notes = data || [];

    return (
        <div className="max-w-2xl mx-auto pb-20">
            <div className="flex items-center gap-3 mb-6">
                <div className="p-2 bg-indigo-100 text-indigo-600 rounded-lg">
                    <UserCheck size={24} />
                </div>
                <div>
                    <h2 className="text-xl font-bold text-slate-800">关注动态</h2>
                    <p className="text-xs text-slate-400">你关注的人最近发布的内容</p>
                </div>
            </div>

            <div className="space-y-4">
                {isLoading ? (
                    <div className="flex justify-center py-10 text-slate-400">
                        <Loader2 className="animate-spin" />
                    </div>
                ) : notes.length > 0 ? (
                    notes.map((note: Note) => (
                        <NoteCard key={note.id} note={note} />
                    ))
                ) : (
                    <div className="text-center py-16">
                        <div className="inline-block p-4 bg-slate-100 rounded-full mb-4 text-slate-400">
                            <UserCheck size={32} />
                        </div>
                        <p className="text-slate-500 mb-2">暂时没有动态</p>
                        <p className="text-sm text-slate-400">去“探索广场”关注一些有趣的灵魂吧！</p>
                    </div>
                )}
            </div>
        </div>
    );
}