package scraper

import "fmt"

// ScrapeError 定义爬虫错误类型
type ScrapeError struct {
	// 错误类型
	Type ErrorType
	// 具体错误信息
	Message string
	// 原始错误
	Cause error
}

// ErrorType 错误类型枚举
type ErrorType int

const (
	// ErrorTypeUnknown 未知错误
	ErrorTypeUnknown ErrorType = iota
	// ErrorTypeLoadFailed 页面加载失败（可重试）
	ErrorTypeLoadFailed
	// ErrorTypeTimeout 超时错误（可重试）
	ErrorTypeTimeout
	// ErrorTypeParseError 解析错误（不可重试）
	ErrorTypeParseError
	// ErrorTypeNoConfig 未找到网站配置（不可重试）
	ErrorTypeNoConfig
	// ErrorTypeNoContent 未找到内容（不可重试）
	ErrorTypeNoContent
)

// Error 实现 error 接口
func (e *ScrapeError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// NewScrapeError 创建新的爬虫错误
func NewScrapeError(errType ErrorType, message string, cause error) *ScrapeError {
	return &ScrapeError{
		Type:    errType,
		Message: message,
		Cause:   cause,
	}
}

// IsRetryable 判断错误是否可以重试
func (e *ScrapeError) IsRetryable() bool {
	switch e.Type {
	case ErrorTypeLoadFailed, ErrorTypeTimeout:
		return true
	default:
		return false
	}
}
