package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"chromedp-scraper/internal/models"
)

var (
	UpdateProgressMutex sync.Mutex // 保护更新进度的并发访问
)

const progressDir = "progress"
const progressExt = ".progress.json"

// GetNovelIdentifier 从URL中提取小说标识
func GetNovelIdentifier(url string) string {
	// 移除协议部分
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")

	// 提取域名和小说ID
	parts := strings.Split(url, "/")
	if len(parts) >= 4 && parts[1] == "book" {
		// 网站域名和小说ID，例如: www.drxsw.com-3570239
		return parts[0] + "-" + parts[2]
	}
	return url
}

// // extractNumbers 提取字符串中的数字
// func extractNumbers(s string) string {
// 	var numbers []rune
// 	for _, r := range s {
// 		if r >= '0' && r <= '9' {
// 			numbers = append(numbers, r)
// 		}
// 	}
// 	if len(numbers) > 0 {
// 		return string(numbers)
// 	}
// 	return ""
// }

// SaveProgress 保存爬取进度
func SaveProgress(progress *models.NovelProgress) error {

	// 创建进度目录
	if err := os.MkdirAll(progressDir, 0755); err != nil {
		return err
	}

	// 构建进度文件路径
	filename := progress.URLIdentifier + progressExt
	filepath := filepath.Join(progressDir, filename)

	// 如果文件已存在，读取现有进度
	var existingProgress *models.NovelProgress
	if _, err := os.Stat(filepath); err == nil {
		data, err := os.ReadFile(filepath)
		if err == nil {
			var existing models.NovelProgress
			if json.Unmarshal(data, &existing) == nil {
				existingProgress = &existing
			}
		}
	}

	// 合并进度信息
	if existingProgress != nil && existingProgress.LastChapterNum > progress.LastChapterNum {
		// 如果现有进度的章节号更大，保留现有进度
		return nil
	}

	// 序列化进度数据
	data, err := json.MarshalIndent(progress, "", "    ")
	if err != nil {
		return err
	}

	// 覆盖性的写入文件
	return os.WriteFile(filepath, data, 0644)
}

// LoadProgress 加载爬取进度
func LoadProgress(urlIdentifier string) (*models.NovelProgress, error) {
	filename := urlIdentifier + progressExt
	filepath := filepath.Join(progressDir, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, nil
	}

	// 读取文件内容
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	// 反序列化
	var progress models.NovelProgress
	if err := json.Unmarshal(data, &progress); err != nil {
		return nil, err
	}

	return &progress, nil
}

// UpdateProgress 更新爬取进度
func UpdateProgress(title, url string, chapterNum int, hasError bool) error {
	UpdateProgressMutex.Lock()
	defer UpdateProgressMutex.Unlock()

	urlIdentifier := GetNovelIdentifier(url)

	// 尝试加载现有进度
	progress, err := LoadProgress(urlIdentifier)
	if err != nil {
		return err
	}

	// 如果不存在则创建新的进度记录
	if progress == nil {
		progress = &models.NovelProgress{
			Title:         title,
			URLIdentifier: urlIdentifier,
		}
	}

	// 更新进度信息
	progress.LastChapterNum = chapterNum
	progress.LastChapterURL = url
	progress.TotalChapters = chapterNum
	progress.LastUpdateTime = time.Now().Unix()

	// 保存更新后的进度
	return SaveProgress(progress)
}

// CheckNovelProgress 检查小说爬取进度
func CheckNovelProgress(url string) (*models.NovelProgress, bool) {
	urlIdentifier := GetNovelIdentifier(url)

	// 加载进度
	progress, err := LoadProgress(urlIdentifier)
	if err != nil {
		return nil, false
	}

	// 检查是否存在进度文件
	if progress == nil {
		return nil, false
	}

	// 只需要检查进度文件中的状态
	if progress.IsCompleted {
		// 如果进度文件标记为已完成
		return progress, true
	} else if progress.LastChapterNum > 0 {
		// 如果有未完成的进度（至少爬取过一章）
		return progress, true
	}

	return progress, false
}
