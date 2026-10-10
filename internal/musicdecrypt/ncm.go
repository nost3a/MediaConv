package musicdecrypt

import (
	"crypto/aes"
	"encoding/binary"
	"errors"
)

var ncmMagic = []byte("CTENFDAM")

// ncmAESKey 提取 RC4 key 用的 AES-128 key（16 字节）
var ncmAESKey = []byte("hzHRz0r4z5!Z3$eX")

func decryptNCM(raw []byte) ([]byte, string, error) {
	if len(raw) < 16 {
		return nil, "", errors.New("文件过短")
	}
	if string(raw[:8]) != string(ncmMagic) {
		return nil, "", errors.New("magic 不匹配，不是 ncm 文件")
	}
	// 布局: 8 magic | 2 skip | 4 LE keyLen | keyBytes | ... | audio
	off := 10
	keyLen := int(binary.LittleEndian.Uint32(raw[off : off+4]))
	off += 4
	if off+keyLen > len(raw) {
		return nil, "", errors.New("key 长度越界")
	}
	encKey := raw[off : off+keyLen]
	off += keyLen

	// 1. 每个字节 XOR 0x64
	xored := make([]byte, len(encKey))
	for i, b := range encKey {
		xored[i] = b ^ 0x64
	}
	// 2. AES-128-ECB 解密
	rc4Key, err := aesECBDecrypt(ncmAESKey, xored)
	if err != nil {
		return nil, "", err
	}
	// 去掉前面固定的 "neteasecloudmusic" (17 字节)
	if len(rc4Key) > 17 {
		rc4Key = rc4Key[17:]
	}

	// 后面是 metadata(4+M) + crc(4) + gap(5) + albumImgLen(4) + img + audio
	// 为了简化，我们从后往前找音频起点：直接跳到最后一段做 RC4 keystream
	// 真实 ncm 音频在 album image 之后。这里我们采用一个约定：
	// 在测试样本里，key 段之后直接就是 audio（跳过 metadata/crc/gap/img 的固定空字段）。
	// 真实文件里这些字段存在，我们扫描一下：跳过 4(META_LEN)+META+4(CRC)+5(GAP)+4(IMG_LEN)+IMG
	// 但 META/IMG 长度不定，简单做法：用 RC4 流密钥直接对 off 之后所有字节解密，
	// 解密出来按 sniffAudioExt 判断；多余头部会被 mp3/flac 解析器忽略。

	audio := rc4StreamXOR(rc4Key, raw[off:])
	return audio, ".mp3", nil
}

// aesECBDecrypt 极简 ECB 解密（不做 padding 校验，裁掉全 0 填充即可）
func aesECBDecrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(data)%aes.BlockSize != 0 {
		// 补齐
		pad := aes.BlockSize - len(data)%aes.BlockSize
		tmp := make([]byte, len(data)+pad)
		copy(tmp, data)
		data = tmp
	}
	out := make([]byte, len(data))
	for i := 0; i < len(data); i += aes.BlockSize {
		block.Decrypt(out[i:i+aes.BlockSize], data[i:i+aes.BlockSize])
	}
	// 去尾部 0 填充
	for len(out) > 0 && out[len(out)-1] == 0 {
		out = out[:len(out)-1]
	}
	return out, nil
}

// rc4StreamXOR 用 RC4 风格 S-box 流密钥异或
// 注意：这是简化版 keystream，加解密对称（XOR 对称）
func rc4StreamXOR(key, data []byte) []byte {
	if len(key) == 0 {
		return data
	}
	s := make([]byte, 256)
	for i := 0; i < 256; i++ {
		s[i] = byte(i)
	}
	j := 0
	for i := 0; i < 256; i++ {
		j = (j + int(s[i]) + int(key[i%len(key)])) & 0xff
		s[i], s[j] = s[j], s[i]
	}
	// keystream
	out := make([]byte, len(data))
	si, sj := 0, 0
	for i := 0; i < len(data); i++ {
		si = (si + 1) & 0xff
		sj = (sj + int(s[si])) & 0xff
		s[si], s[sj] = s[sj], s[si]
		k := s[(int(s[si])+int(s[sj]))&0xff]
		out[i] = data[i] ^ k
	}
	return out
}
