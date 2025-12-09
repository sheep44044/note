package utils

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetUserID(c *gin.Context) (uint, error) {
	uidRaw, exists := c.Get("user_id")
	if !exists {
		return 0, errors.New("未登录")
	}

	uidStr, ok := uidRaw.(string)
	if !ok {
		return 0, errors.New("用户ID类型错误")
	}

	uid, err := strconv.ParseUint(uidStr, 10, 32)
	if err != nil {
		return 0, errors.New("用户ID格式错误")
	}

	return uint(uid), nil
}
