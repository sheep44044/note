import React, { useRef, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { User, Lock, Camera, Loader2, Save } from 'lucide-react';
import api from '@/lib/axios';
import type {UserProfile} from '@/types';

export default function SettingsPage() {
    const queryClient = useQueryClient();
    const fileInputRef = useRef<HTMLInputElement>(null);

    // 1. 获取当前用户信息
    const { data: user, isLoading } = useQuery({
        queryKey: ['users', 'me'],
        queryFn: async () => {
            const res = await api.get<any, { data: UserProfile }>('/users/me');
            return res.data;
        }
    });

    // 状态管理
    const [bio, setBio] = useState('');
    const [avatar, setAvatar] = useState('');
    const [oldPassword, setOldPassword] = useState('');
    const [newPassword, setNewPassword] = useState('');

    // 初始化表单数据
    React.useEffect(() => {
        if (user) {
            setBio(user.bio || '');
            setAvatar(user.avatar || user.avatar_url || '');
        }
    }, [user]);

    // --- 动作 1: 修改资料 ---
    const updateProfileMutation = useMutation({
        mutationFn: async () => {
            await api.put('/users/me', { bio, avatar });
        },
        onSuccess: () => {
            alert('资料更新成功！');
            queryClient.invalidateQueries({ queryKey: ['users', 'me'] });
        }
    });

    // --- 动作 2: 上传头像 ---
    const uploadAvatarMutation = useMutation({
        mutationFn: async (file: File) => {
            const formData = new FormData();
            formData.append('image', file);
            const res = await api.post('/notes/images', formData); // 复用图片上传接口
            return res.data.url;
        },
        onSuccess: (url) => {
            setAvatar(url); // 上传成功后只更新本地预览，需要点击保存才提交给 User
        }
    });

    // --- 动作 3: 修改密码 ---
    const changePasswordMutation = useMutation({
        mutationFn: async () => {
            await api.post('/users/change-password', {
                old_password: oldPassword,
                new_password: newPassword
            });
        },
        onSuccess: () => {
            alert('密码修改成功，请重新登录。');
            localStorage.removeItem('token');
            window.location.href = '/login';
        },
        onError: () => alert('旧密码错误或系统异常')
    });

    if (isLoading) return <div className="flex justify-center py-20"><Loader2 className="animate-spin" /></div>;

    return (
        <div className="max-w-xl mx-auto pb-20">
            <h2 className="text-xl font-bold text-slate-800 mb-6 flex items-center gap-2">
                <User /> 账户设置
            </h2>

            {/* --- Section 1: 基本资料 --- */}
            <div className="bg-white p-6 rounded-2xl shadow-sm border border-slate-100 mb-6">
                <h3 className="font-bold text-slate-700 mb-4 border-b pb-2">基本资料</h3>

                {/* Avatar Upload */}
                <div className="flex items-center gap-6 mb-6">
                    <div className="relative group cursor-pointer" onClick={() => fileInputRef.current?.click()}>
                        <div className="w-20 h-20 rounded-full bg-slate-100 overflow-hidden border-2 border-slate-200">
                            {avatar ? (
                                <img src={avatar} alt="avatar" className="w-full h-full object-cover" />
                            ) : (
                                <div className="w-full h-full flex items-center justify-center text-slate-400 font-bold text-2xl">
                                    {(user?.nickname?.[0] || user?.username?.[0] || 'U').toUpperCase()}
                                </div>
                            )}
                        </div>
                        {/* Overlay */}
                        <div className="absolute inset-0 bg-black/40 rounded-full flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
                            <Camera className="text-white" size={24} />
                        </div>
                        {uploadAvatarMutation.isPending && (
                            <div className="absolute inset-0 bg-white/80 rounded-full flex items-center justify-center">
                                <Loader2 className="animate-spin text-indigo-600" />
                            </div>
                        )}
                    </div>
                    <div className="flex-1">
                        <p className="font-bold text-slate-800">{user?.nickname || user?.username}</p>
                        <p className="text-sm text-slate-500">点击头像可更换图片</p>
                        <input type="file" ref={fileInputRef} className="hidden" accept="image/*" onChange={(e) => e.target.files?.[0] && uploadAvatarMutation.mutate(e.target.files[0])} />
                    </div>
                </div>

                {/* Bio Input */}
                <div className="mb-4">
                    <label className="block text-sm font-medium text-slate-700 mb-1">个人简介 (Bio)</label>
                    <textarea
                        className="w-full px-3 py-2 border border-slate-200 rounded-lg focus:ring-2 focus:ring-indigo-500 outline-none resize-none h-24 text-sm"
                        placeholder="介绍一下你自己..."
                        value={bio}
                        onChange={(e) => setBio(e.target.value)}
                    />
                </div>

                <button
                    onClick={() => updateProfileMutation.mutate()}
                    disabled={updateProfileMutation.isPending}
                    className="bg-indigo-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-indigo-700 flex items-center gap-2 transition-all"
                >
                    {updateProfileMutation.isPending ? <Loader2 className="animate-spin" size={16} /> : <Save size={16} />}
                    保存资料
                </button>
            </div>

            {/* --- Section 2: 安全设置 --- */}
            <div className="bg-white p-6 rounded-2xl shadow-sm border border-slate-100">
                <h3 className="font-bold text-slate-700 mb-4 border-b pb-2 flex items-center gap-2">
                    <Lock size={18} /> 安全设置
                </h3>

                <div className="space-y-4 mb-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">旧密码</label>
                        <input
                            type="password"
                            className="w-full px-3 py-2 border border-slate-200 rounded-lg focus:ring-2 focus:ring-indigo-500 outline-none"
                            value={oldPassword}
                            onChange={(e) => setOldPassword(e.target.value)}
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">新密码</label>
                        <input
                            type="password"
                            className="w-full px-3 py-2 border border-slate-200 rounded-lg focus:ring-2 focus:ring-indigo-500 outline-none"
                            value={newPassword}
                            onChange={(e) => setNewPassword(e.target.value)}
                        />
                    </div>
                </div>

                <button
                    onClick={() => changePasswordMutation.mutate()}
                    disabled={!oldPassword || !newPassword || changePasswordMutation.isPending}
                    className="bg-white border border-slate-300 text-slate-700 px-4 py-2 rounded-lg text-sm font-medium hover:bg-slate-50 transition-all"
                >
                    修改密码
                </button>
            </div>
        </div>
    );
}