package note

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"note/internal/models"
	"note/internal/utils"
	"note/internal/validators"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *NoteHandler) CreateNote(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	needSummary := c.DefaultQuery("gen_summary", "false") == "true"
	needGenTitle := c.DefaultQuery("gen_title", "false") == "true"

	var req validators.CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusUnprocessableEntity, "invalid note")
		return
	}

	title := strings.TrimSpace(req.Title)
	usingDefaultTitle := false

	if title == "" {
		defaultTitle, err := h.generateDefaultTitle(userID)
		if err != nil {
			utils.Error(c, http.StatusInternalServerError, "生成默认标题失败")
			return
		}
		title = defaultTitle
		usingDefaultTitle = true
	}

	var tags []models.Tag
	if len(req.TagIDs) > 0 {
		h.svc.DB.Where("id IN ? AND user_id = ?", req.TagIDs, userID).Find(&tags)
	}

	note := models.Note{
		UserID:    userID,
		Title:     title,
		Content:   req.Content,
		Tags:      tags,
		IsPrivate: req.IsPrivate,
	}

	if err := h.svc.DB.Create(&note).Error; err != nil {
		zap.L().Error("Create note db error", zap.Error(err))
		utils.Error(c, http.StatusInternalServerError, "创建失败")
		return
	}

	cacheKeyAllNotes := fmt.Sprintf("notes:user:%d*", userID)
	_ = h.svc.Cache.ClearCacheByPattern(c, h.svc.Cache, cacheKeyAllNotes)

	go func(n models.Note) {
		// 拼接标题和内容，让搜索更准
		textToEmbed := fmt.Sprintf("%s\n%s", n.Title, n.Content)

		vec, err := h.svc.AI.GetEmbedding(textToEmbed)
		if err != nil {
			zap.L().Error("AI embedding failed", zap.Error(err))
			return
		}

		err = h.svc.Qdrant.Upsert(context.Background(), n.ID, vec, n.UserID, n.IsPrivate)
		if err != nil {
			zap.L().Error("Qdrant upsert failed", zap.Error(err))
		}
	}(note)

	go func() {
		if usingDefaultTitle && needGenTitle {
			h.sendAITask(note.ID, "generate_title")
		}

		if needSummary {
			h.sendAITask(note.ID, "generate_summary")
		}
	}()

	if !note.IsPrivate {
		go func() {
			msg := models.FeedMsg{
				AuthorID: note.UserID,
				NoteID:   note.ID,
				PostTime: note.CreatedAt.Unix(),
			}
			body, _ := json.Marshal(msg)
			if h.svc.Rabbit != nil {
				// 只需要发这一条消息，剩下的交给消费者去扩散
				_ = h.svc.Rabbit.Publish("feed_queue", body)
			}
		}()
	}
	utils.Success(c, note)
}

func (h *NoteHandler) sendAITask(noteID uint, taskType string) {
	if h.svc.Rabbit == nil {
		return
	}
	msg := models.AITaskMsg{
		NoteID: noteID,
		Task:   taskType,
	}
	body, _ := json.Marshal(msg)
	_ = h.svc.Rabbit.Publish("ai_queue", body)
}

func (h *NoteHandler) generateDefaultTitle(userID uint) (string, error) {
	today := time.Now().Format("2006-01-02")
	baseTitle := fmt.Sprintf("笔记 %s", today)
	finalTitle := baseTitle

	var count int64
	suffix := 1

	for {
		err := h.svc.DB.Model(&models.Note{}).
			Where("user_id = ? AND title = ?", userID, finalTitle).
			Count(&count).Error

		if err != nil {
			return "", err
		}

		if count == 0 {
			break
		}

		finalTitle = fmt.Sprintf("%s (%d)", baseTitle, suffix)
		suffix++

		// 防止死循环的保底（虽然不太可能）
		if suffix > 100 {
			break
		}
	}

	return finalTitle, nil
}
