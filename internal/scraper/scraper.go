package scraper

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"chromedp-scraper/internal/models"
	"chromedp-scraper/internal/utils"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

// ScrapeChapter 爬取单个章节的内容
func ScrapeChapter(ctx context.Context, url string) (*models.Chapter, error) {
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
	log.Println("等待页面加载...")
	err := chromedp.Run(taskCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(1*time.Second), // 减少等待时间
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)
	if err != nil {
		return nil, fmt.Errorf("页面加载失败: %v", err)
	}

	// 使用 goquery 解析 HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("解析HTML失败: %v", err)
	}

	// 检查并设置小说标题
	if utils.NovelTitle == "未命名" {
		novelTitle := doc.Find("#wrapper > article > div.con_top > a:nth-child(2)").Text()
		if novelTitle != "" {
			utils.NovelTitle = strings.TrimSpace(novelTitle)
			log.Printf("设置小说标题: %s\n", utils.NovelTitle)
		}
	}

	// 获取标题
	log.Println("正在获取标题...")
	titleSelectors := []string{"#chaptername", "h1", ".chapter-title"}
	for _, selector := range titleSelectors {
		if title := doc.Find(selector).First().Text(); title != "" {
			chapter.Title = strings.TrimSpace(title)
			break
		}
	}
	if chapter.Title == "" {
		return nil, fmt.Errorf("未找到标题")
	}
	log.Printf("成功获取标题: %s\n", chapter.Title)

	// 获取内容
	log.Println("正在获取正文内容...")
	contentSelectors := []string{"#chaptercontent", ".chapter-content", "#content"}
	var content string
	for _, selector := range contentSelectors {
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
		return nil, fmt.Errorf("未找到正文内容")
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

	// 1. 通过 ID 查找
	idSelectors := []string{"#next", "#nextChapter", "#next_chapter"}
	for _, id := range idSelectors {
		if href, exists := doc.Find(id).First().Attr("href"); exists {
			chapter.NextLink = utils.MakeAbsoluteURL(href, baseURL)
			break
		}
	}

	// 2. 通过文本内容查找
	if chapter.NextLink == "" {
		keywords := []string{"下一章", "下一页", "下页", "后一章", "下一节"}
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			for _, keyword := range keywords {
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
		return nil, fmt.Errorf("failed to scrape chapter: empty content")
	}

	return &chapter, nil
}
