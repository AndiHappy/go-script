# Novel Scraper

这是一个使用 Go 语言编写的小说爬虫程序，使用 chromedp 实现网页内容爬取。

## 功能特点

- 使用无头浏览器（Headless Chrome）进行网页爬取
- 自动检测本地 Chrome 浏览器安装情况
- 支持自动获取下一章/页面链接
- 保存小说内容到本地文件

## 使用前提

1. 安装 Go 语言环境（推荐 1.16 或更高版本）
2. 安装 Chrome 浏览器

## 安装

```bash
git clone [your-repository-url]
cd novel-scraper
go mod download
```

## 使用方法

```bash
go run main.go <first-chapter-url>
```

例如：
```bash
go run main.go https://example.com/novel/chapter1
```

## 注意事项

1. 爬虫的选择器（例如标题、正文、下一章链接的选择器）需要根据目标网站的具体结构进行调整
2. 程序会在运行目录下创建章节文件（格式：chapter_001.txt, chapter_002.txt 等）
3. 为了避免对目标网站造成压力，程序内置了 1 秒的请求间隔

## 自定义配置

要修改网页元素的选择器，请编辑 `main.go` 文件中的 `scrapeChapter` 函数：

```go
chromedp.Text("h1", &chapter.Title),                // 修改标题选择器
chromedp.Text("div.content", &chapter.Content),     // 修改正文选择器
chromedp.AttributeValue("a.next-chapter", "href", &chapter.NextLink, nil), // 修改下一章链接选择器
```

## 许可证

MIT License
