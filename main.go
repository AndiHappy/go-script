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
	// 如果需要从命令行参数获取，可以使用 os.Args[1]
	// 例如:
	var firstChapterURL string
	if len(os.Args) > 1 {
		firstChapterURL = os.Args[1]
	}

	firstChapterURL = "https://www.drxsw.com/book/3570239/1944073676.html"
	if firstChapterURL == "" {
		log.Fatal("请提供起始章节的URL")
		return
	}

	// 检查是否已经爬取过这本小说
	progress, exists := utils.CheckNovelProgress(firstChapterURL)
	if progress != nil {
		if progress.HasError {
			log.Printf("检测到小说《%s》上次爬取出现错误\n", progress.Title)
			firstChapterURL = progress.LastChapterURL
			log.Printf("将从上次出错的位置继续: %s\n", firstChapterURL)
		} else if progress.IsCompleted && exists {
			log.Printf("检测到小说《%s》已经爬取完成\n", progress.Title)
			log.Println("已有完整内容，退出程序")
			return
		} else if exists {
			log.Printf("检测到小说《%s》有未完成的爬取进度\n", progress.Title)
			firstChapterURL = progress.LastChapterURL
			log.Printf("将从上次爬取的位置继续: %s\n", firstChapterURL)
		}
	}

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
	var chapterNum = 1
	for currentURL != "" {
		var chapter *models.Chapter
		var err error

		// 添加重试机制
		chapter, err = scraper.RetryScrapeChapter(ctx, currentURL, chapter)

		if err != nil {
			log.Printf("达到最大重试次数，放弃当前章节: %v\n", err)
			utils.UpdateProgress(chapter.Title, currentURL, chapterNum, true)
			break
		}

		// 保存章节内容
		if err := utils.SaveChapter(chapter, chapterNum); err != nil {
			log.Printf("保存章节失败: %v\n", err)
			utils.UpdateProgress(chapter.Title, currentURL, chapterNum, true)
			break
		}

		// 更新爬取进度
		if err := utils.UpdateProgress(chapter.Title, currentURL, chapterNum, false); err != nil {
			log.Printf("更新进度失败: %v\n", err)
		}

		// 每爬取10章就合并一次文件
		if err := utils.MergeChapterFiles(10); err != nil {
			log.Printf("合并文件失败: %v\n", err)
			// 标记错误状态但继续尝试
			utils.UpdateProgress(chapter.Title, currentURL, chapterNum, true)
		}

		// 更新URL到下一章
		currentURL = chapter.NextLink
		chapterNum++

		// 随机延时 1-30 微秒，避免请求过快
		sleepTime := time.Duration(1+rand.Intn(30)) * time.Microsecond
		log.Printf("等待 %v 后继续爬取下一章...\n", sleepTime)
		time.Sleep(sleepTime)
	}

	// 合并所有剩余的章节文件
	if err := utils.MergeChapterFiles(1); err != nil {
		log.Printf("最终合并文件失败: %v\n", err)
	}

	log.Printf("爬取完成，共爬取 %d 章节\n", chapterNum-1)

	// 标记完成状态
	if progress != nil {
		utils.UpdateProgress(progress.Title, currentURL, chapterNum-1, false)
	}
}
