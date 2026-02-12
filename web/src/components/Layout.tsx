import React from 'react';
import { Link, useLocation, Outlet, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Home, Globe, Settings, LogOut, Menu, Search, UserCheck, Tag as TagIcon } from 'lucide-react';
import { cn } from '@/lib/utils';
import api from '@/lib/axios';
import type {UserProfile, Tag} from '@/types';

// 侧边栏单个导航项组件
const SidebarItem = ({ icon: Icon, label, to, active }: any) => (
    <Link
        to={to}
        className={cn(
            "flex items-center gap-3 px-4 py-3 rounded-xl transition-all duration-200 group",
            active
                ? "bg-slate-900 text-white shadow-md"
                : "text-slate-500 hover:bg-slate-100 hover:text-slate-900"
        )}
    >
        <Icon size={20} strokeWidth={active ? 2.5 : 2} />
        <span className="font-medium">{label}</span>
    </Link>
);

export default function Layout() {
    const location = useLocation();
    const navigate = useNavigate();
    const [isMobileMenuOpen, setIsMobileMenuOpen] = React.useState(false);

    // 1. 获取当前登录用户信息 (用于侧边栏底部显示)
    const { data: currentUser } = useQuery({
        queryKey: ['users', 'me'],
        queryFn: async () => {
            try {
                const res = await api.get<any, { data: UserProfile }>('/users/me');
                return res.data;
            } catch (error) {
                return null;
            }
        },
        retry: false
    });

    // 2. 获取所有标签 (用于侧边栏标签云展示)
    const { data: tags = [] } = useQuery({
        queryKey: ['tags'],
        queryFn: async () => {
            try {
                const res = await api.get<any, { data: Tag[] | Tag[] }>('/tags');
                // 兼容处理：有的接口直接返回数组，有的返回 { data: [] }
                if (Array.isArray(res.data)) return res.data;
                if (Array.isArray((res.data as any)?.data)) return (res.data as any).data;
                return [];
            } catch (e) { return []; }
        },
        staleTime: 1000 * 60 * 5 // 5分钟缓存
    });

    const handleLogout = () => {
        if (confirm('确定要退出登录吗？')) {
            localStorage.removeItem('token');
            navigate('/login');
        }
    };

    // 导航菜单配置
    const navItems = [
        { icon: Home, label: '我的笔记', path: '/' },
        { icon: Search, label: '搜索', path: '/search' },
        { icon: UserCheck, label: '关注动态', path: '/following' },
        { icon: Globe, label: '探索广场', path: '/community' },
    ];

    return (
        <div className="min-h-screen bg-[#f3f4f6] flex flex-col md:flex-row">
            {/* Mobile Header (仅在移动端显示) */}
            <div className="md:hidden bg-white px-4 py-3 flex items-center justify-between border-b sticky top-0 z-50 shadow-sm">
                <h1 className="font-bold text-xl text-slate-800">MyNotes</h1>
                <button onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}>
                    <Menu className="text-slate-600" />
                </button>
            </div>

            {/* Sidebar (Desktop & Mobile Overlay) */}
            <aside className={cn(
                "fixed md:sticky top-0 h-screen w-64 bg-white border-r border-slate-200 p-6 flex flex-col justify-between z-40 transition-transform duration-300 md:translate-x-0 shadow-xl md:shadow-none overflow-y-auto custom-scrollbar",
                isMobileMenuOpen ? "translate-x-0" : "-translate-x-full"
            )}>
                {/* Top Section: Logo & Nav */}
                <div>
                    <div className="mb-8 px-4 hidden md:block">
                        <h1 className="text-2xl font-black tracking-tight text-slate-900 flex items-center gap-2">
                            <span className="w-2 h-8 bg-indigo-600 rounded-full inline-block"></span>
                            MyNotes.
                        </h1>
                        <p className="text-xs text-slate-400 mt-1 pl-3">你的数字后花园</p>
                    </div>

                    <nav className="space-y-1 mb-8">
                        {navItems.map((item) => (
                            <SidebarItem
                                key={item.path}
                                {...item}
                                to={item.path}
                                active={location.pathname === item.path}
                            />
                        ))}
                    </nav>

                    {/* 标签云区域 (新增) */}
                    <div className="px-4 mb-6">
                        <h3 className="text-xs font-bold text-slate-400 uppercase tracking-wider mb-3 flex items-center gap-1">
                            <TagIcon size={12} /> 标签筛选
                        </h3>
                        <div className="flex flex-wrap gap-2">
                            {tags.length > 0 ? tags.map((tag: Tag) => (
                                <Link
                                    key={tag.id}
                                    to={`/?tag_id=${tag.id}`}
                                    className={cn(
                                        "text-xs px-2.5 py-1 rounded-full border transition-colors",
                                        location.search.includes(`tag_id=${tag.id}`)
                                            ? "bg-indigo-100 text-indigo-700 border-indigo-200"
                                            : "bg-slate-50 text-slate-600 border-slate-100 hover:bg-indigo-50 hover:text-indigo-600 hover:border-indigo-100"
                                    )}
                                >
                                    #{tag.name}
                                </Link>
                            )) : (
                                <span className="text-xs text-slate-300 italic">暂无标签</span>
                            )}
                        </div>
                    </div>
                </div>

                {/* Bottom Section: User & Settings */}
                <div className="border-t pt-4 space-y-1">
                    {/* 设置链接 */}
                    <Link
                        to="/settings"
                        className={cn(
                            "flex items-center gap-3 px-4 py-3 rounded-xl transition-all duration-200 group text-slate-500 hover:bg-slate-100 hover:text-slate-900",
                            location.pathname === '/settings' && "bg-slate-100 text-slate-900"
                        )}
                    >
                        <Settings size={20} />
                        <span className="font-medium">设置</span>
                    </Link>

                    {/* 当前用户迷你卡片 */}
                    {currentUser && (
                        <Link to={`/u/${currentUser.id}`} className="flex items-center gap-3 px-4 py-2 mt-2 hover:bg-indigo-50 rounded-xl transition-colors group cursor-pointer border border-transparent hover:border-indigo-100">
                            <div className="w-9 h-9 rounded-full bg-slate-200 flex items-center justify-center text-slate-500 font-bold text-sm overflow-hidden border border-white shadow-sm group-hover:border-indigo-200 shrink-0">
                                {currentUser.avatar ? (
                                    <img src={currentUser.avatar} alt="me" className="w-full h-full object-cover" />
                                ) : (
                                    (currentUser.nickname?.[0] || currentUser.username?.[0] || 'U').toUpperCase()
                                )}
                            </div>
                            <div className="flex flex-col overflow-hidden">
                   <span className="text-sm font-bold text-slate-700 truncate group-hover:text-indigo-700">
                     {currentUser.nickname || currentUser.username}
                   </span>
                                <span className="text-xs text-slate-400 group-hover:text-indigo-400">查看主页</span>
                            </div>
                        </Link>
                    )}

                    {/* 退出按钮 */}
                    <button
                        onClick={handleLogout}
                        className="w-full flex items-center gap-3 px-4 py-3 text-red-500 hover:bg-red-50 rounded-xl transition-colors mt-1"
                    >
                        <LogOut size={20} />
                        <span className="font-medium">退出登录</span>
                    </button>
                </div>
            </aside>

            {/* Main Content Area */}
            <main className="flex-1 max-w-4xl mx-auto w-full p-4 md:p-8 overflow-y-auto">
                <Outlet />
            </main>
        </div>
    );
}