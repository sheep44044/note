package user

import (
	"note/internal/svc"
)

type UserHandler struct {
	svc *svc.ServiceContext
}

func NewUserHandler(svc *svc.ServiceContext) *UserHandler {
	return &UserHandler{svc: svc}
}
