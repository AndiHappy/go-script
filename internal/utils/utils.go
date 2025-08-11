package utils

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"chromedp-scraper/internal/models"

	"github.com/chromedp/chromedp"
)

// NovelTitle 存储小说标题
var NovelTitle string = "未命名"

// SaveChapter 保存章节内容到文件
func SaveChapter(chapter *models.Chapter, num int) error {
	// 清理章节内容中的固定文本
	cleanContent := strings.ReplaceAll(chapter.Content, "本章未完，点击下一页继续阅读上一页书页目录下一页", "")
	cleanContent = strings.TrimSpace(cleanContent) // 移除可能产生的多余空行

	content := fmt.Sprintf("第%d章 %s\n\n%s\n", num, chapter.Title, cleanContent)
	filename := fmt.Sprintf("chapter_%03d.txt", num)

	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to save chapter: %v", err)
	}

	log.Printf("保存章节成功: %s\n", filename)
	return nil
}

// 常用浏览器 User-Agent 列表
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Edg/91.0.864.59",
}

// GetRandomUserAgent 返回一个随机的 User-Agent
func GetRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// GetChromeOptions 返回Chrome配置选项
func GetChromeOptions() []chromedp.ExecAllocatorOption {
	return append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.UserAgent(GetRandomUserAgent()),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("enable-javascript", true),
		chromedp.WindowSize(1920, 1080),
		chromedp.Flag("disable-gpu-compositing", true),
		chromedp.Flag("lang", "zh-CN,zh"),
	)
}

// CheckChromeInstalled 检查系统是否安装了Chrome
func CheckChromeInstalled() bool {
	paths := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",     // macOS
		"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",       // Windows
		"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe", // Windows (32-bit)
		"/usr/bin/google-chrome",        // Linux
		"/usr/bin/google-chrome-stable", // Linux
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// DownloadChrome 下载Chrome到项目目录
func DownloadChrome() error {
	return fmt.Errorf("auto download not implemented yet, please install Chrome manually")
}

// MakeAbsoluteURL 将相对URL转换为绝对URL
func MakeAbsoluteURL(href, baseURL string) string {
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "/") {
		urlParts := strings.Split(baseURL, "/")
		if len(urlParts) >= 3 {
			scheme := "http"
			if strings.Contains(baseURL, "https://") {
				scheme = "https"
			}
			return fmt.Sprintf("%s://%s%s", scheme, urlParts[2], href)
		}
		return baseURL + href
	}
	lastSlash := strings.LastIndex(baseURL, "/")
	if lastSlash != -1 {
		return baseURL[:lastSlash+1] + href
	}
	return baseURL + "/" + href
}

// MergeChapterFiles 合并章节文件
func MergeChapterFiles(batchSize int) error {
	// 获取所有章节文件
	files, err := filepath.Glob("chapter_*.txt")
	if err != nil {
		return err
	}

	// 对文件名进行排序
	sort.Strings(files)

	// 如果文件数量不足batchSize，直接返回
	if len(files) < batchSize {
		return nil
	}

	// 创建合并文件的目录
	mergedDir := "merged"
	if err := os.MkdirAll(mergedDir, 0755); err != nil {
		return err
	}

	// 定义合并文件的固定名称
	mergedFilename := filepath.Join(mergedDir, fmt.Sprintf("20020908120445-%s.txt", NovelTitle))
	var allContents []string

	// 如果合并文件已存在，先读取其内容
	if _, err := os.Stat(mergedFilename); err == nil {
		content, err := os.ReadFile(mergedFilename)
		if err != nil {
			return fmt.Errorf("读取已有合并文件失败: %v", err)
		}
		allContents = append(allContents, string(content))
	}

	// 读取所有新章节文件的内容
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			log.Printf("读取文件 %s 失败: %v\n", file, err)
			continue
		}
		allContents = append(allContents, string(content))
	}

	// 合并所有内容并写入文件
	mergedContent := strings.Join(allContents, "\n\n")
	if err := os.WriteFile(mergedFilename, []byte(mergedContent), 0644); err != nil {
		return fmt.Errorf("保存合并文件失败: %v", err)
	}

	// 删除已合并的章节文件
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			log.Printf("删除文件 %s 失败: %v\n", file, err)
		}
	}

	log.Printf("成功合并文件 %s\n", mergedFilename)
	return nil
}
