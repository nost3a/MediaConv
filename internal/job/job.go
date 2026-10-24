package job

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nost3a/MediaConv/internal/convert"
	"github.com/nost3a/MediaConv/internal/preset"
)

type Status string

const (
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusFailed  Status = "failed"
)

type Job struct {
	ID        string    `json:"id"`
	InPath    string    `json:"-"`
	OutPath   string    `json:"-"`
	OutName   string    `json:"out_name"`
	Status    Status    `json:"status"`
	Percent   float64   `json:"percent"` // 0-100，-1 表示 indeterminate
	Duration  float64   `json:"duration"`
	Processed float64   `json:"processed_sec"`
	Err       string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Manager struct {
	mu       sync.Mutex
	jobs     map[string]*Job
	sem      chan struct{} // 并发信号量，最多 2
	workDir  string
	ffBin    string
	probeBin string
}

func NewManager(workDir, ffBin, probeBin string) *Manager {
	return &Manager{
		jobs:     map[string]*Job{},
		sem:      make(chan struct{}, 2),
		workDir:  workDir,
		ffBin:    ffBin,
		probeBin: probeBin,
	}
}

// Submit 异步起一个转码任务，返回 job_id
func (m *Manager) Submit(inPath, outFmt, quality string) (string, error) {
	if outFmt == "" {
		return "", fmt.Errorf("必须指定输出格式")
	}
	// 先校验预设组合是否存在，提前给出明确错误
	inExt := extOf(inPath)
	if _, ok := preset.Find(inExt, outFmt); !ok {
		return "", fmt.Errorf("不支持的组合: %s -> %s", inExt, outFmt)
	}

	id := fmt.Sprintf("job_%d", time.Now().UnixNano())
	outName := id + "." + outFmt
	outPath := filepath.Join(m.workDir, outName)

	j := &Job{
		ID:        id,
		InPath:    inPath,
		OutPath:   outPath,
		OutName:   outName,
		Status:    StatusRunning,
		Percent:   -1,
		CreatedAt: time.Now(),
	}
	m.mu.Lock()
	m.jobs[id] = j
	m.mu.Unlock()

	opts := convert.Options{
		InPath:     inPath,
		OutFmt:     outFmt,
		OutPath:    outPath,
		Quality:    quality,
		FFmpegBin:  m.ffBin,
		FFprobeBin: m.probeBin,
		OnProgress: func(sec, total float64) {
			m.mu.Lock()
			j.Processed = sec
			j.Duration = total
			if total > 0 {
				p := sec / total * 100
				if p > 99.5 {
					p = 99.5
				}
				j.Percent = p
			}
			m.mu.Unlock()
		},
	}
	go m.run(j, opts)
	return id, nil
}

func (m *Manager) run(j *Job, opts convert.Options) {
	m.sem <- struct{}{}
	defer func() { <-m.sem }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	outPath, err := convert.Convert(ctx, opts)

	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		j.Status = StatusFailed
		j.Err = err.Error()
		// 清理失败产物（convert 已删；这里再删一次幂等）
		os.Remove(j.OutPath)
		log.Printf("job %s failed: %v", j.ID, err)
		return
	}
	j.OutPath = outPath
	j.OutName = filepath.Base(outPath)
	j.Status = StatusDone
	j.Percent = 100
}

func (m *Manager) Get(id string) (*Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	if !ok {
		return nil, false
	}
	cp := *j
	return &cp, true
}

// CleanupDone 定期清理完成超过 10 分钟的任务（仅清产物，不动原始输入）
func (m *Manager) CleanupDone() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, j := range m.jobs {
		if j.Status == StatusDone || j.Status == StatusFailed {
			if time.Since(j.CreatedAt) > 10*time.Minute {
				os.Remove(j.OutPath)
				delete(m.jobs, id)
			}
		}
	}
}

func extOf(p string) string {
	ext := filepath.Ext(p)
	if len(ext) > 1 {
		return ext[1:]
	}
	return ""
}
