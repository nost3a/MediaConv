// Package musicdecrypt 把国内音乐平台私有格式还原成裸音频（mp3/flac）。
// 仅做个人备份用途，算法均为公开资料里的轻量异或/固定密钥变换。
package musicdecrypt

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrUnsupported = errors.New("不支持的私有格式")

// 支持的扩展名集合
var privateExts = map[string]bool{
	".ncm":   true,
	".kgm":   true,
	".kgma":  true,
	".qmc":   true,
	".qmcflac": true,
	".mflac": true,
	".qmc0":  true,
	".mgg":   true,
	".kwm":   true,
}

// IsPrivateExt 判断这个后缀是不是私有格式
func IsPrivateExt(name string) bool {
	return privateExts[strings.ToLower(filepath.Ext(name))]
}

// Decrypt 读 inPath，还原成裸音频写到临时目录，返回 (outPath, outExt)。
// outExt 是 .mp3 / .flac / .ogg 等。
func Decrypt(inPath string) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(inPath))
	raw, err := os.ReadFile(inPath)
	if err != nil {
		return "", "", err
	}

	var (
		audio   []byte
		outExt  string
		format  string
	)
	switch ext {
	case ".ncm":
		audio, outExt, err = decryptNCM(raw)
		format = "ncm"
	case ".kgm", ".kgma":
		audio, outExt, err = decryptKGM(raw)
		format = "kgm"
	case ".qmc", ".qmcflac", ".qmc0", ".mflac":
		audio, outExt, err = decryptQMC(raw)
		format = "qmc"
	case ".mgg":
		// mgg 沿用 qmc 系列思路
		audio, outExt, err = decryptQMC(raw)
		format = "mgg"
	case ".kwm":
		audio, outExt, err = decryptKWM(raw)
		format = "kwm"
	default:
		return "", "", ErrUnsupported
	}
	if err != nil {
		return "", "", fmt.Errorf("[%s] 还原失败: %w", format, err)
	}
	if len(audio) < 128 {
		return "", "", errors.New("解密后文件过小，疑似损坏")
	}

	// 按音频头判断真实容器
	outExt = sniffAudioExt(audio, outExt)

	dir := filepath.Dir(inPath)
	base := strings.TrimSuffix(filepath.Base(inPath), filepath.Ext(inPath))
	outPath := filepath.Join(dir, base+".decrypted"+outExt)
	if err := os.WriteFile(outPath, audio, 0644); err != nil {
		return "", "", err
	}
	return outPath, outExt, nil
}

// sniffAudioExt 按音频头猜容器，回退到传入的默认值
func sniffAudioExt(b []byte, def string) string {
	if len(b) < 3 {
		return def
	}
	switch {
	case b[0] == 0x49 && b[1] == 0x44 && b[2] == 0x33: // ID3
		return ".mp3"
	case len(b) >= 2 && b[0] == 0xFF && (b[1]&0xE0) == 0xE0: // mp3 frame sync
		return ".mp3"
	case b[0] == 'f' && b[1] == 'L' && b[2] == 'a' && b[3] == 'C':
		return ".flac"
	case b[0] == 'O' && b[1] == 'g' && b[2] == 'g':
		return ".ogg"
	case b[4] == 'f' && b[5] == 't' && b[6] == 'y' && b[7] == 'p':
		return ".m4a"
	case b[0] == 'R' && b[1] == 'I' && b[2] == 'F' && b[3] == 'F':
		return ".wav"
	}
	return def
}
