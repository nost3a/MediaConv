package musicdecrypt

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	ncmMasterKey   = "hzHRAmso5kInbaxW"
	ncmMetadataKey = "#14ljk_!\\]&0U<'("
	ncmFileHeader  = "CTENFDAM"
	ncmMetaHeader  = "163 key(Don't modify):"
	ncmMetaHeader2 = "music:"
	ncmMetaXor     = 0x63
)

// ncmMetadata NCM 元数据里我们关心的字段（其余忽略）
type ncmMetadata struct {
	MusicName string `json:"musicName"`
	Format    string `json:"format"`
	Album     string `json:"album"`
}

// decryptNCM 还原网易云 ncm。算法：
//  1. 读 RC4 密钥（XOR 0x64 → AES-128-ECB(masterKey) 解密 → 去 PKCS7 → 跳过 17 字节）
//  2. 用密钥做 RC4 式 KSA 生成 keyBox
//  3. 读并校验元数据（XOR 0x63 → Base64 解码 → AES-128-ECB(metadataKey) 解密 → JSON）
//  4. 跳过 crc32 + dummy + 专辑图，剩余数据用 keyBox 逐字节还原
func decryptNCM(raw []byte) ([]byte, string, error) {
	if len(raw) < 0x80 {
		return nil, "", errors.New("文件过小，不是 NCM")
	}
	if string(raw[:8]) != ncmFileHeader {
		return nil, "", errors.New("文件头不是 CTENFDAM，不是 NCM")
	}
	pos := 8
	pos += 2 // 2 字节填充

	keyLen := int(binary.LittleEndian.Uint32(raw[pos : pos+4]))
	pos += 4
	if keyLen <= 17 || pos+keyLen > len(raw) {
		return nil, "", fmt.Errorf("密钥长度异常: %d", keyLen)
	}
	key := make([]byte, keyLen)
	copy(key, raw[pos:pos+keyLen])
	pos += keyLen
	for i := range key {
		key[i] ^= 0x64
	}
	dec, err := aesECBDecrypt([]byte(ncmMasterKey), key)
	if err != nil {
		return nil, "", fmt.Errorf("密钥解密失败: %w", err)
	}
	dec = pkcs7Unpad(dec)
	if len(dec) <= 17 {
		return nil, "", errors.New("密钥解析失败")
	}
	key = dec[17:]

	// 构建 keyBox（RC4 式 KSA）
	var keyBox [256]byte
	for i := range keyBox {
		keyBox[i] = byte(i)
	}
	var prev byte
	offset := 0
	for i := 0; i < 256; i++ {
		v := keyBox[i]
		c := (int(v) + int(prev) + int(key[offset])) & 0xff
		offset = (offset + 1) % len(key)
		keyBox[i] = keyBox[c]
		keyBox[c] = v
		prev = byte(c)
	}

	// 元数据
	if pos+4 > len(raw) {
		return nil, "", errors.New("元数据长度读取越界")
	}
	metaLen := int(binary.LittleEndian.Uint32(raw[pos : pos+4]))
	pos += 4
	if metaLen < len(ncmMetaHeader) || pos+metaLen > len(raw) {
		return nil, "", fmt.Errorf("元数据长度异常: %d", metaLen)
	}
	metaLen -= len(ncmMetaHeader)
	mhdr := raw[pos : pos+len(ncmMetaHeader)]
	pos += len(ncmMetaHeader)
	for i := range mhdr {
		mhdr[i] ^= ncmMetaXor
	}
	if string(mhdr) != ncmMetaHeader {
		return nil, "", errors.New("元数据头不匹配")
	}
	metaPart := raw[pos : pos+metaLen]
	pos += metaLen
	for i := range metaPart {
		metaPart[i] ^= ncmMetaXor
	}
	decoded, err := base64.StdEncoding.DecodeString(string(metaPart))
	if err != nil {
		return nil, "", fmt.Errorf("元数据 base64 失败: %w", err)
	}
	clear, err := aesECBDecrypt([]byte(ncmMetadataKey), decoded)
	if err != nil {
		return nil, "", fmt.Errorf("元数据解密失败: %w", err)
	}
	clear = pkcs7Unpad(clear)
	if len(clear) < len(ncmMetaHeader2) || string(clear[:len(ncmMetaHeader2)]) != ncmMetaHeader2 {
		return nil, "", errors.New("元数据内容异常")
	}
	var meta ncmMetadata
	if err := json.Unmarshal(clear[len(ncmMetaHeader2):], &meta); err != nil {
		// 元数据解析失败不影响音频还原，仅拿不到格式提示
		meta = ncmMetadata{}
	}

	// 跳过 crc32(4) + dummy(5) + 专辑图(imageSize)
	if pos+4+5 > len(raw) {
		return nil, "", errors.New("尾部校验读取越界")
	}
	pos += 4 + 5
	imageSize := int(binary.LittleEndian.Uint32(raw[pos : pos+4]))
	pos += 4
	if imageSize < 0 || pos+imageSize > len(raw) {
		return nil, "", fmt.Errorf("专辑图长度异常: %d", imageSize)
	}
	pos += imageSize

	if pos >= len(raw) {
		return nil, "", errors.New("无可还原音频数据")
	}
	audio := raw[pos:]

	// RC4 改流还原
	for i := 1; i <= len(audio); i++ {
		j := i & 0xff
		idx := (int(keyBox[j]) + int(keyBox[(int(keyBox[j])+j)&0xff])) & 0xff
		audio[i-1] ^= keyBox[idx]
	}

	outExt := ""
	if meta.Format != "" {
		outExt = "." + meta.Format
	}
	return audio, outExt, nil
}
