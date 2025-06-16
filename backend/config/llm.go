package config

import (
	"standardizer/global"
	"standardizer/models"

	"github.com/tmc/langchaingo/llms/ollama"
)

// 规则定义（可逐步扩展）
var gjbRules = []string{
	"规则1: 数组索引必须使用无符号类型（如size_t）",
	"规则2: 禁止使用C风格强制类型转换，必须使用static_cast等C++风格转换",
	"规则3: 所有指针必须初始化（包括nullptr初始化）",
	// 后续添加更多规则...
}

func InitLLM() {
	// 初始化Ollama模型
	llm, err := ollama.New(
		ollama.WithModel("deepseek-r1:7b"),
		ollama.WithServerURL("http://localhost:11434"),
	)
	if err != nil {
		panic(err)
	}
	analyzer := &models.CodeAnalyzer{
		Llm:     llm,
		Rules:   gjbRules,
		Results: make(map[string][]models.Issue),
		Ctx:     global.Ctx,
	}
	global.LLM = llm
	global.CodeAnalyzer = analyzer
}
