package models

type PersonalPage struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar,omitempty"`
	Bio      string `json:"bio,omitempty"`

	FollowCount int `json:"follow_count"`
	FanCount    int `json:"fan_count"`

	IsFollowing bool `json:"is_following"`

	CreatedAt string `json:"created_at"`

	Documents []NoteBrief `json:"documents"`
}

type NoteBrief struct {
	ID            uint     `json:"id"`
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	FavoriteCount int      `json:"favorite_count"`
	IsPrivate     bool     `json:"is_private"`
	IsPinned      bool     `json:"is_pinned"`
	Tags          []string `json:"tags,omitempty"`
	UpdatedAt     string   `json:"updated_at"`
}

type UserBrief struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	Avatar      string `json:"avatar"`
	Bio         string `json:"bio"`
	IsFollowing bool   `json:"is_following"`
}
