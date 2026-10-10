//go:build windows

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/nost3a/MediaConv/internal/ffwrap"
	"github.com/nost3a/MediaConv/internal/musicdecrypt"
	"github.com/nost3a/MediaConv/internal/preset"
)

var lastOutPath string

func startGUI() {
	var mw *walk.MainWindow
	var inLabel *walk.Label
	var fmtBox *walk.ComboBox
	var qualBox *walk.ComboBox
	var progress *walk.ProgressBar
	var status *walk.Label
	var inPath string

	MainWindow{
		Title:  "MediaConv 音视频转换",
		Size:   Size{Width: 600, Height: 400},
		Layout: VBox{},
		Children: []Widget{
			Label{Text: "MediaConv 本地转码"},
			Label{AssignTo: &inLabel, Text: "未选择文件"},
			PushButton{
				Text: "选择文件",
				OnClicked: func() {
					dlg := new(walk.FileDialog)
					if ok, err := dlg.ShowOpen(mw); err == nil && ok {
						inPath = dlg.FilePath
						inLabel.SetText("输入: " + filepath.Base(inPath))
					}
				},
			},
			Label{Text: "输出格式:"},
			ComboBox{AssignTo: &fmtBox, Model: []string{"mp3", "aac", "flac", "wav", "ogg", "m4a", "opus", "mp4", "webm", "mkv"}},
			Label{Text: "音质:"},
			ComboBox{AssignTo: &qualBox, Model: []string{"128k", "192k", "320k"}},
			PushButton{
				Text: "开始转换",
				OnClicked: func() {
					if inPath == "" {
						walk.MsgBox(mw, "提示", "先选文件", walk.MsgBoxIconWarning)
						return
					}
					go func() {
						f := fmtBox.Text()
						q := qualBox.Text()
						err := doConvertWalk(inPath, f, q, progress, status)
						if err != nil {
							status.SetText("失败: " + err.Error())
						} else {
							status.SetText("完成: " + lastOutPath)
						}
					}()
				},
			},
			ProgressBar{AssignTo: &progress},
			Label{AssignTo: &status, Text: "就绪"},
		},
	}.Create()
	mw.Run()
}

func doConvertWalk(inPath, outFmt, quality string, bar *walk.ProgressBar, status *walk.Label) error {
	if musicdecrypt.IsPrivateExt(inPath) {
		dec, _, err := musicdecrypt.Decrypt(inPath)
		if err != nil {
			return err
		}
		inPath = dec
	}
	inExt := strings.TrimPrefix(filepath.Ext(inPath), ".")
	p, ok := preset.Find(inExt, outFmt, quality)
	if !ok {
		return fmt.Errorf("不支持: %s -> %s", inExt, outFmt)
	}
	dir := filepath.Dir(inPath)
	base := strings.TrimSuffix(filepath.Base(inPath), filepath.Ext(inPath))
	outPath := filepath.Join(dir, base+"."+outFmt)
	lastOutPath = outPath

	args := p.BuildArgs(quality, outPath)
	args = append([]string{"-i", inPath}, args...)

	ffBin, err := ffwrap.FindFFmpeg()
	if err != nil {
		return err
	}
	probeBin, err := ffwrap.FindFFprobe()
	if err != nil {
		return err
	}
	dur := ffwrap.ProbeDuration(probeBin, inPath)
	log.Printf("ffmpeg %v", args)

	return ffwrap.Run(context.Background(), ffBin, args, func(sec float64) {
		if dur > 0 {
			pct := int(sec / dur * 100)
			if pct > 100 {
				pct = 100
			}
			bar.SetValue(pct)
			status.SetText(fmt.Sprintf("转码中 %d%%", pct))
		}
	})
}

var _ = os.Args
