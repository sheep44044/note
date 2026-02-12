import React from 'react';
import { useNavigate } from 'react-router-dom';
import api from '@/lib/axios';
import type{ AuthResponse } from '@/types';

export default function LoginPage() {
    const [isLogin, setIsLogin] = React.useState(true);
    const [username, setUsername] = React.useState('');
    const [password, setPassword] = React.useState('');
    const [loading, setLoading] = React.useState(false);
    const navigate = useNavigate();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setLoading(true);
        try {
            const endpoint = isLogin ? '/login' : '/register';
            const res = await api.post<any, AuthResponse>(endpoint, { username, password });

            // 注意：根据 axios interceptor，这里拿到的是 res.data (根据你的契约结构是 code, data)
            // 如果后端严格返回 { code: 200, data: { ... } }，Axios拦截器返回了 response.data

            if (res.code === 200 || (res as any).token) { // 兼容性处理
                // 登录成功
                const token = res.data?.token;
                if (token) {
                    localStorage.setItem('token', token);
                    navigate('/');
                } else if (!isLogin) {
                    alert('注册成功，请登录');
                    setIsLogin(true);
                }
            } else {
                alert('操作失败');
            }
        } catch (err) {
            console.error(err);
            alert('请求错误，请检查控制台');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen flex items-center justify-center bg-[#f3f4f6] p-4">
            <div className="bg-white w-full max-w-md p-8 rounded-2xl shadow-xl border border-slate-100">
                <div className="text-center mb-8">
                    <h1 className="text-3xl font-black text-slate-900 mb-2">MyNotes.</h1>
                    <p className="text-slate-500">记录想法，连接未来</p>
                </div>

                <div className="flex mb-6 bg-slate-100 p-1 rounded-lg">
                    <button
                        className={`flex-1 py-2 text-sm font-medium rounded-md transition-all ${isLogin ? 'bg-white shadow text-slate-900' : 'text-slate-500'}`}
                        onClick={() => setIsLogin(true)}
                    >
                        登录
                    </button>
                    <button
                        className={`flex-1 py-2 text-sm font-medium rounded-md transition-all ${!isLogin ? 'bg-white shadow text-slate-900' : 'text-slate-500'}`}
                        onClick={() => setIsLogin(false)}
                    >
                        注册
                    </button>
                </div>

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">用户名</label>
                        <input
                            type="text"
                            required
                            className="w-full px-4 py-2 border border-slate-200 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:outline-none transition-all"
                            value={username}
                            onChange={e => setUsername(e.target.value)}
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-slate-700 mb-1">密码</label>
                        <input
                            type="password"
                            required
                            className="w-full px-4 py-2 border border-slate-200 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:outline-none transition-all"
                            value={password}
                            onChange={e => setPassword(e.target.value)}
                        />
                    </div>
                    <button
                        type="submit"
                        disabled={loading}
                        className="w-full bg-indigo-600 text-white py-2.5 rounded-lg font-medium hover:bg-indigo-700 active:scale-[0.98] transition-all disabled:opacity-50"
                    >
                        {loading ? '处理中...' : (isLogin ? '立即登录' : '创建账号')}
                    </button>
                </form>
            </div>
        </div>
    );
}