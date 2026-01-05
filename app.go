package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"io"
	"os"
	"sync"
	"unicode/utf8"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"net/http"
	"encoding/json"
	"strings"
	"time"
)

// App struct
type App struct {
	ctx          context.Context
	filePath     string
	encoding     string
	headers      []string
	lineOffsets  []int64 // Byte offset for the start of each data row
	headerOffset int64   // Byte offset where data begins (after header)
	mu           sync.Mutex
	version      string
}

// UpdateInfo represents the update check result
type UpdateInfo struct {
	HasUpdate   bool   `json:"hasUpdate"`
	LatestVer   string `json:"latestVer"`
	CurrentVer  string `json:"currentVer"`
	DownloadURL string `json:"downloadURL"`
	Desc        string `json:"desc"`
	Error       string `json:"error"`
}

// CSVDataMetadata represents the initial file load info
type CSVDataMetadata struct {
	FilePath string   `json:"filePath"`
	Headers  []string `json:"headers"`
	Total    int      `json:"total"`
	Error    string   `json:"error"`
}

// CSVRowsResponse represents a chunk of rows
type CSVRowsResponse struct {
	Rows  [][]string `json:"rows"`
	Error string     `json:"error"`
}

// NewApp creates a new App application struct
func NewApp(version string) *App {
	return &App{
		version: version,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetVersion returns the current application version
func (a *App) GetVersion() string {
	return a.version
}

// CheckUpdate checks GitHub for the latest release
func (a *App) CheckUpdate() UpdateInfo {
	// 定义 GitHub API 地址
	repo := "TwoThreeWang/EasyCSV"
	url := "https://api.github.com/repos/" + repo + "/releases/latest"

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return UpdateInfo{Error: "网络请求失败: " + err.Error(), CurrentVer: a.version}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return UpdateInfo{Error: "无法获取版本信息 (HTTP " + resp.Status + ")", CurrentVer: a.version}
	}

	var release struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		HtmlUrl string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return UpdateInfo{Error: "解析版本信息失败", CurrentVer: a.version}
	}

	// 简单版本比较 (假设 Tag 格式为 v1.0.0)
	// 去除 'v' 前缀进行比较可能更严谨，但这里先用字符串简单判断不相等
	// 实际项目中建议引入 github.com/Masterminds/semver
	hasUpdate := release.TagName != a.version && release.TagName != "" && a.version != "v0.0.0-dev"
	
	// 如果是开发环境，永远不提示更新，或者你可以强制设为 true 来测试
	if strings.Contains(a.version, "dev") {
		hasUpdate = false
	}

	return UpdateInfo{
		HasUpdate:   hasUpdate,
		LatestVer:   release.TagName,
		CurrentVer:  a.version,
		DownloadURL: release.HtmlUrl, // 跳转到 Release 页面让用户自己下载
		Desc:        release.Body,
	}
}

// SelectFile opens a file dialog and returns the selected path
func (a *App) SelectFile() string {
	path, _ := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 CSV 文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	return path
}

// OpenCSV opens a file dialog and builds an index for the selected CSV file
func (a *App) OpenCSV() CSVDataMetadata {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 CSV 文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})

	if err != nil {
		return CSVDataMetadata{Error: "打开文件对话框失败: " + err.Error()}
	}

	if path == "" {
		return CSVDataMetadata{Error: "已取消"}
	}

	return a.LoadCSV(path)
}

