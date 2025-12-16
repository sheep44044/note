package mq

import (
	"encoding/json"
	"errors"
	"log/slog"
	"note/internal/models"

	"gorm.io/gorm"
)

// Consumer 结构体用于持有 DB 连接等依赖
type Consumer struct {
	db     *gorm.DB
	rabbit *RabbitMQ
}

// NewConsumer 初始化消费者管理器
func NewConsumer(db *gorm.DB, rabbit *RabbitMQ) *Consumer {
	return &Consumer{
		db:     db,
		rabbit: rabbit,
	}
}

// Start 启动所有消费者监听
func (c *Consumer) Start() {
	// 启动点赞消费者
	go c.consumeFavorite()
	go c.consumeReaction()
	go c.ConsumeHistory()
	// 这里可以启动其他消费者...
}

func (c *Consumer) consumeFavorite() {
	msgs, err := c.rabbit.Consume("favorite_queue")
	if err != nil {
		slog.Error("Failed to start favorite consumer", "error", err)
		return
	}

	slog.Info("Waiting for favorite messages...")

	for d := range msgs {
		var msg models.FavoriteMsg
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			slog.Error("Failed to unmarshal msg", "error", err)
			continue // 格式错误直接丢弃
		}

		if msg.Action == "add" {
			fav := models.Favorite{UserID: msg.UserID, NoteID: msg.NoteID}
			if err := c.db.Create(&fav).Error; err != nil {
				// 只有非重复键错误才当做异常处理
				if !errors.Is(err, gorm.ErrDuplicatedKey) {
					slog.Error("Failed to insert favorite", "error", err)
					// TODO: 在这里可以做重试逻辑，或者手动 Nack
				}
			} else {
				// 更新计数（+1）
				c.db.Model(&models.Note{}).Where("id = ?", msg.NoteID).
					Update("favorite_count", gorm.Expr("favorite_count + 1"))
				slog.Info("Favorite added", "user_id", msg.UserID, "note_id", msg.NoteID)
			}
		} else if msg.Action == "remove" {
			// 处理取消收藏
			if err := c.db.Where("user_id = ? AND note_id = ?", msg.UserID, msg.NoteID).
				Delete(&models.Favorite{}).Error; err == nil {

				c.db.Model(&models.Note{}).Where("id = ?", msg.NoteID).
					Update("favorite_count", gorm.Expr("GREATEST(favorite_count - 1, 0)"))
				slog.Info("Favorite removed", "user_id", msg.UserID, "note_id", msg.NoteID)
			}
		}
	}
}

func (c *Consumer) consumeReaction() {
	msgs, err := c.rabbit.Consume("react_queue")
	if err != nil {
		slog.Error("Failed to start react consumer", "error", err)
		return
	}

	slog.Info("Waiting for react messages...")
	for d := range msgs {
		var msg models.ReactionMsg
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			slog.Error("Failed to unmarshal msg", "error", err)
			continue
		}
		if msg.Action == "add" {
			reaction := models.Reaction{
				UserID: msg.UserID,
				NoteID: msg.NoteID,
				Emoji:  msg.Emoji,
			}
			if err := c.db.Create(&reaction).Error; err != nil {
				if !errors.Is(err, gorm.ErrDuplicatedKey) {
					slog.Error("Failed to insert reaction", "error", err)
				}
			}
		}
	}
}

func (c *Consumer) ConsumeHistory() {
	msgs, err := c.rabbit.Consume("history_queue")
	if err != nil {
		slog.Error("Failed to start react consumer", "error", err)
		return
	}

	slog.Info("Waiting for react messages...")
	for d := range msgs {
		var msg models.HistoryMsg
		if err := json.Unmarshal(d.Body, &msg); err != nil {
			slog.Error("Failed to unmarshal msg", "error", err)
			continue
		}

		c.db.Where("user_id = ? AND note_id = ?", msg.UserID, msg.NoteID).Delete(&models.History{})

		history := models.History{
			UserID: msg.UserID,
			NoteID: msg.NoteID,
		}

		if err := c.db.Create(&history).Error; err != nil {
			slog.Error("Failed to insert history", "error", err)
		}
	}
}
