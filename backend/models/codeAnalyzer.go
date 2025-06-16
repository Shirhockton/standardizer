package models

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	// "standardizer/global"
	"standardizer/utils"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"
	"github.com/xuri/excelize/v2"
)

// 代码分析结构体
type CodeAnalyzer struct {
	Llm     llms.Model
	Rules   []string
	Results map[string][]Issue
	Mu      sync.Mutex
	Ctx     context.Context
}

// 问题描述
type Issue struct {
	File      string
	Line      int
	Rule      string
	Original  string
	Suggested string
}

// 处理单个文件
func (c *CodeAnalyzer) ProcessFile(path string) error {

	// 只处理C++文件
	if !strings.HasSuffix(path, ".cpp") && !strings.HasSuffix(path, ".h") && !strings.HasSuffix(path, ".hpp") {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("读取文件失败 %s: %v\n", path, err)
		return nil
	}

	// 分块处理大文件
	chunks := utils.SplitCodeIntoChunks(string(content), 500) // 500行/块

	for i, chunk := range chunks {
		startLine := i * 500
		c.analyzeCodeChunk(path, chunk, startLine)
	}

	return nil
}

// 分析代码块
func (c *CodeAnalyzer) analyzeCodeChunk(filePath, code string, startLine int) {
	// 构造LLM提示
	prompt := c.buildPrompt(code)

	// 调用LLM
	response, err := c.Llm.Call(c.Ctx, prompt)
	if err != nil {
		fmt.Printf("LLM分析失败 %s: %v\n", filePath, err)
		return
	}

	// 解析响应
	issues := parseLLMResponse(response, filePath, startLine)

	// 存储结果
	c.Mu.Lock()
	c.Results[filePath] = append(c.Results[filePath], issues...)
	c.Mu.Unlock()
}

// 构建LLM提示
func (c *CodeAnalyzer) buildPrompt(code string) string {
	rulesStr := strings.Join(c.Rules, "\n")

	return fmt.Sprintf(`你是一个C++专家，正在检查代码是否符合代码规范。请遵循以下规则：
%s

请分析以下C++代码片段：

输出格式要求：
1. 按行分析，每行格式：[行号]:[规则编号]:[问题描述]:[建议修正]
2. 如果没有问题，输出"共检查xx行代码，没有问题"
3. 示例：
   42:规则2:危险的类型转换:使用static_cast<int>(value)代替(int)value

请开始分析：`, rulesStr, code)
}

// 生成报告
func (c *CodeAnalyzer) GenerateReport() map[string]interface{} {
	report := make(map[string]interface{})
	report["title"] = "代码规范检查报告"
	report["rule_count"] = len(c.Rules)

	var issues []map[string]interface{}
	totalIssues := 0
	for file, fileIssues := range c.Results {
		for _, issue := range fileIssues {
			issueData := map[string]interface{}{
				"file":      file,
				"line":      issue.Line,
				"rule":      issue.Rule,
				"original":  issue.Original,
				"suggested": issue.Suggested,
			}
			issues = append(issues, issueData)
			totalIssues++
		}
	}

	report["total_files"] = len(c.Results)
	report["total_issues"] = totalIssues
	report["issues"] = issues
	c.SaveExcelReport(report)
	return report
}

// 保存 Excel 文件
func (c *CodeAnalyzer) SaveExcelReport(report map[string]interface{}) {
	// 提取第一个文件路径作为文件名基础
	var baseFileName string
	for file := range c.Results {
		baseFileName = file
		break
	}
	if baseFileName == "" {
		baseFileName = "default"
	}

	// 提取文件名并去掉扩展名
	lastSlash := strings.LastIndex(baseFileName, "\\")
	if lastSlash == -1 {
		lastSlash = 0
	}
	lastDot := strings.LastIndex(baseFileName[lastSlash+1:], ".")
	if lastDot == -1 {
		lastDot = len(baseFileName[lastSlash+1:])
	}
	fileName := baseFileName[lastSlash+1 : lastSlash+1+lastDot]

	// 构建完整文件路径
	dir := "results"
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("创建结果目录失败: %v\n", err)
		return
	}
	fullPath := filepath.Join(dir, fmt.Sprintf("%s_result.xlsx", fileName))

	f := excelize.NewFile()
	sheetName := "CodeAnalysisReport"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		fmt.Printf("创建 Excel 工作表失败: %v\n", err)
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
		fmt.Printf("保存 Excel 报告失败: %v\n", err)
	}

	fmt.Printf("\n=== 总计: %d个文件, %d处问题 ===\n", report["total_files"], report["total_issues"])
	fmt.Printf("Excel 报告已生成: %s\n", fullPath)
}

// 解析LLM响应
func parseLLMResponse(response, filePath string, startLine int) []Issue {
	var issues []Issue

	lines := strings.Split(response, "\n")
	re := regexp.MustCompile(`(\d+):规则(\d+):([^:]+):(.+)`)

	for _, line := range lines {
		if line == "OK" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) != 5 {
			continue
		}

		lineNum := startLine + utils.Atoi(matches[1])
		issues = append(issues, Issue{
			File:      filePath,
			Line:      lineNum,
			Rule:      "规则" + matches[2],
			Original:  matches[3],
			Suggested: matches[4],
		})
	}
	return issues
}

func (c *CodeAnalyzer) ClearAnalyzerResults() {
	c.Results = make(map[string][]Issue)
}

// func GetResponse(ctx *gin.Context) {
// 	// 设置工程路径
// 	projectPath := "./sample_project"
// 	analyzer := global.CodeAnalyzer
// 	// 步骤1: 遍历C++工程
// 	if err := filepath.Walk(projectPath, analyzer.ProcessFile); err != nil {
// 		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("遍历工程失败: %v", err)})
// 		return
// 	}

// 	// 步骤2: 生成报告
// 	report := analyzer.GenerateReport()
// 	ctx.JSON(http.StatusOK, report)

// 	// 步骤3: 保存 Excel 报告
// 	go analyzer.SaveExcelReport()
// }
