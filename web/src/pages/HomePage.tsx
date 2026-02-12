import { useRef, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom'; // 👈 引入 hook 读取 URL
import { Send, Image as ImageIcon, Lock, Unlock, Loader2, Sparkles, X } from 'lucide-react';
import api from '@/lib/axios';
import { NoteCard } from '@/components/NoteCard';
import { TagInput } from '@/components/TagInput'; // 👈 引入新组件
import { cn } from '@/lib/utils';

export default function HomePage() {
    const queryClient = useQueryClient();
    const [searchParams, setSearchParams] = useSearchParams();
    const currentTagId = searchParams.get('tag_id'); // 获取筛选标签

    // 发布状态
    const [content, setContent] = useState('');
    const [isPrivate, setIsPrivate] = useState(false);
    const [selectedTagIds, setSelectedTagIds] = useState<number[]>([]);
    // 👇 AI 开关状态
    const [aiOptions, setAiOptions] = useState({ genTitle: true, genSummary: true });

    const fileInputRef = useRef<HTMLInputElement>(null);

    // 1. 获取笔记列表 (支持 ?tag_id=x 筛选)
    const { data, isLoading } = useQuery({
        queryKey: ['notes', currentTagId], // tagId 变化时重新请求
        queryFn: async () => {
            const url = currentTagId ? `/notes?tag_id=${currentTagId}` : '/notes';
            const res = await api.get(url);
            return res.data;
        },
    });

    // 2. 创建笔记 (包含 AI 参数和标签)
    const createMutation = useMutation({
        mutationFn: (newNote: any) => {
            // 动态构建 query string
            const params = new URLSearchParams();
            if (aiOptions.genTitle) params.append('gen_title', 'true');
            if (aiOptions.genSummary) params.append('gen_summary', 'true');

            return api.post(`/notes?${params.toString()}`, newNote);
        },
        onSuccess: () => {
            setContent('');
            setSelectedTagIds([]); // 清空标签
            queryClient.invalidateQueries({ queryKey: ['notes'] });
        },
    });

    // 图片上传逻辑 (保持不变)
    const uploadMutation = useMutation({
        mutationFn: async (file: File) => {
            const formData = new FormData();
            formData.append('image', file);
            const res = await api.post('/notes/images', formData);
            return res.data;
        },
        onSuccess: (data: any) => {
            const markdownImage = `\n![image](${data.url})\n`;
            setContent((prev) => prev + markdownImage);
        },
    });

    const handlePost = () => {
        if (!content.trim()) return;
        createMutation.mutate({
            title: content.slice(0, 20), // 后端如果开启 gen_title 会覆盖这个
            content: content,
            is_private: isPrivate,
            tag_ids: selectedTagIds // 👈 传选中的标签
        });
    };

    // 数据兼容处理
    const notesList = Array.isArray(data) ? data : (data as any)?.list || (data as any)?.data || [];

    return (
        <div className="max-w-2xl mx-auto pb-20">

            {/* 筛选状态提示 (如果有) */}
            {currentTagId && (
                <div className="mb-4 flex items-center justify-between bg-indigo-50 text-indigo-700 px-4 py-3 rounded-xl border border-indigo-100">
          <span className="text-sm font-medium flex items-center gap-2">
            <Sparkles size={16} /> 正在筛选标签 ID: {currentTagId} 的笔记
          </span>
                    <button
                        onClick={() => setSearchParams({})}
                        className="p-1 hover:bg-indigo-100 rounded-full"
                    >
                        <X size={16} />
                    </button>
                </div>
            )}

            {/* Input Area */}
            <div className="bg-white rounded-2xl shadow-sm border border-slate-200 p-4 mb-8 transition-all focus-within:shadow-md focus-within:border-indigo-200">
        <textarea
            className="w-full resize-none outline-none text-slate-700 placeholder:text-slate-400 min-h-[100px]"
            placeholder="现在的想法是..."
            value={content}
            onChange={(e) => setContent(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && (e.metaKey || e.ctrlKey) && handlePost()}
        />

                {/* 图片上传 Input */}
                <input
                    type="file"
                    ref={fileInputRef}
                    className="hidden"
                    accept="image/*"
                    onChange={(e) => e.target.files?.[0] && uploadMutation.mutate(e.target.files[0])}
                />

                {/* 👇 标签选择器 */}
                <TagInput selectedTagIds={selectedTagIds} onChange={setSelectedTagIds} />

                {/* 底部工具栏 */}
                <div className="flex flex-wrap justify-between items-center mt-3 pt-3 border-t border-slate-50 gap-3">
                    <div className="flex items-center gap-2">
                        {/* 图片按钮 */}
                        <button
                            className="p-2 text-slate-400 hover:bg-slate-100 rounded-full transition-colors disabled:opacity-50"
                            onClick={() => fileInputRef.current?.click()}
                            disabled={uploadMutation.isPending}
                            title="上传图片"
                        >
                            {uploadMutation.isPending ? <Loader2 className="animate-spin" size={18} /> : <ImageIcon size={18} />}
                        </button>

                        {/* 私密按钮 */}
                        <button
                            className={cn("p-2 rounded-full transition-colors flex items-center gap-1", isPrivate ? 'text-indigo-600 bg-indigo-50' : 'text-slate-400 hover:bg-slate-100')}
                            onClick={() => setIsPrivate(!isPrivate)}
                        >
                            {isPrivate ? <Lock size={18} /> : <Unlock size={18} />}
                        </button>

                        <div className="w-px h-4 bg-slate-200 mx-1"></div>

                        {/* 👇 AI 开关 */}
                        <label className="flex items-center gap-1.5 cursor-pointer select-none group">
                            <input
                                type="checkbox"
                                checked={aiOptions.genTitle}
                                onChange={e => setAiOptions({...aiOptions, genTitle: e.target.checked})}
                                className="w-3.5 h-3.5 accent-indigo-600 rounded"
                            />
                            <span className="text-xs text-slate-500 group-hover:text-indigo-600 transition-colors">AI标题</span>
                        </label>
                        <label className="flex items-center gap-1.5 cursor-pointer select-none group">
                            <input
                                type="checkbox"
                                checked={aiOptions.genSummary}
                                onChange={e => setAiOptions({...aiOptions, genSummary: e.target.checked})}
                                className="w-3.5 h-3.5 accent-indigo-600 rounded"
                            />
                            <span className="text-xs text-slate-500 group-hover:text-indigo-600 transition-colors">AI摘要</span>
                        </label>
                    </div>

                    <button
                        onClick={handlePost}
                        disabled={!content.trim() || createMutation.isPending}
                        className="bg-slate-900 text-white px-5 py-2 rounded-full text-sm font-medium hover:bg-slate-800 disabled:opacity-50 flex items-center gap-2 transition-all shadow-sm hover:shadow-md"
                    >
                        {createMutation.isPending ? <Loader2 className="animate-spin" size={16} /> : <Send size={16} />}
                        <span>发布</span>
                    </button>
                </div>
            </div>

            {/* Note List */}
            <div className="space-y-4">
                {isLoading ? (
                    <div className="flex justify-center py-10"><Loader2 className="animate-spin text-slate-400" /></div>
                ) : notesList.map((note: any) => (
                    <NoteCard key={note.id} note={note} />
                ))}
                {!isLoading && notesList.length === 0 && (
                    <div className="text-center text-slate-400 py-10">没有找到相关笔记。</div>
                )}
            </div>
        </div>
    );
}