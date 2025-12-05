package validators

type RegisterUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UpdateProfileRequest struct {
	Username *string `json:"username,omitempty" binding:"omitempty,min=2,max=20"`
	Avatar   *string `json:"avatar,omitempty" binding:"omitempty,url"`
	Bio      *string `json:"bio,omitempty" binding:"omitempty,max=150"`
}
