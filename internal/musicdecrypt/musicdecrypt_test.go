package musicdecrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/md5"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func md5hex(b []byte) string {
	s := md5.Sum(b)
	return hex.EncodeToString(s[:])
}

func aesECBEncrypt(key, src []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	bs := block.BlockSize()
	pad := bs - len(src)%bs
	padded := append(src, bytes.Repeat([]byte{byte(pad)}, pad)...)
	out := make([]byte, len(padded))
	for i := 0; i < len(padded); i += bs {
		block.Encrypt(out[i:i+bs], padded[i:i+bs])
	}
	return out, nil
}

// f(x) = x ^ ((x & 0xf) << 4)，是对合变换（f(f(x)) == x）
func f(x byte) byte { return x ^ ((x & 0xf) << 4) }

func fakeMP3() []byte {
	b := []byte{'I', 'D', '3', 3, 0, 0, 0, 0, 0, 0x20}
	for i := 0; i < 1024; i++ {
		b = append(b, byte(i*7+3))
	}
	return b
}

func fakeFLAC() []byte {
	b := []byte{'f', 'L', 'a', 'C', 0, 0, 0, 0}
	for i := 0; i < 1024; i++ {
		b = append(b, byte(i*11+5))
	}
	return b
}

// ---- 用新算法逆运算构造样本，供 round-trip 验证 ----

func buildNCM(_ *testing.T, audio []byte) []byte {
	rc4Key := []byte("mysecretkey1234")                       // 15 字节，使 fullKey 总长 32（16 对齐）
	fullKey := append([]byte("neteasecloudmusic"), rc4Key...) // 32 字节
	enc, _ := aesECBEncrypt([]byte(ncmMasterKey), fullKey)    // 内部已做 PKCS7 填充
	storedKey := make([]byte, len(enc))
	for i := range enc {
		storedKey[i] = enc[i] ^ 0x64
	}
	keyLen := len(storedKey)

	// 元数据（16 字节，内部填充后整块还原）
	clear := []byte(`music:{"f":"mp3"}`)
	metaEnc, _ := aesECBEncrypt([]byte(ncmMetadataKey), clear)
	metaB64 := base64.StdEncoding.EncodeToString(metaEnc)
	storedMeta := make([]byte, len(metaB64))
	for i := range metaB64 {
		storedMeta[i] = metaB64[i] ^ ncmMetaXor
	}
	mhdr := make([]byte, len(ncmMetaHeader))
	for i := range ncmMetaHeader {
		mhdr[i] = ncmMetaHeader[i] ^ ncmMetaXor
	}
	metaLen := len(ncmMetaHeader) + len(storedMeta)

	// 音频：用 keyBox 还原的逆运算
	var keyBox [256]byte
	for i := range keyBox {
		keyBox[i] = byte(i)
	}
	var prev byte
	offset := 0
	for i := 0; i < 256; i++ {
		v := keyBox[i]
		c := (int(v) + int(prev) + int(rc4Key[offset%len(rc4Key)])) & 0xff
		offset++
		keyBox[i] = keyBox[c]
		keyBox[c] = v
		prev = byte(c)
	}
	audioEnc := make([]byte, len(audio))
	for i := 1; i <= len(audio); i++ {
		j := i & 0xff
		idx := (int(keyBox[j]) + int(keyBox[(int(keyBox[j])+j)&0xff])) & 0xff
		audioEnc[i-1] = audio[i-1] ^ keyBox[idx]
	}

	var buf bytes.Buffer
	buf.WriteString(ncmFileHeader)
	buf.Write([]byte{0, 0})
	kl := make([]byte, 4)
	binary.LittleEndian.PutUint32(kl, uint32(keyLen))
	buf.Write(kl)
	buf.Write(storedKey)
	ml := make([]byte, 4)
	binary.LittleEndian.PutUint32(ml, uint32(metaLen))
	buf.Write(ml)
	buf.Write(mhdr)
	buf.Write(storedMeta)
	buf.Write([]byte{0, 0, 0, 0})    // crc32
	buf.Write([]byte{0, 0, 0, 0, 0}) // dummy
	il := make([]byte, 4)
	binary.LittleEndian.PutUint32(il, 0)
	buf.Write(il) // imageSize = 0
	buf.Write(audioEnc)
	return buf.Bytes()
}

func buildKGM(_ *testing.T, plain []byte) []byte {
	// 解密端读 raw[0x1c:0x2c] 只有 16 字节（key[16] 恒为 0），故这里 key 也取 16 字节 + 末尾补 0
	key := append([]byte("0123456789abcdef"), 0)
	headerLen := 0x40
	header := make([]byte, headerLen)
	copy(header[0:16], kgmHeader[:])
	hl := make([]byte, 4)
	binary.LittleEndian.PutUint32(hl, uint32(headerLen))
	copy(header[0x10:0x14], hl)
	copy(header[0x1c:0x2c], key)
	audio := make([]byte, len(plain))
	for i := range plain {
		g := kgmGetMask(i)
		A := f(plain[i] ^ f(g))
		audio[i] = key[i%17] ^ A
	}
	return append(header, audio...)
}

func buildKWM(_ *testing.T, plain []byte) []byte {
	fileKey := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	raw := make([]byte, 0x400+len(plain))
	copy(raw[0:16], kwmMagicHeader[:])
	copy(raw[0x18:0x20], fileKey)
	mask := kwmCreateMask(fileKey)
	for i := range plain {
		raw[0x400+i] = plain[i] ^ mask[i%32]
	}
	return raw
}

func buildQMC(_ *testing.T, plain []byte) []byte {
	out := make([]byte, len(plain))
	index := -1
	maskIdx := -1
	for cur := 0; cur < len(plain); cur++ {
		index++
		maskIdx++
		if index == 0x8000 || (index > 0x8000 && (index+1)%0x8000 == 0) {
			index++
			maskIdx++
		}
		if maskIdx >= 128 {
			maskIdx -= 128
		}
		out[cur] = plain[cur] ^ qmcMatrix128[maskIdx]
	}
	return out
}

func TestRoundTrip(t *testing.T) {
	plain := fakeMP3()
	origMD5 := md5hex(plain)

	cases := []struct {
		name  string
		ext   string
		build func(*testing.T, []byte) []byte
	}{
		{"ncm", ".ncm", buildNCM},
		{"kgm", ".kgm", buildKGM},
		{"kwm", ".kwm", buildKWM},
		{"qmc", ".qmcflac", buildQMC},
	}

	dir := t.TempDir()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sample := c.build(t, plain)
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
			if gotMD5 != origMD5 {
				t.Errorf("MD5 不一致: orig=%s got=%s (outExt=%s)", origMD5, gotMD5, outExt)
			}
			if outExt != ".mp3" {
				t.Logf("outExt=%s（提示）", outExt)
			}
		})
	}
}

func TestSniffOutExts(t *testing.T) {
	if sniffAudioExt([]byte("fLaCxxxx"), "") != ".flac" {
		t.Error("flac 嗅探失败")
	}
	if sniffAudioExt([]byte("Oggxxxx"), "") != ".ogg" {
		t.Error("ogg 嗅探失败")
	}
}
