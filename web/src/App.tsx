import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import Layout from './components/Layout';
import HomePage from './pages/HomePage';
import LoginPage from './pages/LoginPage';
import CommunityPage from './pages/CommunityPage';
import SearchPage from './pages/SearchPage';
import ProfilePage from './pages/ProfilePage';
import SettingsPage from './pages/SettingsPage';
import FollowingPage from './pages/FollowingPage';
import NoteDetailPage from './pages/NoteDetailPage';

const queryClient = new QueryClient();

const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
    const token = localStorage.getItem('token');
    if (!token) return <Navigate to="/login" replace />;
    // 这里需要强制类型转换为 JSX.Element，或者直接返回 children (React Router 6 支持直接渲染 Node)
    return <>{children}</>;
};

function App() {
    return (
        <QueryClientProvider client={queryClient}>
            <BrowserRouter>
                <Routes>
                    <Route path="/login" element={<LoginPage />} />

                    <Route path="/" element={
                        <ProtectedRoute>
                            <Layout />
                        </ProtectedRoute>
                    }>
                        <Route index element={<HomePage />} />
                        <Route path="community" element={<CommunityPage />} />
                        <Route path="search" element={<SearchPage />} />
                        <Route path="u/:id" element={<ProfilePage />} />
                        <Route path="settings" element={<SettingsPage />} />
                        <Route path="following" element={<FollowingPage />} />
                        <Route path="/notes/:id" element={<NoteDetailPage />} />
                    </Route>
                </Routes>
            </BrowserRouter>
        </QueryClientProvider>
    );
}

export default App;