// Package musicdecrypt 把国内音乐平台私有格式还原成裸音频（mp3/flac/ogg/m4a）。
//
// 算法均来自公开逆向资料（keuin/ncm、unlock-music 系与 ilyhacker/kgm-decrypt），
// 仅用于个人备份，请勿用于其他用途。
//
// 注意：本包只还原离线可解的变体——
//   - ncm：完整支持（AES-128-ECB 还原 RC4 密钥 + RC4 改流向）
//   - kgm / kgma / vpr：完整支持（17 字节密钥 + 两阶段 XOR + 掩码表）
//   - kwm：完整支持（8 字节 fileKey → 32 字节掩码）
//   - qmc0/qmc2/qmc3/qmcogg/qmcflac/bkcmp3/bkcflac/tkm：默认掩码矩阵离线可解
//   - mflac / mgg：需要文件内嵌密钥（keybox），离线无法还原，会返回明确错误
package musicdecrypt

import (
	"crypto/aes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrUnsupported = errors.New("不支持的私有格式")

// 支持的扩展名集合
var privateExts = map[string]bool{
	".ncm":     true,
	".kgm":     true,
	".kgma":    true,
	".qmc":     true,
	".qmcflac": true,
	".qmc0":    true,
	".qmc2":    true,
	".qmc3":    true,
	".qmcogg":  true,
	".bkcmp3":  true,
	".bkcflac": true,
	".tkm":     true,
	".mflac":   true, // 需内嵌密钥，离线无法还原
	".mgg":     true, // 需内嵌密钥，离线无法还原
	".kwm":     true,
}

// IsPrivateExt 判断这个后缀是不是私有格式
func IsPrivateExt(name string) bool {
	return privateExts[strings.ToLower(filepath.Ext(name))]
}

// Decrypt 读 inPath，还原成裸音频并写到系统临时目录，
// 返回 (outPath, outExt)。outPath 由调用方负责清理（通常 defer os.Remove）。
// outExt 形如 .mp3 / .flac / .ogg / .m4a。
func Decrypt(inPath string) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(inPath))
	raw, err := os.ReadFile(inPath)
	if err != nil {
		return "", "", err
	}

	var (
		audio  []byte
		outExt string
		format string
	)
	switch ext {
	case ".ncm":
		audio, outExt, err = decryptNCM(raw)
		format = "ncm"
	case ".kgm", ".kgma", ".vpr":
		audio, outExt, err = decryptKGM(raw)
		format = "kgm"
	case ".qmc", ".qmcflac", ".qmc0", ".qmc2", ".qmc3", ".qmcogg", ".bkcmp3", ".bkcflac", ".tkm":
		audio, outExt, err = decryptQMC(raw)
		format = "qmc"
	case ".mflac", ".mgg":
		return "", "", fmt.Errorf("[%s] 该变体需文件内嵌密钥(keybox)，离线无法还原，请用源平台导出", format)
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
		return "", "", errors.New("解密后文件过小，疑似损坏或格式不匹配")
	}

	// 按音频头判断真实容器（覆盖传入的默认值）
	outExt = sniffAudioExt(audio, outExt)

	f, err := os.CreateTemp("", "mediaconv-dec-*"+outExt)
	if err != nil {
		return "", "", err
	}
	if _, err := f.Write(audio); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", "", err
	}
	return f.Name(), outExt, nil
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

// aesECBDecrypt AES-128/256 ECB 解密（密钥长度决定）。输入必须按块对齐。
func aesECBDecrypt(key, src []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	bs := block.BlockSize()
	if len(src)%bs != 0 {
		return nil, fmt.Errorf("密文长度 %d 不是块大小 %d 的整数倍", len(src), bs)
	}
	out := make([]byte, len(src))
	for i := 0; i < len(src); i += bs {
		block.Decrypt(out[i:i+bs], src[i:i+bs])
	}
	return out, nil
}

// pkcs7Unpad 去掉 PKCS7 填充
func pkcs7Unpad(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	n := int(b[len(b)-1])
	if n == 0 || n > len(b) {
		return b
	}
	return b[:len(b)-n]
}
