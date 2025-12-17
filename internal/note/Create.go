package note

import (
	"encoding/json"
	"fmt"
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"note/internal/validators"

	"github.com/gin-gonic/gin"
)

func (h *NoteHandler) CreateNote(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	var req validators.CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "invalid note")
		return
	}

	var tags []models.Tag
	if len(req.TagIDs) > 0 {
		h.db.Where("id IN ?", req.TagIDs).Find(&tags)
	}

	note := models.Note{
		UserID:    userID,
		Title:     req.Title,
		Content:   req.Content,
		Tags:      tags,
		IsPrivate: req.IsPrivate,
	}

	h.db.Create(&note)

	cacheKeyAllNotes := fmt.Sprintf("notes:user:%d", userID)
	h.cache.Del(c, cacheKeyAllNotes)

	if !note.IsPrivate {
		go func() {
			msg := models.FeedMsg{
				AuthorID: note.UserID,
				NoteID:   note.ID,
				PostTime: note.CreatedAt.Unix(),
			}
			body, _ := json.Marshal(msg)
			if h.rabbit != nil {
				// 只需要发这一条消息，剩下的交给消费者去扩散
				h.rabbit.Publish("feed_queue", body)
			}
		}()
	}
	utils.Success(c, note)
}
