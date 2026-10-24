//go:build linux && cgo

package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/nost3a/MediaConv/internal/convert"
	"github.com/nost3a/MediaConv/internal/ffwrap"
	"image/color"
)

var lastOutPath string

// blueTheme 主色 #2563eb 的浅色主题
type blueTheme struct{}

func (t *blueTheme) Color(c fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	if c == theme.ColorNamePrimary {
		return color.RGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff}
	}
	return theme.DefaultTheme().Color(c, v)
}
func (t *blueTheme) Font(s fyne.TextStyle) fyne.Resource     { return theme.DefaultTheme().Font(s) }
func (t *blueTheme) Size(n fyne.ThemeSizeName) float32       { return theme.DefaultTheme().Size(n) }
func (t *blueTheme) Icon(n fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(n) }

func startGUI() {
	myApp := app.New()
	myApp.Settings().SetTheme(&blueTheme{})
	w := myApp.NewWindow("MediaConv 音视频转换")
	w.Resize(fyne.NewSize(700, 500))

	var inPath string
	inLabel := widget.NewLabel("未选择文件")

	btnPick := widget.NewButton("选择文件", func() {
		dialog.ShowFileOpen(func(rc fyne.URIReadCloser, err error) {
			if err != nil || rc == nil {
				return
			}
			defer rc.Close()
			inPath = rc.URI().Path()
			inLabel.SetText("输入: " + filepath.Base(inPath))
		}, w)
	})

	fmtSel := widget.NewSelect([]string{"mp3", "aac", "flac", "wav", "ogg", "m4a", "opus", "mp4", "webm", "mkv"}, nil)
	fmtSel.SetSelected("mp3")
	qualSel := widget.NewSelect([]string{"128k", "192k", "320k"}, nil)
	qualSel.SetSelected("192k")

	bar := widget.NewProgressBar()
	status := widget.NewLabel("就绪")

	var btnGo *widget.Button
	btnGo = widget.NewButton("开始转换", func() {
		if inPath == "" {
			dialog.ShowError(fmt.Errorf("先选文件"), w)
			return
		}
		btnGo.Disable()
		status.SetText("处理中...")
		bar.SetValue(0)

		go func() {
			defer btnGo.Enable()
			err := doConvertGUI(inPath, fmtSel.Selected, qualSel.Selected, bar, status)
			if err != nil {
				status.SetText("失败: " + err.Error())
				dialog.ShowError(err, w)
			} else {
				status.SetText("完成: " + lastOutPath)
			}
		}()
	})

	w.SetContent(container.NewVBox(
		widget.NewLabel("MediaConv 本地转码"),
		inLabel,
		btnPick,
		widget.NewLabel("输出格式:"),
		fmtSel,
		widget.NewLabel("音质:"),
		qualSel,
		btnGo,
		bar,
		status,
	))
	w.ShowAndRun()
}

func doConvertGUI(inPath, outFmt, quality string, bar *widget.ProgressBar, status *widget.Label) error {
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
				pct := sec / total
				if pct > 1 {
					pct = 1
				}
				bar.SetValue(pct)
				status.SetText(fmt.Sprintf("转码中 %.0f%%", pct*100))
			}
		},
	}
	_, err = convert.Convert(context.Background(), opts)
	return err
}
