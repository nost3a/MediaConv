package musicdecrypt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strconv"
)

// kwmMagicHeader "yeelion-kuo-tme"
var kwmMagicHeader = [16]byte{
	0x79, 0x65, 0x65, 0x6C, 0x69, 0x6F, 0x6E, 0x2D,
	0x6B, 0x75, 0x6F, 0x2D, 0x74, 0x6D, 0x65,
}

// kwmPreDefKey 32 字节预定义密钥，与 fileKey 推算出的字符串逐字节异或得到掩码
const kwmPreDefKey = "MoOtOiTvINGwd2E6n0E1i7L5t2IoOoNk"

// kwmCreateMask 由 fileKey(8 字节) 生成 32 字节掩码：
// 1. 把 fileKey 当小端 uint64 转成十进制字符串
// 2. 截/补到 32 字节
// 3. mask[i] = PreDefKey[i] ^ keyStr[i]
func kwmCreateMask(fileKey []byte) [32]byte {
	num := binary.LittleEndian.Uint64(fileKey)
	s := strconv.FormatUint(num, 10)
	if len(s) > 32 {
		s = s[:32]
	}
	for len(s) < 32 {
		s += s
	}
	s = s[:32]
	var mask [32]byte
	for i := 0; i < 32; i++ {
		mask[i] = kwmPreDefKey[i] ^ s[i]
	}
	return mask
}

// decryptKWM 还原酷我 kwm。
//  1. 校验 16 字节头部 "yeelion-kuo-tme"
//  2. fileKey = data[0x18:0x20]，生成 32 字节掩码
//  3. 音频从 0x400 开始，逐字节 out[i] ^= mask[i%32]
func decryptKWM(raw []byte) ([]byte, string, error) {
	if len(raw) < 0x400 {
		return nil, "", errors.New("文件过小，不是 KWM")
	}
	if !bytes.Equal(raw[:0x10], kwmMagicHeader[:]) {
		return nil, "", errors.New("头部不匹配，不是 KWM")
	}
	fileKey := raw[0x18:0x20]
	mask := kwmCreateMask(fileKey)
	audio := raw[0x400:]
	out := make([]byte, len(audio))
	copy(out, audio)
	for i := range out {
		out[i] ^= mask[i%32]
	}
	return out, "", nil
}
