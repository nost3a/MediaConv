//go:build windows

package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/nost3a/MediaConv/internal/convert"
	"github.com/nost3a/MediaConv/internal/ffwrap"
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
	ffBin, err := ffwrap.FindFFmpeg()
	if err != nil {
		return err
	}
	probeBin, err := ffwrap.FindFFprobe()
	if err != nil {
		return err
	}

	dir := filepath.Dir(inPath)
	base := strings.TrimSuffix(filepath.Base(inPath), filepath.Ext(inPath))
	outPath := filepath.Join(dir, base+"."+outFmt)
	lastOutPath = outPath

	opts := convert.Options{
		InPath:     inPath,
		OutFmt:     outFmt,
		OutPath:    outPath,
		Quality:    quality,
		FFmpegBin:  ffBin,
		FFprobeBin: probeBin,
		OnProgress: func(sec, total float64) {
			if total > 0 {
				pct := int(sec / total * 100)
				if pct > 100 {
					pct = 100
				}
				bar.SetValue(pct)
				status.SetText(fmt.Sprintf("转码中 %d%%", pct))
			}
		},
	}
	_, err = convert.Convert(context.Background(), opts)
	return err
}
