package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"chromedp-scraper/internal/models"
	"chromedp-scraper/internal/scraper"
	"chromedp-scraper/internal/utils"

	"github.com/chromedp/chromedp"
)

func main() {
	// 初始化随机数种子
	rand.Seed(time.Now().UnixNano())

	// 获取os.arg的参数
	// 这里可以根据需要修改为从命令行参数获取起始章节URL
	// firstChapterURL := "https://www.3378.org/read/28595/8108534.html"
	// 如果需要从命令行参数获取，可以使用 os.Args[1]
	// 例如:
	var firstChapterURL string
	if len(os.Args) > 1 {
		firstChapterURL = os.Args[1]
	}

	if firstChapterURL == "" {
		log.Fatal("请提供起始章节的URL")
		return
	}

	// firstChapterURL := "https://www.3378.org/read/28595/8108534.html"

	// 设置 Chrome 选项
	opts := utils.GetChromeOptions()

	// 检查本地是否有 Chrome，如果没有则下载到项目目录
	if !utils.CheckChromeInstalled() {
		log.Println("Chrome not found, downloading...")
		if err := utils.DownloadChrome(); err != nil {
			log.Fatal("Failed to download Chrome:", err)
		}
		// 添加自定义 Chrome 路径
		opts = append(opts, chromedp.ExecPath(filepath.Join(".", "chrome-linux", "chrome")))
	}

	// 创建一个根上下文
	rootCtx := context.Background()

	// 创建浏览器实例
	allocCtx, allocCancel := chromedp.NewExecAllocator(rootCtx, opts...)
	defer allocCancel()

	// 创建浏览器上下文
	browserCtx, browserCancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf), // 添加日志记录
	)
	defer browserCancel()

	// 设置全局超时
	ctx, cancel := context.WithTimeout(browserCtx, 24*time.Hour) // 设置一个较长的全局超时
	defer cancel()

	// 开始爬取章节
	currentURL := firstChapterURL
	chapterNum := 1
	maxRetries := 3               // 最大重试次数
	retryDelay := 2 * time.Second // 减少重试等待时间

	for currentURL != "" {
		var chapter *models.Chapter
		var err error

		// 添加重试机制
		for retry := 0; retry < maxRetries; retry++ {
			if retry > 0 {
				log.Printf("第 %d 次重试爬取页面...\n", retry+1)
				waitTime := retryDelay * time.Duration(retry+1) // 线性增加等待时间
				log.Printf("等待时间增加到: %v\n", waitTime)
				time.Sleep(waitTime)
			}

			chapter, err = scraper.ScrapeChapter(ctx, currentURL)
			if err == nil {
				break // 成功获取，退出重试循环
			}

			log.Printf("爬取失败: %v\n", err)
		}

		if err != nil {
			log.Printf("达到最大重试次数，放弃当前章节: %v\n", err)
			break
		}

		// 保存章节内容
		if err := utils.SaveChapter(chapter, chapterNum); err != nil {
			log.Printf("保存章节失败: %v\n", err)
			break
		}

		// 每爬取10章就合并一次文件
		if err := utils.MergeChapterFiles(10); err != nil {
			log.Printf("合并文件失败: %v\n", err)
		}

		// 更新URL到下一章
		currentURL = chapter.NextLink
		chapterNum++

		// 随机延时 1-30 微秒，避免请求过快
		sleepTime := time.Duration(1+rand.Intn(30)) * time.Microsecond
		log.Printf("等待 %v 后继续爬取下一章...\n", sleepTime)
		time.Sleep(sleepTime)
	}

	log.Printf("爬取完成，共爬取 %d 章节\n", chapterNum-1)
}
