package note

import (
	"encoding/json"
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *NoteHandler) ReactToNote(c *gin.Context) {
	noteID := c.Param("id")
	noteIDUint64, _ := strconv.ParseUint(noteID, 10, 64)

	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	var input struct {
		Emoji string `json:"emoji" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, http.StatusBadRequest, "需要 emoji")
		return
	}

	// 校验 emoji（简单白名单）
	validEmojis := map[string]bool{
		"❤️": true, "👍": true, "🔥": true, "👏": true, "😂": true, "😮": true,
	}
	if !validEmojis[input.Emoji] {
		utils.Error(c, http.StatusBadRequest, "不支持的 emoji")
		return
	}

	msg := models.ReactionMsg{
		UserID: userID,
		NoteID: uint(noteIDUint64),
		Emoji:  input.Emoji,
		Action: "toggle",
	}

	body, _ := json.Marshal(msg)
	if err := h.svc.Rabbit.Publish("react_queue", body); err != nil {
		utils.Error(c, http.StatusInternalServerError, "操作失败")
		return
	}

	// 清理缓存（笔记详情缓存）
	// 注意：这里可能需要清理很频繁，如果是高并发场景，建议只更新 Redis 的 Hash 计数，不删整个 Key
	_ = h.svc.Cache.Del(c, "note:"+noteID)

	utils.Success(c, gin.H{"message": "操作已接收"})
}
