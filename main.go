package main

import (
	"context"
	"fmt"
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

	shouldReturn := LoadNovelFromCategoryChapterLink()
	if shouldReturn != nil {
		return
	}

	// shouldReturn2 := LoadNovelFromFirstChapterLink()
	// if shouldReturn2 {
	// 	return
	// }
}

// LoadNovelFromCategoryChapterLink 根据目录页，首先统计出来目录页的所有章节的链接，然后再
// 抓取每个章节的内容，最后将结果保存到文件中，这样爬取章节内容的时候，可以并发爬取
func LoadNovelFromCategoryChapterLink() error {
	// 初始化随机数种子
	rand.Seed(time.Now().UnixNano())

	// 获取目录页URL
	catalogURL := "https://www.dxmwx.org/chapter/12865.html" // 设置默认值，也可以从命令行参数获取

	// 检查是否已经爬取过这本小说
	progress, err := utils.LoadProgress(utils.GetNovelIdentifier(catalogURL))
	if err == nil && progress != nil {
		if progress.IsCompleted {
			log.Printf("检测到小说《%s》已经爬取完成\n", progress.Title)
			log.Println("已有完整内容，退出程序")
			return nil
		}
	}

	// 设置 Chrome 选项
	opts := utils.GetChromeOptions()

	// 检查 Chrome 安装
	if !utils.CheckChromeInstalled() {
		return fmt.Errorf("请先安装 Chrome 浏览器")
	}

	// 创建上下文
	rootCtx := context.Background()
	allocCtx, allocCancel := chromedp.NewExecAllocator(rootCtx, opts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
	)
	defer browserCancel()

	ctx, cancel := context.WithTimeout(browserCtx, 24*time.Hour)
	defer cancel()

	// 抓取目录
	catalog, err := scraper.ScrapeCatalog(ctx, catalogURL)
	if err != nil {
		return fmt.Errorf("获取目录失败: %v", err)
	}

	// 创建工作池
	workerCount := 5 // 同时爬取的章节数
	batchSize := 10  // 每批处理的章节数
	totalChapters := len(catalog.Chapters)

	for i := 0; i < totalChapters; i += batchSize {
		// 确定当前批次的结束索引
		end := i + batchSize
		if end > totalChapters {
			end = totalChapters
		}

		// 当前批次的章节
		currentBatch := catalog.Chapters[i:end]

		// 创建用于当前批次的通道
		chapterChan := make(chan models.ChapterInfo, len(currentBatch))
		resultChan := make(chan *models.Chapter, workerCount)
		errorChan := make(chan error, workerCount)
		doneChan := make(chan bool)

		// 启动工作协程
		for w := 0; w < workerCount; w++ {
			go func() {
				for chapter := range chapterChan {
					// 检查是否需要爬取这一章
					if progress != nil && progress.LastChapterURL == chapter.URL {
						continue
					}

					// 随机延时，避免请求过快
					time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

					// 爬取章节内容
					novel := &models.Novel{
						Title: catalog.Title,
					}
					chapterContent, err := scraper.ScrapeChapter(ctx, chapter.URL, novel)
					if err != nil {
						errorChan <- fmt.Errorf("章节 %d 爬取失败: %v", chapter.Index, err)
						continue
					}

					// 保存章节
					if err := utils.SaveChapter(chapterContent, chapter.Index); err != nil {
						errorChan <- fmt.Errorf("章节 %d 保存失败: %v", chapter.Index, err)
						continue
					}

					// 更新进度
					utils.UpdateProgress(catalog.Title, chapter.URL, chapter.Index, false)

					// 发送结果
					resultChan <- chapterContent
				}
				doneChan <- true
			}()
		}

		// 发送章节到工作池
		go func() {
			for _, chapter := range currentBatch {
				chapterChan <- chapter
			}
			close(chapterChan)
		}()

		// 等待当前批次完成
		finished := 0
		for finished < workerCount {
			select {
			case err := <-errorChan:
				log.Println("错误:", err)
			case <-resultChan:
				// 每完成一批就合并一次文件
				if err := utils.MergeChapterFiles(batchSize, catalog.Title); err != nil {
					log.Printf("合并文件失败: %v\n", err)
				}
			case <-doneChan:
				finished++
			}
		}
	}

	// 最终合并所有文件
	if err := utils.MergeChapterFiles(1, catalog.Title); err != nil {
		log.Printf("最终合并文件失败: %v\n", err)
	}

	log.Printf("爬取完成，共处理 %d 章节\n", totalChapters)

	// 标记完成状态
	utils.UpdateProgress(catalog.Title, catalogURL, totalChapters, true)
	return nil
}

// LoadNovelFromFirstChapterLink 根据起始章节的链接，抓取该章节的内容
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
