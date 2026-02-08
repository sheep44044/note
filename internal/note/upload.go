package note

import (
	"fmt"
	"note/internal/utils"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var allowedImages = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

func (h *NoteHandler) UploadImage(c *gin.Context) {
	// 1. 获取上传的文件
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		utils.Error(c, 400, "请上传图片")
		return
	}
	defer file.Close()

	const MaxFileSize = 5 * 1024 * 1024
	if header.Size > MaxFileSize {
		utils.Error(c, 400, "图片不能超过 5MB")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if !allowedImages[contentType] {
		utils.Error(c, 400, "不支持的图片格式")
		return
	}

	// 使用 UUID 生成文件名
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	// 生成类似 "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11.jpg"
	newFileName := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	url, err := h.svc.Minio.UploadImage(c, newFileName, header.Size, file, contentType)
	if err != nil {
		zap.L().Error("MinIO upload failed", zap.Error(err))
		utils.Error(c, 500, "图片上传服务繁忙")
		return
	}

	utils.Success(c, gin.H{"url": url})
}