// LoadCSV scans the file to build a line index for fast random access
func (a *App) LoadCSV(path string) CSVDataMetadata {
	a.mu.Lock()
	defer a.mu.Unlock()

	file, err := os.Open(path)
	if err != nil {
		return CSVDataMetadata{Error: "无法打开文件: " + err.Error()}
	}
	defer file.Close()

	// 1. Detect encoding
	encoding := a.detectEncoding(file)
	file.Seek(0, 0)

	// 2. Prepare Reader
	var reader io.Reader = file
	if encoding == "GBK" {
		reader = transform.NewReader(file, simplifiedchinese.GBK.NewDecoder())
	}
	bufReader := bufio.NewReader(reader)

	// 3. Read Header (First Line)
	// We use csv.Reader for the first line to get proper header parsing
	csvR := csv.NewReader(bufReader)
	csvR.LazyQuotes = true
	headers, err := csvR.Read()
	if err != nil {
		return CSVDataMetadata{Error: "读取表头失败: " + err.Error()}
	}

	// 4. Build Index (Scan remaining lines)
	// Since we can't easily map decoded reader offsets back to file offsets mixed with buffering,
	// we will use a simpler approach: Scan raw bytes for '\n'.
	// Limitation: This assumes 1 Row = 1 Line. (Multiline cells might break view, but is faster)
	
	// Re-open file for raw scanning to get accurate byte offsets
	file.Seek(0, 0)
	
	// Skip header line in raw scan
	// This is tricky because "header line" might be different in raw bytes vs decoded.
	// For robustness/simplicity in this "Large File Mode":
	// We will scan the whole file, store ALL line offsets.
	// Index 0 = Header, Index 1 = Row 1, etc.
	
	var offsets []int64
	var currentOffset int64 = 0
	
	// If the file is huge, this loop runs in Go, which is fast (approx 100MB/s+)
	// Reset to 0
	file.Seek(0, 0)
	
	// Create a custom scanner loop to count bytes accurately
	// bufio.Scanner doesn't give current offset easily.
	
	buf := make([]byte, 32*1024) // 32KB buffer
	offsets = append(offsets, 0) // First line starts at 0
	
	for {
		n, err := file.Read(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				if buf[i] == '\n' {
					// Found a newline, the NEXT character is the start of a new line
					offsets = append(offsets, currentOffset + int64(i) + 1)
				}
			}
			currentOffset += int64(n)
		}
		if err != nil {
			break
		}
	}

	// Verify headers match index 0
	// We already have headers from the decoded read above.
	
	// Update State
	a.filePath = path
	a.encoding = encoding
	a.headers = headers
	
	// If offsets has at least 2 entries (0: header, 1: row1...), then we have data.
	// We only care about data rows.
	if len(offsets) > 1 {
		a.lineOffsets = offsets[1:] // Skip header offset
	} else {
		a.lineOffsets = []int64{}
	}

	return CSVDataMetadata{
		FilePath: path,
		Headers:  headers,
		Total:    len(a.lineOffsets),
	}
}

// GetRows reads a specific range of rows using the index
func (a *App) GetRows(start int, limit int) CSVRowsResponse {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.filePath == "" {
		return CSVRowsResponse{Error: "没有打开的文件"}
	}

	total := len(a.lineOffsets)
	if start >= total {
		return CSVRowsResponse{Rows: [][]string{}}
	}

	end := start + limit
	if end > total {
		end = total
	}

	// Calculate bytes to read
	startOffset := a.lineOffsets[start]
	
	// Determine end offset
	// If it's the last row, read until EOF (we don't know exact length, but can read enough)
	// Actually, if we recorded offsets correctly, lineOffsets[end] (if exists) is the end.
	// But we slicing lineOffsets to exclude header.
	// lineOffsets[i] is the start of Row i.
	// To read Row i, we need to read from lineOffsets[i] to lineOffsets[i+1].
	
	// Let's simplified: We Seek to startOffset, and use a Scanner/Reader to read (end-start) lines.
	
	file, err := os.Open(a.filePath)
	if err != nil {
		return CSVRowsResponse{Error: "读取文件失败: " + err.Error()}
	}
	defer file.Close()

	file.Seek(startOffset, 0)

	var reader io.Reader = file
	if a.encoding == "GBK" {
		reader = transform.NewReader(file, simplifiedchinese.GBK.NewDecoder())
	}

	// Use csv.Reader to parse the chunk
	csvR := csv.NewReader(reader)
	csvR.LazyQuotes = true
	// Important: we might start reading in the middle of a file.
	// csv.Reader expects consistent fields.
	// Since we seeked to a newline, we are at the start of a record (assuming 1 record/line).

	var rows [][]string
	for i := 0; i < (end - start); i++ {
		record, err := csvR.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Parsing error (maybe due to multiline quote issue or encoding)
			// We return a placeholder row to avoid breaking the grid
			rows = append(rows, []string{"Error: " + err.Error()})
			continue
		}
		rows = append(rows, record)
	}

	return CSVRowsResponse{Rows: rows}
}

// SaveCSV saves the given data to a CSV file (Overwrites the whole file)
// Note: This still requires sending all data from frontend if user edited everything.
// For "Huge File" mode, usually we only save edits. But for this "EasyCSV", we stick to full save for now.
func (a *App) SaveCSV(path string, headers []string, rows [][]string) string {
	file, err := os.Create(path)
	if err != nil {
		return "无法创建文件: " + err.Error()
	}
	defer file.Close()

	// Add UTF-8 BOM for Excel compatibility
	file.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write Headers
	if err := writer.Write(headers); err != nil {
		return "写入表头失败: " + err.Error()
	}

	// Write Rows
	if err := writer.WriteAll(rows); err != nil {
		return "写入内容失败: " + err.Error()
	}

	return "" // Empty string means success
}

// detectEncoding checks if the file is UTF-8, otherwise assumes GBK (common in CN Windows)
func (a *App) detectEncoding(file *os.File) string {
	// Read first 4KB to guess
	buf := make([]byte, 4096)
	n, _ := file.Read(buf)
	buf = buf[:n]

	if utf8.Valid(buf) {
		return "UTF-8"
	}
	return "GBK"
}
