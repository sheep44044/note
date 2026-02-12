// 用户信息
export interface User {
    id: number;
    username: string;
    nickname?: string; // 兼容前端显示
    avatar?: string;   // 对应后端 /users/me 返回的字段
    avatar_url?: string; // 兼容旧定义
    bio?: string;
    created_at?: string;
}

export interface UserProfile extends User {
    bio?: string;
    follow_count?: number; // 关注数
    fan_count?: number;    // 粉丝数
    is_following?: boolean; // 当前用户是否关注了该用户
    documents?: Note[];    // 该用户发布的笔记列表
}

// 标签信息
export interface Tag {
    id: number;
    name: string;
    color?: string;
    user_id?: number;
}

// 核心笔记对象 (严格对应 Go 后端 JSON Tag)
export interface Note {
    id: number;
    title: string;
    content: string;
    summary?: string;
    created_at: string;
    updated_at: string;

    // 👇 Snake_case 字段 (对应后端 is_private)
    is_private: boolean;

    // 👇 PascalCase 字段 (对应后端 Go 结构体导出字段)
    UserID: number;
    IsPinned?: boolean;      // 注意大写 I
    IsFavorite?: boolean;    // 注意大写 I
    FavoriteCount?: number;  // 注意大写 F

    // 关联数据
    user?: User;             // 作者信息 (如果后端 Preload 了)
    Tags?: Tag[];            // 注意大写 T
    reaction_counts?: Record<string, number>; // 对应 map[string]int
}

// 通用 API 响应结构
export interface ApiResponse<T = any> {
    code: number;
    message: string;
    data: T;
}

// 认证响应
export interface AuthResponse {
    code: number;
    message: string;
    data: {
        token: string;
        user: User;
    };
}

// 笔记列表响应
// 后端 /notes 直接返回数组，但 /notes/search 返回对象，这里做联合类型兼容
export interface NoteListResponse {
    code: number;
    message: string;
    data: Note[] | {
        list?: Note[];
        notes?: Note[]; // 搜索接口返回的是 notes 字段
        total: number;
        page?: number;
    };
}