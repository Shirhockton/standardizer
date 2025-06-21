package controllers

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"standardizer/global"
	"standardizer/utils"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
	"github.com/xuri/excelize/v2"

	"standardizer/models"
)

func GetResponse(ctx *gin.Context) {
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
		slog.Error("文件不存在", "file", filePath)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "文件不存在"})
		return
	}
	md5Low32 := utils.CalcMd5(filePath)
	// 检查文件报告是否已在数据库中，若在，则直接返回报告
	var reportModel models.Report
	if err := global.Db.Where("md5_low32 = ?", md5Low32).First(&reportModel).Error; err == nil {
		slog.Info("文件报告已存在于数据库", "file", filePath)
		ctx.JSON(http.StatusOK, reportModel)
		return
	}

	// 若数据库中无报告，将任务发布到消息队列
	ch, err := global.RabbitMQConn.Channel()
	if err != nil {
		slog.Error("打开 RabbitMQ 通道失败", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "无法连接到消息队列"})
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"file_scan_queue", // 队列名称
		true,              // 持久化
		false,             // 自动删除
		false,             // 排他
		false,             // 等待服务器响应
		nil,               // 额外参数
	)
	if err != nil {
		slog.Error("声明 RabbitMQ 队列失败", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "无法声明消息队列"})
		return
	}

	err = ch.Publish(
		"",     // 交换器
		q.Name, // 路由键
		false,  // 强制
		false,  // 立即
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         []byte(filePath),
		})
	if err != nil {
		slog.Error("发布消息到 RabbitMQ 失败", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "无法发布扫描任务"})
		return
	}

	// 返回任务已接收状态
	ctx.JSON(http.StatusAccepted, gin.H{"message": "文件扫描任务已接收，请稍后查询结果", "md5_low32": md5Low32})
}

// 保存 Excel 文件
func SaveExcelReport(report map[string]interface{}) {
	slog.Info("开始保存Excel报告")
	// 提取第一个文件路径作为文件名基础
	// var baseFileName string
	// for file := range c.Results {
	// 	baseFileName = file
	// 	break
	// }
	// if baseFileName == "" {
	// 	baseFileName = "default"
	// }

	// // 提取文件名并去掉扩展名
	// lastSlash := strings.LastIndex(baseFileName, "\\")
	// if lastSlash == -1 {
	// 	lastSlash = 0
	// }
	// lastDot := strings.LastIndex(baseFileName[lastSlash+1:], ".")
	// if lastDot == -1 {
	// 	lastDot = len(baseFileName[lastSlash+1:])
	// }
	// fileName := baseFileName[lastSlash+1 : lastSlash+1+lastDot]

	// 构建完整文件路径
	dir := "results"
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Error("创建结果目录失败", "error", err)
		// fmt.Printf("创建结果目录失败: %v\n", err)
		return
	}
	fullPath := filepath.Join(dir, fmt.Sprintf("%s_result.xlsx", report["file-name"]))

	f := excelize.NewFile()
	sheetName := "CodeAnalysisReport"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		slog.Error("创建 Excel 工作表失败", "error", err)
		// fmt.Printf("创建 Excel 工作表失败: %v\n", err)
		return
	}
	f.SetActiveSheet(index)

	// 设置表头
	headers := []string{"文件", "行号", "规则", "问题描述", "建议修正"}
	for colIndex, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(colIndex+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	row := 2
	for _, issue := range report["issues"].([]map[string]interface{}) {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), issue["file"])
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), issue["line"])
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), issue["rule"])
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), issue["original"])
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), issue["suggested"])
		row++
	}

	// 保存 Excel 文件
	if err := f.SaveAs(fullPath); err != nil {
		slog.Error("保存 Excel 报告失败", "file", fullPath, "error", err)
		// fmt.Printf("保存 Excel 报告失败: %v\n", err)
	} else {
		slog.Info("Excel 报告已生成", "file", fullPath)
	}
	slog.Info(" 总计:", report["total_files"], "个文件", report["total_issues"], "处问题")
	fmt.Printf("\n=== 总计: %d个文件, %d处问题 ===\n", report["total_files"], report["total_issues"])
	// fmt.Printf("Excel 报告已生成: %s\n", fullPath)
}

// 新增下载报告接口
func DownloadReport(ctx *gin.Context) {
	// 从请求头获取报告文件名
	fileName := ctx.GetHeader("File-Name")
	if fileName == "" {
		slog.Error("未提供文件名")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "未提供文件名"})
		return
	}

	// 构建报告文件路径
	lastDot := strings.LastIndex(fileName, ".")
	if lastDot == -1 {
		lastDot = len(fileName)
	}
	fileName = fileName[:lastDot]
	reportName := fileName + "_result.xlsx"
	reportDir := filepath.Join(".", "results")
	reportPath := filepath.Join(reportDir, reportName)

	// 检查报告文件是否存在
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		slog.Error("报告文件不存在")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "报告文件不存在"})
		return
	}

	// 设置响应头
	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", reportName))
	ctx.Header("Content-Transfer-Encoding", "binary")
	ctx.Header("Expires", "0")
	ctx.Header("Cache-Control", "must-revalidate")
	ctx.Header("Pragma", "public")
	ctx.Header("File-Name", reportName)

	// 发送文件
	ctx.File(reportPath)
}

// ProcessFileOrDirectory 处理文件或目录的逻辑
func ProcessFileOrDirectory(filePath string, analyzer *models.CodeAnalyzer) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		slog.Error("获取文件信息失败", "error", err)
		return err
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
			return fmt.Errorf("遍历目录失败: %w", err)
		}
	} else {
		// 处理单个文件
		if err := analyzer.ProcessFile(filePath); err != nil {
			return fmt.Errorf("处理文件失败: %w", err)
		}
	}

	// 检查 Ollama LLM 是否启动
	req, err := http.NewRequest("GET", "http://localhost:11434/api/tags", nil)
	if err != nil {
		slog.Error("创建请求失败", "error", err)
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		slog.Error("Ollama LLM 未启动")
		return fmt.Errorf("Ollama LLM 未启动")
	}
	return nil
}

func CheckReport(ctx *gin.Context) {
	md5Low32 := ctx.Query("md5_low32")
	if md5Low32 == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "未提供 md5_low32"})
		return
	}

	var reportModel models.Report
	if err := global.Db.Where("md5_low32 = ?", md5Low32).First(&reportModel).Error; err == nil {
		ctx.JSON(http.StatusOK, reportModel)
	} else {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "报告尚未生成"})
	}
}
