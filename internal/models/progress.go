package models

// NovelProgress 小说爬取进度
type NovelProgress struct {
	// 小说标题
	Title string `json:"title"`
	// 最后爬取的章节编号
	LastChapterNum int `json:"lastChapter"`
	// 最后爬取的章节URL
	LastChapterURL string `json:"lastChapterUrl"`
	// 总章节数
	TotalChapters int `json:"totalChapters"`
	// 最后更新时间
	LastUpdateTime int64 `json:"lastUpdateTime"`
	// 是否有错误发生
	HasError bool `json:"hasError"`
	// 是否已完成
	IsCompleted bool `json:"isCompleted"`
}
