import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Tag as TagIcon, Loader2 } from 'lucide-react';
import api from '@/lib/axios';
import type {Tag} from '@/types';
import { cn } from '@/lib/utils';

interface TagInputProps {
    selectedTagIds: number[];
    onChange: (ids: number[]) => void;
}

export const TagInput: React.FC<TagInputProps> = ({ selectedTagIds, onChange }) => {
    const [isCreating, setIsCreating] = useState(false);
    const [newTagName, setNewTagName] = useState('');
    const queryClient = useQueryClient();

    // 1. 获取所有标签
    const { data: tags = [] } = useQuery({
        queryKey: ['tags'],
        queryFn: async () => {
            const res = await api.get<any, { data: Tag[] }>('/tags');
            // 兼容后端可能返回 { data: [] } 或 直接 []
            return Array.isArray(res.data) ? res.data : [];
        }
    });

    // 2. 创建标签
    const createTagMutation = useMutation({
        mutationFn: async (name: string) => {
            const res = await api.post('/tags', { name, color: 'blue' }); // 默认颜色
            return res.data;
        },
        onSuccess: (newTag: Tag) => {
            queryClient.invalidateQueries({ queryKey: ['tags'] });
            setNewTagName('');
            setIsCreating(false);
            // 创建后自动选中
            onChange([...selectedTagIds, newTag.id]);
        }
    });

    const toggleTag = (id: number) => {
        if (selectedTagIds.includes(id)) {
            onChange(selectedTagIds.filter(tid => tid !== id));
        } else {
            onChange([...selectedTagIds, id]);
        }
    };

    const handleCreate = (e: React.FormEvent) => {
        e.preventDefault();
        if (newTagName.trim()) {
            createTagMutation.mutate(newTagName.trim());
        }
    };

    return (
        <div className="flex flex-wrap items-center gap-2 mt-3 pt-3 border-t border-slate-100">
            <div className="flex items-center text-xs text-slate-400 mr-2">
                <TagIcon size={14} className="mr-1" /> 标签:
            </div>

            {/* 现有标签列表 */}
            {tags.map(tag => (
                <button
                    key={tag.id}
                    onClick={() => toggleTag(tag.id)}
                    className={cn(
                        "text-xs px-2 py-1 rounded-full border transition-all",
                        selectedTagIds.includes(tag.id)
                            ? "bg-indigo-100 border-indigo-200 text-indigo-700"
                            : "bg-white border-slate-200 text-slate-600 hover:border-indigo-200"
                    )}
                >
                    #{tag.name}
                </button>
            ))}

            {/* 创建新标签按钮/输入框 */}
            {isCreating ? (
                <form onSubmit={handleCreate} className="flex items-center gap-1">
                    <input
                        autoFocus
                        type="text"
                        className="text-xs px-2 py-1 rounded border border-indigo-300 outline-none w-20"
                        placeholder="新标签名"
                        value={newTagName}
                        onChange={e => setNewTagName(e.target.value)}
                        onBlur={() => !newTagName && setIsCreating(false)} // 失去焦点且无内容时关闭
                    />
                    <button
                        type="submit"
                        disabled={createTagMutation.isPending}
                        className="text-indigo-600 hover:bg-indigo-50 rounded p-0.5"
                    >
                        {createTagMutation.isPending ? <Loader2 size={12} className="animate-spin"/> : <Plus size={14} />}
                    </button>
                </form>
            ) : (
                <button
                    onClick={() => setIsCreating(true)}
                    className="text-xs px-2 py-1 rounded-full border border-dashed border-slate-300 text-slate-400 hover:text-indigo-600 hover:border-indigo-300 flex items-center gap-1"
                >
                    <Plus size={12} /> 新建
                </button>
            )}
        </div>
    );
};