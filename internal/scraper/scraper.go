package scraper

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"chromedp-scraper/internal/config"
	"chromedp-scraper/internal/models"
	"chromedp-scraper/internal/utils"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

// ScrapeCatalog 爬取小说目录页面
func ScrapeCatalog(ctx context.Context, u string) (*models.Catalog, error) {
	// 获取网站配置
	siteConfig := config.GetSiteConfig(u)
	if siteConfig == nil {
		return nil, NewScrapeError(ErrorTypeNoConfig, "未找到网站配置", nil)
	}

	var html string
	timeS := time.Now() // 记录开始时间

	// 加载页面
	if err := chromedp.Run(ctx,
		chromedp.Navigate(u),
		chromedp.OuterHTML("html", &html),
	); err != nil {
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
			return nil, NewScrapeError(ErrorTypeTimeout, "页面加载超时", err)
		}
		return nil, NewScrapeError(ErrorTypeLoadFailed, "页面加载失败", err)
	}

	// 解析HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, NewScrapeError(ErrorTypeParseError, "解析HTML失败", err)
	}

	log.Println("目录页面加载解析完成,耗时:", time.Since(timeS).Seconds(), "秒")

	// 创建目录对象
	catalog := &models.Catalog{}
	// 获取章节标题
	log.Println("正在获取标题...")
	for _, selector := range siteConfig.NovelTitleSelectors {
		if title := doc.Find(selector).First().Text(); title != "" {
			catalog.Title = strings.TrimSpace(title)
			break
		}
	}
	catalog.Title = strings.TrimSpace(catalog.Title)

	// 获取章节列表
	chapters := make([]models.ChapterInfo, 0)
	log.Println("正在获取章节列表...")
	for _, selector := range siteConfig.ChapterListSelectors {
		doc.Find(selector).Find("a").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if !exists {
				return
			}

			title := strings.TrimSpace(s.Text())
			if title == "" {
				return
			}

			if strings.Contains(title, "第") { //这种确定链接的方式，有点low，后续进行选择性的抓取
				chapters = append(chapters, models.ChapterInfo{
					Index: i + 1,
					Title: title,
					URL:   utils.MakeAbsoluteURL(href, u),
				})
			} else {
				log.Println("跳过该a: ", s.Text(), href)
			}
		})
	}

	if len(chapters) == 0 {
		return nil, NewScrapeError(ErrorTypeNoContent, "未找到章节列表", nil)
	}

	catalog.Chapters = chapters
	log.Printf("成功获取目录，共 %d 章\n", len(chapters))

	return catalog, nil
}

var maxRetries = 3               // 最大重试次数
var retryDelay = 1 * time.Second // 减少重试等待时间

// RetryScrapeChapter 带重试机制的章节爬取
func RetryScrapeChapter(ctx context.Context, currentURL string,
	chapter *models.Chapter, novel *models.Novel) (*models.Chapter, error) {
	var lastError error
	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			// 检查上一次错误是否可重试
			if scrapeErr, ok := lastError.(*ScrapeError); ok && !scrapeErr.IsRetryable() {
				log.Printf("错误不可重试，放弃后续尝试: %v\n", lastError)
				return nil, lastError
			}
			log.Printf("第 %d 次重试爬取页面...\n", retry+1)
			waitTime := retryDelay * time.Duration(retry+1) // 线性增加等待时间
			log.Printf("等待时间增加到: %v\n", waitTime)
			time.Sleep(waitTime)
		}

		chapter, err := ScrapeChapter(ctx, currentURL, novel)
		if err == nil {
			return chapter, nil
		}

		lastError = err
		if scrapeErr, ok := err.(*ScrapeError); ok {
			log.Printf("爬取失败: [%v] %s\n", scrapeErr.Type, scrapeErr.Error())
			if !scrapeErr.IsRetryable() {
				return nil, err
			}
		} else {
			log.Printf("爬取失败（未知错误）: %v\n", err)
		}
	}
	return nil, lastError
}

