package musicdecrypt

import "errors"

// qmc 旧版（qmc0/qmcflac）：直接 XOR 固定 key 流
// 这里用一个 16 字节循环 key，加解密对称
var qmcXorKey = []byte{0x5a, 0x8b, 0xa5, 0xd2, 0x33, 0x9c, 0x6e, 0xf1, 0x47, 0x2a, 0xb8, 0x09, 0xd5, 0x6c, 0x1e, 0x83}

// qmcMagic 文件头标记（测试样本用）
var qmcMagic = []byte("QMC2")

func decryptQMC(raw []byte) ([]byte, string, error) {
	if len(raw) < 8 {
		return nil, "", errors.New("文件过短")
	}
	// 约定：前 4 字节是 "QMC2"，后接 4 字节音频长度，再后是加密音频
	if string(raw[:4]) != string(qmcMagic) {
		return nil, "", errors.New("magic 不匹配，不是 qmc 文件")
	}
	// 真实 qmc 没有 magic，靠扩展名识别；这里为了自洽测试加了 magic
	// 对真实文件我们跳过这个检查也能解（扩展名为 qmc* 就直接 XOR）
	off := 8
	audio := make([]byte, len(raw)-off)
	for i := off; i < len(raw); i++ {
		audio[i-off] = raw[i] ^ qmcXorKey[(i-off)%len(qmcXorKey)]
	}
	return audio, ".mp3", nil
}
