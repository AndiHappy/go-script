package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

var csvFilePath = "words.csv"

type Record struct {
	Word        string
	Explain     string
	Proficiency string
}

var title = []string{"单词", "释义", "熟练度"}

func main() {

	// debug
	//os.Args = []string{"n", "example",
	//	"Example is a noun commonly used to illustrate or demonstrate a concept, principle, or situation. For instance, in academic writing, examples help readers better understand theoretical points. If you need a more specific translation or usage context, feel free to provide additional details!",
	//	"50"}

	// 检查命令行参数
	if len(os.Args) < 4 || os.Args[1] != "n" {
		fmt.Println("使用方法: n <Word> <explain> <熟练度>")
		return
	}

	fmt.Printf("Debug: %+v \n", os.Args)

	word := os.Args[2]
	explain := os.Args[3]
	proficiency, err := strconv.Atoi(os.Args[4])
	if err != nil {
		fmt.Printf("熟练度必须是数字: %v\n", err)
		return
	}

	// 处理命令
	switch os.Args[1] {
	case "n":
		handleNoun(word, explain, proficiency)
	default:
		fmt.Println("未知命令")
	}
}

func trapBOM(fileBytes []byte) []byte {
	trimmedBytes := bytes.Trim(fileBytes, "\xef\xbb\xbf")
	return trimmedBytes
}

func trapBOMString(s string) string {
	trimmedBytes := strings.Trim(s, "\xef\xbb\xbf")
	return trimmedBytes
}

// handleNoun 处理命令行n命令
func handleNoun(word, explain string, proficiency int) {
	fmt.Printf("处理名词: %s\n解释: %s\n熟练度: %d\n", word, explain, proficiency)
	// 检查文件是否存在，不存在则创建
	file, err := os.OpenFile(csvFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Printf("打开文件失败: %v\n", err)
		return
	}
	defer file.Close()

	// 如果是新文件，写入UTF-8 BOM头
	if stat, _ := file.Stat(); stat.Size() == 0 {
		//预防写入中文,excel打开是乱码
		if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
			fmt.Printf("写入BOM头失败: %v\n", err)
			return
		}
		//写入标题
		writer := csv.NewWriter(file)
		err = writer.Write(title)
		if err != nil {
			fmt.Printf("写入标题失败: %v\n", err)
			return
		}
		writer.Flush()
	}

	// 重置文件指针到开头
	if _, err := file.Seek(0, 0); err != nil {
		fmt.Printf("重置文件指针失败: %v\n", err)
		return
	}

	// 读取现有数据
	var records []Record
	reader := csv.NewReader(file)
	for {
		row, err := reader.Read()
		if err == io.EOF {
			fmt.Println("文件读取结束")
			break
		}
		if err != nil && err != io.EOF {
			fmt.Errorf("文件读取失败:%+v,Err:%+v", file, err)
		}
		//cleanRow := trapBOMString(row[0])
		//cleanTitle := strings.TrimSpace(title[0])
		//if cleanRow == cleanTitle {
		//	continue
		//}
		// 直接的跳过头标题
		records = append(records, Record{row[0], row[1], row[2]})
	}

	// 添加新记录
	newRecord := Record{Word: word, Explain: explain, Proficiency: strconv.Itoa(proficiency)}
	records = append(records, newRecord)

	// 检查是否已存在相同的单词
	recordsMap := make(map[string]Record)
	for _, record := range records {
		recordsMap[record.Word] = record
	}

	if _, exists := recordsMap[word]; exists {
		fmt.Printf("单词 '%s' 已存在，无法重复添加。\n", word)
		record := recordsMap[word]
		record.Explain = explain
		record.Proficiency = strconv.Itoa(proficiency)
		recordsMap[word] = record
		fmt.Printf("已更新单词 '%s' 的解释和熟练度。\n", word)
	}

	// 根据Word字段排序
	sort.Slice(records, func(i, j int) bool {
		return records[i].Word > records[j].Word
	})

	//针对文件，进行覆盖性的重写
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 清空文件内容
	if err := file.Truncate(0); err != nil {
		fmt.Printf("清空文件失败: %v\n", err)
		return
	}
	if _, err := file.Seek(0, 0); err != nil {
		fmt.Printf("重置文件指针失败: %v\n", err)
		return
	}

	// 写入排序后的数据
	for _, r := range recordsMap {
		if strings.TrimSpace(r.Word) == strings.TrimSpace(title[0]) {
			continue
		}
		if err := writer.Write([]string{strings.TrimSpace(r.Word), r.Explain, r.Proficiency}); err != nil {
			fmt.Printf("写入数据失败: %v\n", err)
			return
		}
	}
	fmt.Printf("数据已排序插入: %s, %s, %d\n", word, explain, proficiency)
}