// ScrapeChapter 爬取单个章节的内容
func ScrapeChapter(ctx context.Context, url string, novel *models.Novel) (*models.Chapter, error) {
	var chapter models.Chapter

	log.Printf("开始爬取页面: %s\n", url)
	log.Printf("使用 User-Agent: %s\n", utils.GetRandomUserAgent())

	// 为整个抓取过程创建一个超时上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second) // 减少超时时间
	defer cancel()

	// 创建一个新的 tab
	taskCtx, cancel := chromedp.NewContext(timeoutCtx)
	defer cancel()

	// 获取页面 HTML
	var html string
	timeS := time.Now() // 记录开始时间
	log.Println("等待页面加载...")
	err := chromedp.Run(taskCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)
	if err != nil {
		// 检查是否为超时错误
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
			return nil, NewScrapeError(ErrorTypeTimeout, "页面加载超时", err)
		}
		return nil, NewScrapeError(ErrorTypeLoadFailed, "页面加载失败", err)
	}

	// 使用 goquery 解析 HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, NewScrapeError(ErrorTypeParseError, "解析HTML失败", err)
	}
	log.Println("页面加载解析完成,耗时:", time.Since(timeS).Seconds(), "秒")

	// 获取网站配置
	siteConfig := config.GetSiteConfig(url)
	if siteConfig == nil {
		return nil, NewScrapeError(ErrorTypeNoConfig, "未找到网站配置", nil)
	}

	// 检查并设置小说标题
	if novel.Title == "未命名" {
		log.Println("正在获取小说标题...")
		for _, selector := range siteConfig.NovelTitleSelectors {
			if title := doc.Find(selector).First().Text(); title != "" {
				novel.Title = strings.TrimSpace(title)
				log.Printf("设置小说标题: %s\n", novel.Title)
				break
			}
		}
	}

	// 获取章节标题
	log.Println("正在获取标题...")
	for _, selector := range siteConfig.ChapterTitleSelectors {
		if title := doc.Find(selector).First().Text(); title != "" {
			chapter.Title = strings.TrimSpace(title)
			break
		}
	}
	if chapter.Title == "" {
		return nil, NewScrapeError(ErrorTypeNoContent, "未找到章节标题", nil)
	}
	log.Printf("成功获取标题: %s\n", chapter.Title)

	// 获取内容
	log.Println("正在获取正文内容...")
	var content string
	for _, selector := range siteConfig.ContentSelectors {
		if contentEl := doc.Find(selector).First(); contentEl.Length() > 0 {
			// 移除所有 script 标签
			contentEl.Find("script").Remove()

			// 获取所有文本节点和段落的内容
			var paragraphs []string
			contentEl.Contents().Each(func(i int, s *goquery.Selection) {
				if s.Is("p") || goquery.NodeName(s) == "#text" {
					if text := strings.TrimSpace(s.Text()); text != "" {
						paragraphs = append(paragraphs, text)
					}
				}
			})

			if len(paragraphs) > 0 {
				content = strings.Join(paragraphs, "\n\n")
				break
			}

			// 如果没有找到有效的段落，使用完整文本
			content = strings.TrimSpace(contentEl.Text())
			break
		}
	}
	if content == "" {
		return nil, NewScrapeError(ErrorTypeNoContent, "未找到正文内容", nil)
	}
	chapter.Content = content
	log.Printf("成功获取正文，长度: %d 字符\n", len(chapter.Content))

	// 获取下一章链接
	log.Println("正在获取下一章链接...")

	// 解析当前页面的URL，用于后面构建绝对路径
	baseURL := url // 默认使用当前页面URL
	if href, exists := doc.Find("base[href]").First().Attr("href"); exists {
		baseURL = href
	}

	// 1. 通过选择器查找
	for _, selector := range siteConfig.NextChapterSelectors {
		if href, exists := doc.Find(selector).First().Attr("href"); exists {
			chapter.NextLink = utils.MakeAbsoluteURL(href, baseURL)
			break
		}
	}

	// 2. 通过文本内容查找
	if chapter.NextLink == "" {
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			for _, keyword := range siteConfig.NextChapterKeywords {
				if strings.Contains(text, keyword) {
					if href, exists := s.Attr("href"); exists {
						chapter.NextLink = utils.MakeAbsoluteURL(href, baseURL)
						return
					}
				}
			}
		})
	}

	// 3. 通过 URL 模式匹配
	if chapter.NextLink == "" {
		currentPath := strings.TrimPrefix(url, "https://")
		currentPath = strings.TrimPrefix(currentPath, "http://")
		if idx := strings.Index(currentPath, "/"); idx != -1 {
			currentPath = currentPath[idx:]
		}
		if match := strings.Split(currentPath, ".html")[0]; match != "" {
			parts := strings.Split(match, "_")
			if len(parts) > 0 {
				nextPath := fmt.Sprintf("%s_1.html", parts[0])
				chapter.NextLink = fmt.Sprintf("%s%s", strings.Split(url, currentPath)[0], nextPath)
			}
		}
	}

	log.Printf("获取到下一章链接: %s\n", chapter.NextLink)

	// 清理内容
	chapter.Title = strings.TrimSpace(chapter.Title)
	chapter.Content = strings.TrimSpace(chapter.Content)

	// 如果下一章链接是 JavaScript:void(0) 或类似的，将其设置为空
	if strings.Contains(chapter.NextLink, "javascript:") || chapter.NextLink == "" {
		chapter.NextLink = ""
	}

	// 确保内容不为空
	if chapter.Title == "" || chapter.Content == "" {
		return nil, NewScrapeError(ErrorTypeNoContent, "章节内容或标题为空", nil)
	}

	return &chapter, nil
}
