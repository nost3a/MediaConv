package musicdecrypt

import (
	"crypto/aes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// 造一个假的 mp3 字节流（ID3 头 + 随机内容），测试 round-trip
func fakeMP3() []byte {
	b := []byte{'I', 'D', '3', 3, 0, 0, 0, 0, 0, 0x20}
	// 1KB 伪音频数据
	for i := 0; i < 1024; i++ {
		b = append(b, byte(i*7+3))
	}
	return b
}

func md5hex(b []byte) string {
	s := md5.Sum(b)
	return hex.EncodeToString(s[:])
}

// 反向构造 ncm 样本：encrypt 是 decrypt 的逆
func buildNCM(t *testing.T, mp3 []byte) []byte {
	// 1. 准备 rc4 key（16 字节）
	rc4Key := []byte("1234567890abcdef")
	// 2. 前面拼 "neteasecloudmusic" (17 字节)，再做 AES-ECB 加密 + 每个字节 XOR 0x64
	fullKey := append([]byte("neteasecloudmusic"), rc4Key...)
	// 全 0 补齐到 16 倍数（与 decrypt 的去 0 尾逻辑对应）
	pad := aes.BlockSize - len(fullKey)%aes.BlockSize
	padded := make([]byte, len(fullKey)+pad)
	copy(padded, fullKey)
	block, _ := aes.NewCipher(ncmAESKey)
	enc := make([]byte, len(padded))
	for i := 0; i < len(padded); i += aes.BlockSize {
		block.Encrypt(enc[i:i+aes.BlockSize], padded[i:i+aes.BlockSize])
	}
	// 每个字节 XOR 0x64
	for i := range enc {
		enc[i] ^= 0x64
	}
	// 3. 音频段：rc4 stream XOR
	audio := rc4StreamXOR(rc4Key, mp3)

	// 拼文件: magic(8) + skip(2) + keyLen(4 LE) + encKey + audio
	out := append([]byte{}, ncmMagic...)
	out = append(out, 1, 0)
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(len(enc)))
	out = append(out, buf...)
	out = append(out, enc...)
	out = append(out, audio...)
	return out
}

func buildKGM(t *testing.T, mp3 []byte) []byte {
	// magic 4 字节 + 0x3C 空 header + XOR 音频
	head := append([]byte{}, kgmMagic1...)
	head = append(head, make([]byte, 0x40-4)...)
	body := make([]byte, len(mp3))
	for i, b := range mp3 {
		body[i] = b ^ kgmXorTable[i%len(kgmXorTable)]
	}
	return append(head, body...)
}

func buildQMC(t *testing.T, mp3 []byte) []byte {
	head := append([]byte{}, qmcMagic...)
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(len(mp3)))
	head = append(head, buf...)
	body := make([]byte, len(mp3))
	for i, b := range mp3 {
		body[i] = b ^ qmcXorKey[i%len(qmcXorKey)]
	}
	return append(head, body...)
}

func buildKWM(t *testing.T, mp3 []byte) []byte {
	head := append([]byte{}, kwmMagic...)
	ver := make([]byte, 4)
	binary.LittleEndian.PutUint32(ver, 2)
	head = append(head, ver...)
	off := make([]byte, 4)
	binary.LittleEndian.PutUint32(off, 11) // audioOffset = 11
	head = append(head, off...)
	body := rc4StreamXOR(kwmRC4Key, mp3)
	return append(head, body...)
}

func TestRoundTrip(t *testing.T) {
	mp3 := fakeMP3()
	origMD5 := md5hex(mp3)
	t.Logf("原始 mp3 MD5=%s  size=%d", origMD5, len(mp3))

	cases := []struct {
		name    string
		ext     string
		build   func(*testing.T, []byte) []byte
	}{
		{"ncm", ".ncm", buildNCM},
		{"kgm", ".kgm", buildKGM},
		{"qmc", ".qmc", buildQMC},
		{"kwm", ".kwm", buildKWM},
	}

	dir := t.TempDir()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sample := c.build(t, mp3)
			samplePath := filepath.Join(dir, "test"+c.ext)
			if err := os.WriteFile(samplePath, sample, 0644); err != nil {
				t.Fatal(err)
			}
			outPath, outExt, err := Decrypt(samplePath)
			if err != nil {
				t.Fatalf("Decrypt 失败: %v", err)
			}
			defer os.Remove(outPath)
			got, _ := os.ReadFile(outPath)
			gotMD5 := md5hex(got)
			t.Logf("  sample=%d bytes, out=%s, outExt=%s, MD5=%s, match=%v",
				len(sample), outPath, outExt, gotMD5, gotMD5 == origMD5)
			if gotMD5 != origMD5 {
				t.Errorf("MD5 不一致: orig=%s got=%s", origMD5, gotMD5)
			}
		})
	}
}
