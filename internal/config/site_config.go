package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// SiteConfig 网站配置
type SiteConfig struct {
	// 网站标识
	Host string `json:"host"`
	// 网站名称
	Name string `json:"name"`
	// 小说标题选择器
	NovelTitleSelector string `json:"novelTitleSelector"`
	// 目录页章节列表选择器
	ChapterListSelector string `json:"chapterListSelector"`
	// 章节标题选择器列表
	ChapterTitleSelectors []string `json:"chapterTitleSelectors"`
	// 章节内容选择器列表
	ContentSelectors []string `json:"contentSelectors"`
	// 下一章链接选择器列表
	NextChapterSelectors []string `json:"nextChapterSelectors"`
	// 下一章链接文本关键词
	NextChapterKeywords []string `json:"nextChapterKeywords"`
}

// SitesConfig 网站配置集合
type SitesConfig struct {
	Sites map[string]*SiteConfig `json:"sites"`
}

// 全局配置变量
var (
	siteConfigs map[string]*SiteConfig
)

func init() {
	// 初始化配置
	siteConfigs = make(map[string]*SiteConfig)

	// 获取可执行文件所在目录
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("获取可执行文件路径失败: %v", err)
	}
	execDir := filepath.Dir(execPath)

	// 尝试不同的配置文件位置
	configPaths := []string{
		filepath.Join(execDir, "configs", "sites.json"),       // 可执行文件目录下的configs
		filepath.Join(execDir, "..", "configs", "sites.json"), // 上级目录的configs
		"configs/sites.json",                                  // 当前目录的configs
	}

	var loaded bool
	for _, configPath := range configPaths {
		if err := loadConfig(configPath); err == nil {
			loaded = true
			log.Printf("成功从 %s 加载网站配置\n", configPath)
			break
		}
	}

	if !loaded {
		log.Fatal("未能找到或加载网站配置文件")
	}
}

// loadConfig 加载配置文件
func loadConfig(configPath string) error {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	// 解析配置
	var config SitesConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// 更新全局配置
	siteConfigs = config.Sites
	return nil
}

// GetSiteConfig 根据URL获取网站配置
func GetSiteConfig(url string) *SiteConfig {
	for host, config := range siteConfigs {
		if strings.Contains(url, host) {
			return config
		}
	}
	return nil
}

// ReloadConfig 重新加载配置文件
func ReloadConfig(configPath string) error {
	return loadConfig(configPath)
}
