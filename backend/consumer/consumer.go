package consumer

import (
	"encoding/json"
	"log/slog"
	"os"
	"standardizer/controllers"
	"standardizer/global"
	"standardizer/models"
	"standardizer/utils"
	"time"
)

// StartConsumer 启动 RabbitMQ 消费者协程
func StartConsumer() {
	go func() {
		for {
			// 创建 RabbitMQ 通道
			ch, err := global.RabbitMQConn.Channel()
			if err != nil {
				slog.Error("打开 RabbitMQ 通道失败", "error", err)
				time.Sleep(5 * time.Second) // 等待 5 秒后重试
				continue
			}

			// 声明队列
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
				ch.Close()
				time.Sleep(5 * time.Second) // 等待 5 秒后重试
				continue
			}

			// 注册消费者
			msgs, err := ch.Consume(
				q.Name, // 队列名称
				"",     // 消费者名称
				true,   // 自动确认
				false,  // 排他
				false,  // 本地
				false,  // 等待服务器响应
				nil,    // 额外参数
			)
			if err != nil {
				slog.Error("注册 RabbitMQ 消费者失败", "error", err)
				ch.Close()
				time.Sleep(5 * time.Second) // 等待 5 秒后重试
				continue
			}

			slog.Info(" [*] 等待文件扫描任务消息。")
			// 处理接收到的消息
			for d := range msgs {
				// 执行文件扫描逻辑
				filePath := string(d.Body)
				err := controllers.ProcessFileOrDirectory(filePath, global.CodeAnalyzer)
				if err != nil {
					slog.Error("处理文件或目录失败", "error", err)
					continue
				}

				//生成报告
				report := global.CodeAnalyzer.GenerateReport(filePath)

				// 保存报告到数据库
				SaveReportInDB(filePath, report)
			}

			// 如果消息通道关闭，尝试重新连接
			ch.Close()
			time.Sleep(5 * time.Second)
		}
	}()
}
func SaveReportInDB(filePath string, report map[string]interface{}) {
	// filePath := c.PostForm("filePath")
	// 假设这里有生成报告内容的逻辑
	// 	// reportContent := generateReport(filePath)

	// 读取文件，计算文件内容的 MD5 低 32 位
	file, err := os.Open(filePath)
	if err != nil {
		slog.Error("打开文件失败", "error", err)
		return
	}
	defer file.Close()

	// 计算文件 MD5
	md5Low32 := utils.CalcMd5(filePath)

	// 转换报告内容为 JSON 字符串
	reportJSON, err := json.Marshal(report)
	if err != nil {
		slog.Error("报告内容转换为 JSON 失败", "error", err)
		// fmt.Printf("报告内容转换为 JSON 失败: %v\n", err)
		return
	}

	// 创建 Report 实例
	reportModel := models.Report{
		MD5Low32:  md5Low32,
		Content:   string(reportJSON),
		CreatedAt: time.Now(),
	}

	// 保存报告到数据库
	if err := global.Db.AutoMigrate(&models.Report{}); err != nil {
		slog.Error("自动迁移数据库失败", "error", err)
		// ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := global.Db.Create(&reportModel).Error; err != nil {
		slog.Error("保存报告到数据库失败", "error", err)
		return
	}
}
