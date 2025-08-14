package models

// Novel 结构体用于存储小说信息
type Novel struct {
	Title    string
	Author   string
	Chapters []*Chapter
}

// Chapter 结构体用于存储小说章节信息
type Chapter struct {
	Title    string
	Content  string
	NextLink string
}

// ChapterInfo 结构体用于存储目录页面的章节信息
type ChapterInfo struct {
	Index          int    // 章节序号
	Title          string // 章节标题
	URL            string // 章节链接
	ChapterContent string //章节内容
}

// Catalog 结构体用于存储目录信息
type Catalog struct {
	Title    string //整部小说的标题
	Chapters []ChapterInfo
}
