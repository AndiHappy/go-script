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
