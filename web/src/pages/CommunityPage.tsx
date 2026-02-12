import { useQuery } from '@tanstack/react-query';
import api from '@/lib/axios';
import type { NoteListResponse, Note } from '@/types'; // 👈 引入 Note 类型
import { NoteCard } from '@/components/NoteCard';
import { Loader2 } from 'lucide-react'; // 顺手加个 Loading 图标

export default function CommunityPage() {
    const { data, isLoading } = useQuery({
        queryKey: ['community-notes'],
        queryFn: () => api.get<any, NoteListResponse>('/notes/community'),
    });

    // 👇 核心修复：数据标准化 (Data Normalization)
    // 我们手动检查 data 的结构，把它统一变成 Note[] 数组
    const responseData = data?.data;
    let notes: Note[] = [];

    if (responseData) {
        if (Array.isArray(responseData)) {
            // 情况 1: 后端直接返回数组 [Note, Note]
            notes = responseData;
        } else if (responseData.notes) {
            // 情况 2: 后端返回 { notes: [...] } (根据你的API文档，社区接口应该是这个)
            notes = responseData.notes;
        } else if (responseData.list) {
            // 情况 3: 后端返回 { list: [...] } (为了兼容性)
            notes = responseData.list;
        }
    }

    return (
        <div className="max-w-2xl mx-auto pb-20">
            <h2 className="text-xl font-bold text-slate-800 mb-6">探索广场</h2>
            <div className="space-y-4">
                {isLoading ? (
                    <div className="flex justify-center py-10 text-slate-400">
                        <Loader2 className="animate-spin" />
                    </div>
                ) : (
                    // 👇 直接遍历处理好的 notes 数组
                    notes.map((note) => (
                        <NoteCard key={note.id} note={note} />
                    ))
                )}

                {/* 空状态处理 */}
                {!isLoading && notes.length === 0 && (
                    <div className="text-center text-slate-400 py-10">
                        暂时没有公开的笔记。
                    </div>
                )}
            </div>
        </div>
    );
}