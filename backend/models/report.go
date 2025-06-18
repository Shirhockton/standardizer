package models

import (
	"time"

	"gorm.io/gorm"
)

type Report struct {
	gorm.Model
	MD5Low32  string    `gorm:"size:32"`   // 前端上传文件 MD5 码低 32 位，添加索引
	Content   string    `gorm:"type:text"` // 报告内容
	CreatedAt time.Time // 生成时间
}
