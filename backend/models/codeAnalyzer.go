package models

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"standardizer/utils"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"
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
	slog.Info("开始处理文件", "file", path)

	// 只处理C++文件
	if !strings.HasSuffix(path, ".cpp") && !strings.HasSuffix(path, ".h") && !strings.HasSuffix(path, ".hpp") {
		slog.Debug("跳过非C++文件", "file", path)
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		slog.Error("读取文件失败", "file", path, "error", err)
		return err
	}

	// 分块处理大文件
	chunks := utils.SplitCodeIntoChunks(string(content), 500) // 500行/块
	slog.Debug("文件分块完成", "file", path, "chunk_count", len(chunks))

	for i, chunk := range chunks {
		startLine := i * 500
		c.analyzeCodeChunk(path, chunk, startLine)
	}

	slog.Info("文件处理完成", "file", path)
	return nil
}

// 分析代码块
func (c *CodeAnalyzer) analyzeCodeChunk(filePath, code string, startLine int) {
	slog.Info("开始分析代码块", "file", filePath, "start_line", startLine)
	// 构造LLM提示
	prompt := c.buildPrompt(code)

	// 调用LLM
	response, err := c.Llm.Call(c.Ctx, prompt)
	if err != nil {
		slog.Error("LLM分析失败", "file", filePath, "error", err)
		return
	}
	slog.Debug("LLM分析成功", "file", filePath)

	// 解析响应
	issues := parseLLMResponse(response, filePath, startLine)

	// 存储结果
	c.Mu.Lock()
	c.Results[filePath] = append(c.Results[filePath], issues...)
	c.Mu.Unlock()
	slog.Debug("结果存储完成", "file", filePath, "issue_count", len(issues))
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

请开始分析：%s`, rulesStr, code)
}

// 提取文件名并去掉扩展名
func extractFileName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// 生成报告
func (c *CodeAnalyzer) GenerateReport(path string) map[string]interface{} {
	slog.Info("开始生成报告")

	fileName := extractFileName(path)

	report := make(map[string]interface{})
	report["file-name"] = fileName
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

	slog.Info("报告生成完成", "total_files", report["total_files"], "total_issues", report["total_issues"])
	return report
}

// 解析LLM响应
func parseLLMResponse(response, filePath string, startLine int) []Issue {
	slog.Debug("开始解析LLM响应", "file", filePath)
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
	slog.Debug("LLM响应解析完成", "file", filePath, "issue_count", len(issues))
	return issues
}

func (c *CodeAnalyzer) ClearAnalyzerResults() {
	c.Results = make(map[string][]Issue)
}
