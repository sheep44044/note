import { useParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { UserPlus, UserMinus, Users, FileText, Loader2, Settings } from 'lucide-react';
import api from '@/lib/axios';
import { NoteCard } from '@/components/NoteCard';
import type {UserProfile} from '@/types';
import { cn } from '@/lib/utils';
import { Link } from 'react-router-dom';

export default function ProfilePage() {
    const { id } = useParams();
    const queryClient = useQueryClient();

    // 获取当前登录用户的 ID (用于判断是否是自己)
    // 这里简单处理：实际项目中最好从全局 Context 或 localStorage 读取
    const currentUserStr = localStorage.getItem('user');
    const currentUserId = currentUserStr ? JSON.parse(currentUserStr).id : 0;
    const isMe = Number(id) === currentUserId;

    // 1. 获取用户详情 (包含笔记列表)
    const { data, isLoading } = useQuery({
        queryKey: ['user', id],
        queryFn: async () => {
            const res = await api.get<any, { data: UserProfile }>(`/users/${id}`);
            return res.data;
        }
    });

    // 2. 关注/取消关注逻辑
    const followMutation = useMutation({
        mutationFn: async (isFollowing: boolean) => {
            if (isFollowing) {
                await api.delete(`/users/${id}/follow`);
            } else {
                await api.post(`/users/${id}/follow`);
            }
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['user', id] });
        }
    });

    if (isLoading) {
        return <div className="flex justify-center py-20"><Loader2 className="animate-spin text-slate-400" /></div>;
    }

    if (!data) return <div className="text-center py-20">用户不存在</div>;

    return (
        <div className="max-w-2xl mx-auto pb-20">
            {/* --- Profile Header --- */}
            <div className="bg-white rounded-2xl p-6 shadow-sm border border-slate-100 mb-6 relative overflow-hidden">
                <div className="flex flex-col md:flex-row gap-6 items-start md:items-center">

                    {/* Avatar */}
                    <div className="w-24 h-24 rounded-full bg-indigo-100 border-4 border-white shadow-md flex items-center justify-center text-3xl font-bold text-indigo-500 shrink-0 overflow-hidden">
                        {data.avatar ? (
                            <img src={data.avatar} alt={data.nickname} className="w-full h-full object-cover" />
                        ) : (
                            (data.nickname?.[0] || data.username?.[0] || 'U').toUpperCase()
                        )}
                    </div>

                    {/* Info */}
                    <div className="flex-1">
                        <h1 className="text-2xl font-bold text-slate-900 mb-1">
                            {data.nickname || data.username}
                        </h1>
                        <p className="text-slate-500 text-sm mb-4">@{data.username}</p>

                        {/* Bio */}
                        {data.bio && (
                            <p className="text-slate-600 text-sm mb-4 bg-slate-50 p-2 rounded-lg">
                                {data.bio}
                            </p>
                        )}

                        {/* Stats */}
                        <div className="flex gap-6 text-sm">
                            <div className="flex items-center gap-1.5">
                                <FileText size={16} className="text-slate-400" />
                                <span className="font-bold text-slate-800">{data.documents?.length || 0}</span>
                                <span className="text-slate-500">笔记</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <Users size={16} className="text-slate-400" />
                                <span className="font-bold text-slate-800">{data.fan_count || 0}</span>
                                <span className="text-slate-500">粉丝</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <span className="font-bold text-slate-800 ml-5">{data.follow_count || 0}</span>
                                <span className="text-slate-500">关注</span>
                            </div>
                        </div>
                    </div>

                    {/* Actions */}
                    <div className="self-start md:self-center">
                        {isMe ? (
                            <Link
                                to="/settings"
                                className="px-4 py-2 border border-slate-200 rounded-lg text-sm font-medium text-slate-600 hover:bg-slate-50 flex items-center gap-2 transition-colors"
                            >
                                <Settings size={16} /> 编辑资料
                            </Link>
                        ) : (
                            <button
                                onClick={() => followMutation.mutate(!!data.is_following)}
                                disabled={followMutation.isPending}
                                className={cn(
                                    "px-6 py-2 rounded-lg text-sm font-medium flex items-center gap-2 transition-all shadow-sm",
                                    data.is_following
                                        ? "bg-slate-100 text-slate-600 hover:bg-slate-200 border border-slate-200"
                                        : "bg-slate-900 text-white hover:bg-slate-800"
                                )}
                            >
                                {followMutation.isPending ? <Loader2 className="animate-spin" size={16} /> : (
                                    data.is_following ? <><UserMinus size={16}/> 已关注</> : <><UserPlus size={16}/> 关注</>
                                )}
                            </button>
                        )}
                    </div>
                </div>
            </div>

            {/* --- Notes List --- */}
            <h3 className="font-bold text-lg text-slate-800 mb-4 px-1">发布的笔记</h3>
            <div className="space-y-4">
                {data.documents && data.documents.length > 0 ? (
                    data.documents.map((note) => (
                        // 这里的 note 往往没有 User 信息，我们需要手动把当前 data (作者) 补进去，
                        // 否则 NoteCard 会显示不出来头像
                        <NoteCard
                            key={note.id}
                            note={{...note, user: data, UserID: data.id}}
                        />
                    ))
                ) : (
                    <div className="text-center text-slate-400 py-10 bg-slate-50 rounded-xl border border-dashed border-slate-200">
                        这个人很懒，什么都没写。
                    </div>
                )}
            </div>
        </div>
    );
}