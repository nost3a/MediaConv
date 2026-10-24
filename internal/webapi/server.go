// Package webapi 提供无桌面的本地 Web 转码服务。
// 把原本散落在 main 里的 HTTP 处理抽出来，供 CLI 的 `start` 命令调用。
package webapi

import (
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nost3a/MediaConv/internal/ffwrap"
	"github.com/nost3a/MediaConv/internal/job"
	"github.com/nost3a/MediaConv/internal/preset"
)

const maxUploadSize = 500 << 20 // 500MB 硬上限

// Server 持有 Web 服务依赖
type Server struct {
	webFS     fs.FS
	mgr       *job.Manager
	ffBin     string
	probeBin  string
	uploadDir string
}

// Serve 启动一个仅绑定回环地址的本地 Web 转码服务，阻塞直到退出。
func Serve(webFS fs.FS, host, port, ffBin, probeBin string) error {
	workDir, err := os.MkdirTemp("", "mediaconv-web-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	s := &Server{
		webFS:     webFS,
		mgr:       job.NewManager(workDir, ffBin, probeBin),
		ffBin:     ffBin,
		probeBin:  probeBin,
		uploadDir: workDir,
	}
	log.Printf("使用 ffmpeg: %s", ffBin)
	log.Printf("使用 ffprobe: %s", probeBin)

	go func() {
		t := time.NewTicker(1 * time.Minute)
		for range t.C {
			s.mgr.CleanupDone()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/upload", s.handleUpload)
	mux.HandleFunc("/api/convert", s.handleConvert)
	mux.HandleFunc("/api/progress/", s.handleProgress)
	mux.HandleFunc("/api/download/", s.handleDownload)
	mux.HandleFunc("/api/formats", s.handleFormats)
	mux.HandleFunc("/api/info", s.handleInfo)

	if webFS != nil {
		if sub, err := fs.Sub(webFS, "web"); err == nil {
			mux.Handle("/", http.FileServer(http.FS(sub)))
		}
	}

	addr := net.JoinHostPort(host, port)
	log.Printf("MediaConv Web 服务: http://%s", addr)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("监听 %s 失败: %w", addr, err)
	}
	return http.Serve(ln, mux)
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	// 硬性大小限制，防滥用
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "解析表单失败（可能超过 500MB 上限）: "+err.Error(), 400)
		return
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "缺少 file 字段", 400)
		return
	}
	defer f.Close()

	id := fmt.Sprintf("up_%d", time.Now().UnixNano())
	safeName := filepath.Base(hdr.Filename)
	inPath := filepath.Join(s.uploadDir, id+"_"+safeName)
	out, err := os.Create(inPath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if _, err := copyTo(out, f); err != nil {
		out.Close()
		http.Error(w, err.Error(), 500)
		return
	}
	out.Close()

	writeJSON(w, map[string]string{
		"file_id": inPath,
		"name":    safeName,
		"size":    fmt.Sprintf("%d", hdr.Size),
	})
}

type convertReq struct {
	FileID       string `json:"file_id"`
	TargetFormat string `json:"target_format"`
	Quality      string `json:"quality"`
}

func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	var req convertReq
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "JSON 解析失败: "+err.Error(), 400)
		return
	}
	if req.FileID == "" || req.TargetFormat == "" {
		http.Error(w, "file_id / target_format 必填", 400)
		return
	}
	if _, err := os.Stat(req.FileID); err != nil {
		http.Error(w, "输入文件不存在", 400)
		return
	}
	jobID, err := s.mgr.Submit(req.FileID, req.TargetFormat, req.Quality)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	writeJSON(w, map[string]string{"job_id": jobID})
}

func (s *Server) handleProgress(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/progress/")
	j, ok := s.mgr.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, j)
}

// handleDownload 流式返回成品，避免整文件读进内存
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/download/")
	j, ok := s.mgr.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if j.Status != job.StatusDone {
		http.Error(w, "任务未完成", 409)
		return
	}
	ext := filepath.Ext(j.OutName)
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="convert%s"`, ext))
	http.ServeFile(w, r, j.OutPath)
}

func (s *Server) handleFormats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, preset.ListFormats())
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	var req struct {
		FileID string `json:"file_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "JSON 解析失败: "+err.Error(), 400)
		return
	}
	if req.FileID == "" {
		http.Error(w, "file_id 必填", 400)
		return
	}
	if _, err := os.Stat(req.FileID); err != nil {
		http.Error(w, "输入文件不存在", 400)
		return
	}
	info, err := ffwrap.Probe(s.probeBin, req.FileID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, info)
}
