import React, { useState, useRef, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import ReactMarkdown from 'react-markdown';
import { formatDistanceToNow } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { Pin, Trash2, Edit2, Star, Smile } from 'lucide-react';
import type { Note } from '@/types';
import { cn } from '@/lib/utils';
import api from '@/lib/axios';
import { EditNoteModal } from './EditNoteModal';

interface NoteCardProps {
    note: Note;
    onDelete?: () => void;
    onUpdate?: () => void;
}

// 预设的常用 Emoji 列表
const QUICK_EMOJIS = ["👍", "❤️", "😂", "😮", "😢", "🔥", "🎉", "👀"];

export const NoteCard: React.FC<NoteCardProps> = ({ note, onDelete, onUpdate }) => {
    const navigate = useNavigate();
    const [isDeleting, setIsDeleting] = useState(false);
    const [isEditOpen, setIsEditOpen] = useState(false);
    const [showEmojiPicker, setShowEmojiPicker] = useState(false);
    const emojiPickerRef = useRef<HTMLDivElement>(null);

    // --- 收藏 (Star) ---
    const [isFav, setIsFav] = useState(!!note.IsFavorite);
    const [favCount, setFavCount] = useState(note.FavoriteCount || 0);

    // --- Emoji Reactions ---
    const [reactions, setReactions] = useState<Record<string, number>>(note.reaction_counts || {});

    // 🛡️ 用户名显示修复逻辑
    const displayUser = note.user || {
        id: note.UserID,
        username: `User${note.UserID}`, // 兜底显示
        avatar: undefined
    };

    // 强制显示 Username (去除 nickname 逻辑)
    const displayName = `@${displayUser.username}`;
    const avatarLetter = (displayUser.username?.[0] || 'U').toUpperCase();

    // 点击外部关闭 Emoji 选择器
    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (emojiPickerRef.current && !emojiPickerRef.current.contains(event.target as Node)) {
                setShowEmojiPicker(false);
            }
        };
        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    const handleCardClick = () => {
        if (window.getSelection()?.toString()) return;
        navigate(`/notes/${note.id}`);
    };

    const handleDelete = async (e: React.MouseEvent) => {
        e.stopPropagation();
        if (!confirm('确定要删除这条笔记吗？')) return;
        setIsDeleting(true);
        try {
            await api.delete(`/notes/${note.id}`);
            onDelete?.();
        } catch { alert('删除失败'); }
        finally { setIsDeleting(false); }
    };

    const handlePin = async (e: React.MouseEvent) => {
        e.stopPropagation();
        try {
            await api.patch(`/notes/${note.id}/pin`);
            onUpdate?.();
        } catch (error) { console.error(error); }
    };

    const handleFavorite = async (e: React.MouseEvent) => {
        e.stopPropagation();
        const prevFav = isFav;
        const prevCount = favCount;
        setIsFav(!prevFav);
        setFavCount(prevFav ? prevCount - 1 : prevCount + 1);
        try {
            if (prevFav) await api.delete(`/notes/${note.id}/unfavorite`);
            else await api.post(`/notes/${note.id}/favorite`);
        } catch {
            setIsFav(prevFav); setFavCount(prevCount);
        }
    };

    // 处理 Emoji 点击 (对接后端 API)
    const handleReaction = async (emoji: string) => {
        // 乐观更新
        setReactions(prev => {
            const currentCount = prev[emoji] || 0;
            return { ...prev, [emoji]: currentCount + 1 };
        });
        setShowEmojiPicker(false);

        try {
            await api.post(`/notes/${note.id}/reaction`, { emoji });
        } catch (error) {
            console.error("Reaction failed", error);
        }
    };

    return (
        <>
            <div
                onClick={handleCardClick}
                className={cn(
                    "bg-white rounded-2xl p-5 mb-4 shadow-sm border border-slate-100 transition-all duration-200 hover:shadow-md cursor-pointer group relative",
                    note.IsPinned && "border-l-4 border-l-indigo-500"
                )}
            >
                {/* Header */}
                <div className="flex justify-between items-start mb-3">
                    <div className="flex items-center gap-3">
                        <Link to={`/u/${note.UserID}`} onClick={e => e.stopPropagation()}>
                            <div className="w-10 h-10 rounded-full bg-indigo-100 text-indigo-600 flex items-center justify-center font-bold text-sm overflow-hidden border border-white shadow-sm">
                                {displayUser.avatar ? (
                                    <img src={displayUser.avatar} alt="avatar" className="w-full h-full object-cover" />
                                ) : (
                                    avatarLetter
                                )}
                            </div>
                        </Link>
                        <div className="flex flex-col">
                            <Link to={`/u/${note.UserID}`} onClick={e => e.stopPropagation()} className="text-sm font-bold text-slate-700 hover:text-indigo-600">
                                {displayName}
                            </Link>
                            <span className="text-xs text-slate-400">
                {formatDistanceToNow(new Date(note.created_at), { addSuffix: true, locale: zhCN })}
              </span>
                        </div>
                    </div>
                    <div className="flex items-center gap-2 text-slate-400">
                        {note.IsPinned && <div className="bg-indigo-50 text-indigo-600 px-2 py-0.5 rounded text-xs font-medium flex items-center gap-1"><Pin size={12} className="fill-current"/> 置顶</div>}
                        {note.is_private && <span className="text-xs bg-slate-100 px-2 py-0.5 rounded text-slate-500 font-medium">私密</span>}
                    </div>
                </div>

                {/* Content */}
                <div className="prose prose-sm prose-slate max-w-none mb-3 text-slate-700 break-words">
                    <ReactMarkdown components={{ img: ({...props}) => <img {...props} className="rounded-xl border border-slate-100 shadow-sm max-h-[400px] object-cover my-2" /> }}>
                        {note.content}
                    </ReactMarkdown>
                </div>

                {/* 🌟 AI Summary (摘要在上方) */}
                {note.summary && (
                    <div className="bg-gradient-to-r from-indigo-50 to-purple-50 p-3 rounded-xl mb-4 border border-indigo-100/50">
                        <p className="text-xs font-bold text-indigo-600 mb-1 flex items-center gap-1">
                            ✨ AI 智能摘要
                        </p>
                        <p className="text-sm text-slate-700 opacity-90 leading-relaxed">{note.summary}</p>
                    </div>
                )}

                {/* Tags (标签在摘要下方) */}
                {note.Tags && note.Tags.length > 0 && (
                    <div className="flex flex-wrap gap-2 mb-3">
                        {note.Tags.map(tag => (
                            <Link key={tag.id} to={`/?tag_id=${tag.id}`} onClick={e => e.stopPropagation()} className="text-xs bg-slate-100 px-2.5 py-1 rounded-full text-slate-500 hover:bg-indigo-50 hover:text-indigo-600 transition-colors">
                                #{tag.name}
                            </Link>
                        ))}
                    </div>
                )}

                {/* Footer Actions (Emoji 与 操作按钮同一行) */}
                <div className="flex items-center justify-between pt-3 border-t border-slate-50 min-h-[32px]">

                    {/* 左侧：Emoji 互动区 */}
                    <div className="flex flex-wrap items-center gap-2">
                        {/* 已有的 Emoji 胶囊 */}
                        {Object.entries(reactions).map(([emoji, count]) => count > 0 && (
                            <button
                                key={emoji}
                                onClick={(e) => { e.stopPropagation(); handleReaction(emoji); }}
                                className="flex items-center gap-1 px-2 py-1 bg-slate-50 border border-slate-200 rounded-full text-xs font-medium hover:bg-indigo-50 hover:border-indigo-200 transition-colors group/emoji"
                            >
                                <span>{emoji}</span>
                                <span className="text-slate-500 group-hover/emoji:text-indigo-600">{count}</span>
                            </button>
                        ))}

                        {/* 添加 Emoji 按钮 (带弹窗) */}
                        <div className="relative" ref={emojiPickerRef}>
                            <button
                                onClick={(e) => { e.stopPropagation(); setShowEmojiPicker(!showEmojiPicker); }}
                                className="flex items-center justify-center w-7 h-7 bg-slate-50 border border-slate-200 rounded-full text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-colors"
                                title="添加回应"
                            >
                                <Smile size={14} />
                            </button>

                            {/* Emoji 选择器弹窗 */}
                            {showEmojiPicker && (
                                <div
                                    className="absolute bottom-full left-0 mb-2 bg-white border border-slate-200 shadow-xl rounded-xl p-2 z-20 flex gap-1 w-max animate-in fade-in zoom-in-95 duration-200"
                                    onClick={(e) => e.stopPropagation()}
                                >
                                    {QUICK_EMOJIS.map(emoji => (
                                        <button
                                            key={emoji}
                                            onClick={() => handleReaction(emoji)}
                                            className="w-8 h-8 flex items-center justify-center text-lg hover:bg-slate-100 rounded-lg transition-colors"
                                        >
                                            {emoji}
                                        </button>
                                    ))}
                                </div>
                            )}
                        </div>
                    </div>

                    {/* 右侧：操作按钮 (收藏、编辑、置顶、删除) */}
                    <div className="flex items-center gap-1 ml-auto">
                        <button
                            onClick={handleFavorite}
                            className={cn("p-2 rounded-lg transition-colors flex items-center gap-1.5 text-xs font-medium", isFav ? "text-amber-500 bg-amber-50" : "text-slate-400 hover:bg-amber-50 hover:text-amber-500")}
                            title="收藏"
                        >
                            <Star size={16} className={cn(isFav && "fill-current")} />
                            {favCount > 0 && <span>{favCount}</span>}
                        </button>

                        <button onClick={(e) => { e.stopPropagation(); setIsEditOpen(true); }} className="p-2 hover:bg-slate-100 rounded-lg text-slate-400 hover:text-indigo-600" title="编辑">
                            <Edit2 size={16} />
                        </button>

                        <button onClick={handlePin} className={cn("p-2 hover:bg-slate-100 rounded-lg transition-colors", note.IsPinned ? "text-indigo-600 bg-indigo-50" : "text-slate-400 hover:text-indigo-600")} title="置顶">
                            <Pin size={16} className={cn(note.IsPinned && "fill-current")} />
                        </button>

                        <button onClick={handleDelete} disabled={isDeleting} className="p-2 hover:bg-red-50 rounded-lg text-slate-400 hover:text-red-500" title="删除">
                            <Trash2 size={16} />
                        </button>
                    </div>
                </div>
            </div>

            <EditNoteModal note={note} isOpen={isEditOpen} onClose={() => setIsEditOpen(false)} />
        </>
    );
};