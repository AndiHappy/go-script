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
	shouldReturn := LoadNovelFromFirstChapterLink()
	if shouldReturn {
		return
	}
}

func LoadNovelFromFirstChapterLink() bool {
	var firstChapterURL string
	var startChapterNum int
	var novel *models.Novel
	if len(os.Args) > 1 {
		firstChapterURL = os.Args[1]
	}

	firstChapterURL = "https://www.drxsw.com/book/3570239/1944073676.html"
	startChapterNum = 1
	novel = &models.Novel{
		Title:    "未命名",
		Author:   "未知",
		Chapters: []*models.Chapter{},
	}
	if firstChapterURL == "" {
		log.Fatal("请提供起始章节的URL")
		return true
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
	var chapterNum = startChapterNum
	var chapter *models.Chapter
	// 抓取第一个章节，拿到初始化的信息
	chapter, err := scraper.RetryScrapeChapter(ctx, currentURL, chapter, novel)
	if err != nil {
		log.Printf("达到最大重试次数，放弃当前章节: %v\n", err)
		return true
	}

	// 检查是否已经爬取过这本小说
	progress, exists := utils.CheckNovelProgress(novel)
	if progress != nil {
		if progress.HasError {
			log.Printf("检测到小说《%s》上次爬取出现错误\n", progress.Title)
			firstChapterURL = progress.LastChapterURL
			chapterNum = progress.LastChapterNum + 1
			log.Printf("将从上次出错的位置继续: %s\n", firstChapterURL)
			// 抓取第一个章节，拿到初始化的信息
			chapter, err = scraper.RetryScrapeChapter(ctx, firstChapterURL, chapter, novel)
			if err != nil {
				log.Printf("达到最大重试次数，放弃当前章节: %v\n", err)
				return true
			}
		} else if progress.IsCompleted && exists {
			log.Printf("检测到小说《%s》已经爬取完成\n", progress.Title)
			log.Println("已有完整内容，退出程序")
			return true
		} else if exists {
			log.Printf("检测到小说《%s》有未完成的爬取进度\n", progress.Title)
			firstChapterURL = progress.LastChapterURL
			chapterNum = progress.LastChapterNum + 1
			log.Printf("将从上次爬取的位置继续: %s\n", firstChapterURL)
			// 抓取第一个章节，拿到初始化的信息
			chapter, err = scraper.RetryScrapeChapter(ctx, firstChapterURL, chapter, novel)
			if err != nil {
				log.Printf("达到最大重试次数，放弃当前章节: %v\n", err)
				return true
			}
		}
	}

	// 循环的向后迭代
	for chapter != nil && currentURL != "" {
		var err error
		novel.Chapters = append(novel.Chapters, chapter)
		// 保存章节内容
		if err := utils.SaveChapter(chapter, chapterNum); err != nil {
			log.Printf("保存章节失败: %v\n", err)
			utils.UpdateProgress(novel.Title, chapter.NextLink, chapterNum, true)
			break
		}
		// 更新爬取进度
		if err := utils.UpdateProgress(novel.Title, chapter.NextLink, chapterNum, false); err != nil {
			log.Printf("更新进度失败: %v\n", err)
		}

		// 每爬取10章就合并一次文件
		if err := utils.MergeChapterFiles(10, novel.Title); err != nil {
			log.Printf("合并文件失败: %v\n", err)
			// 标记错误状态但继续尝试
			utils.UpdateProgress(novel.Title, chapter.NextLink, chapterNum, true)
		}

		// 更新URL到下一章
		currentURL = chapter.NextLink
		chapterNum++
		if currentURL != "" {
			// 随机延时 1-30 微秒，避免请求过快
			sleepTime := time.Duration(1+rand.Intn(30)) * time.Microsecond
			log.Printf("等待 %v 后继续爬取下一章...\n", sleepTime)
			time.Sleep(sleepTime)
			// 添加重试机制
			chapter, err = scraper.RetryScrapeChapter(ctx, currentURL, chapter, novel)
			if err != nil {
				log.Printf("达到最大重试次数，放弃当前章节: %v\n", err)
				break
			}
		}
	}

	// 合并所有剩余的章节文件
	if err := utils.MergeChapterFiles(1, novel.Title); err != nil {
		log.Printf("最终合并文件失败: %v\n", err)
	}

	log.Printf("爬取完成，共爬取 %d 章节\n", chapterNum-1)

	// 标记完成状态
	if progress != nil {
		utils.UpdateProgress(progress.Title, currentURL, chapterNum, false)
	}
	return false
}
