import React from 'react';
import { Loader2, X } from 'lucide-react';
import api from '@/lib/axios';
import type {Note} from '@/types';
import { useMutation, useQueryClient } from '@tanstack/react-query';

interface EditNoteModalProps {
    note: Note;
    isOpen: boolean;
    onClose: () => void;
}

export const EditNoteModal: React.FC<EditNoteModalProps> = ({ note, isOpen, onClose }) => {
    const [content, setContent] = React.useState(note.content);
    const [title, setTitle] = React.useState(note.title);
    const [isPrivate, setIsPrivate] = React.useState(note.is_private);
    const queryClient = useQueryClient();

    // 每次打开模态框时重置内容
    React.useEffect(() => {
        if (isOpen) {
            setContent(note.content);
            setTitle(note.title);
            setIsPrivate(note.is_private);
        }
    }, [isOpen, note]);

    const updateMutation = useMutation({
        mutationFn: async () => {
            // API: PUT /notes/:id
            // Body: { title, content, is_private, tag_ids }
            await api.put(`/notes/${note.id}`, {
                title,
                content,
                is_private: isPrivate,
                tag_ids: note.Tags?.map(t => t.id) || [] // 暂时保持原有标签
            });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['notes'] });
            onClose();
        },
        onError: () => {
            alert('更新失败');
        }
    });

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
            <div className="bg-white rounded-2xl w-full max-w-lg shadow-2xl flex flex-col max-h-[90vh]">

                {/* Header */}
                <div className="flex justify-between items-center p-4 border-b">
                    <h3 className="font-bold text-lg text-slate-800">编辑笔记</h3>
                    <button onClick={onClose} className="p-1 hover:bg-slate-100 rounded-full">
                        <X size={20} className="text-slate-500" />
                    </button>
                </div>

                {/* Body */}
                <div className="p-4 flex-1 overflow-y-auto space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">标题</label>
                        <input
                            value={title}
                            onChange={e => setTitle(e.target.value)}
                            className="w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-indigo-500 outline-none"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">内容 (Markdown)</label>
                        <textarea
                            value={content}
                            onChange={e => setContent(e.target.value)}
                            className="w-full h-40 px-3 py-2 border rounded-lg focus:ring-2 focus:ring-indigo-500 outline-none resize-none font-mono text-sm"
                        />
                    </div>

                    <div className="flex items-center gap-2">
                        <input
                            type="checkbox"
                            id="private-check"
                            checked={isPrivate}
                            onChange={e => setIsPrivate(e.target.checked)}
                            className="w-4 h-4 text-indigo-600 rounded focus:ring-indigo-500"
                        />
                        <label htmlFor="private-check" className="text-sm text-slate-700">设为私密笔记</label>
                    </div>
                </div>

                {/* Footer */}
                <div className="p-4 border-t bg-slate-50 rounded-b-2xl flex justify-end gap-3">
                    <button
                        onClick={onClose}
                        className="px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-200 rounded-lg"
                    >
                        取消
                    </button>
                    <button
                        onClick={() => updateMutation.mutate()}
                        disabled={updateMutation.isPending}
                        className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 rounded-lg flex items-center gap-2"
                    >
                        {updateMutation.isPending && <Loader2 className="animate-spin" size={16} />}
                        保存修改
                    </button>
                </div>
            </div>
        </div>
    );
};