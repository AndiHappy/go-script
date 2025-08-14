package utils

import (
	"encoding/json"
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

func ToString(v any) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		log.Fatal("json Marshal Err: ", err)
		return ""
	}
	return string(bytes)
}

// SaveChapter 保存章节内容到文件
func SaveChapter(chapter *models.Chapter, num int) error {
	// 清理章节内容中的固定文本
	cleanContent := strings.ReplaceAll(chapter.Content, "本章未完，点击下一页继续阅读上一页书页目录下一页", "")
	cleanContent = strings.TrimSpace(cleanContent) // 移除可能产生的多余空行

	content := fmt.Sprintf("第%d章 %s\n\n%s\n", num, chapter.Title, cleanContent)
	filename := fmt.Sprintf("chapter_%04d.txt", num)

	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to save chapter: %v", err)
	}

	log.Printf("保存章节成功: %s\n", filename)
	return nil
}

// 常用浏览器 User-Agent 列表
var userAgents = []string{
	// Chrome for Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36",

	// Chrome for MacOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36",

	// Firefox for Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/116.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/115.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/114.0",

	// Firefox for MacOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13.5; rv:109.0) Gecko/20100101 Firefox/116.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13.4; rv:109.0) Gecko/20100101 Firefox/115.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13.3; rv:109.0) Gecko/20100101 Firefox/114.0",

	// Safari for MacOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5.2 Safari/605.1.15",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Safari/605.1.15",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_3) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.4 Safari/605.1.15",

	// Edge for Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36 Edg/115.0.1901.200",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36 Edg/114.0.1823.82",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36 Edg/113.0.1774.57",

	// Edge for MacOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36 Edg/115.0.1901.200",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36 Edg/114.0.1823.82",

	// 移动浏览器 UA
	"Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPad; CPU OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/115.0.5790.130 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 13; SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Mobile Safari/537.36",
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

// MergeChaptersToFile 合并catalog中所有章节的Content内容，并写入指定文件
// 参数：
//
//	catalog: 包含章节列表的目录结构体
//	filePath: 目标文件路径（如 "./output.txt"）
//
// 返回：
//
//	error: 若过程中出现错误（如目录为空、文件写入失败等），返回具体错误信息；成功则返回nil
func MergeChaptersToFile(catalog *models.Catalog, filePath string) error {
	// 1. 检查入参合法性
	if catalog == nil {
		return nil // 或返回错误：fmt.Errorf("catalog不能为空")
	}
	if len(catalog.Chapters) == 0 {
		return nil // 或返回错误：fmt.Errorf("章节列表为空，无需合并")
	}

	// 2. 初始化字符串构建器，高效拼接大量字符串
	var contentBuilder strings.Builder

	// 3. 遍历所有章节，合并内容
	for i, chapter := range catalog.Chapters {
		// 写入当前章节内容
		contentBuilder.WriteString(chapter.ChapterContent)

		// 章节间添加分隔符（如换行），最后一章不需要
		if i != len(catalog.Chapters)-1 {
			contentBuilder.WriteString("\n\n") // 用两个换行分隔章节，增强可读性
		}
	}

	// 4. 将合并后的内容写入文件
	// 0644 表示文件权限：所有者可读写，其他用户可读
	return os.WriteFile(filePath, []byte(contentBuilder.String()), 0644)
}

// MergeChapterFiles 合并章节文件
func MergeChapterFiles(batchSize int, title string) error {
	// 获取所有章节文件
	files, err := filepath.Glob("chapter_????.txt")
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
	mergedFilename := filepath.Join(mergedDir, fmt.Sprintf("20020908120445-%s.txt", title))
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
