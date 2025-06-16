package controllers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"standardizer/global"

	"github.com/gin-gonic/gin"
)

func GetResponse(ctx *gin.Context) {
	// 清空分析结果
	global.CodeAnalyzer.ClearAnalyzerResults()

	// 从请求头获取文件名
	fileName := ctx.GetHeader("File-Name")
	if fileName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "未提供文件名"})
		return
	}

	// 构建上传文件夹中的文件路径
	uploadDir := filepath.Join(".", "uploads")
	filePath := filepath.Join(uploadDir, fileName)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "文件不存在"})
		return
	}

	analyzer := global.CodeAnalyzer
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取文件信息失败"})
		return
	}

	if fileInfo.IsDir() {
		// 处理目录
		err := filepath.Walk(filePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				return analyzer.ProcessFile(path)
			}
			return nil
		})
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("遍历目录失败: %v", err)})
			return
		}
	} else {
		// 处理单个文件
		if err := analyzer.ProcessFile(filePath); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("处理文件失败: %v", err)})
			return
		}
	}

	// 检查 Ollama LLM 是否启动
	req := httptest.NewRequest("GET", "http://localhost:11434/api/tags", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ollama LLM 未启动"})
		return
	}

	// 步骤2: 生成报告
	report := analyzer.GenerateReport()
	ctx.JSON(http.StatusOK, report)
}
