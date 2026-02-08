package user

import (
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"note/internal/validators"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func (h *UserHandler) Register(c *gin.Context) {
	var req validators.RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid request")
		return
	}

	var exists models.User
	if h.svc.DB.Where("username = ?", req.Username).First(&exists).RowsAffected > 0 {
		utils.Error(c, http.StatusConflict, "username already exists")
		return
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	user := models.User{
		Username: req.Username,
		Password: string(hashed),
	}
	h.svc.DB.Create(&user)

	utils.Success(c, gin.H{"message": "user registered"})
}
