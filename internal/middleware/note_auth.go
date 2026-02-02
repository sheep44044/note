package middleware

import (
	"errors"
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NoteOwnerMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetUserID(c)
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, err.Error())
			return
		}

		noteIDStr := c.Param("id")
		noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
		if err != nil {
			utils.Error(c, http.StatusBadRequest, "无效的笔记ID")
			c.Abort()
			return
		}

		var note models.Note
		if err := db.Where("id = ? AND user_id = ?", noteID, userID).First(&note).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				utils.Error(c, http.StatusForbidden, "你没有权限操作这篇笔记")
			} else {
				utils.Error(c, http.StatusInternalServerError, "数据库错误")
			}
			c.Abort()
			return
		}

		c.Next()
	}
}
