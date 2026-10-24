package musicdecrypt

import (
	"errors"
	"sort"
)

// qmcDefaultMaskMatrix 默认 44 字节掩码矩阵。
// 用于离线可解的 QMC 变体（qmc0/qmc2/qmc3/qmcogg/qmcflac/bkcmp3/bkcflac/tkm）。
// 来自 unlock-music 系的 QMCDefaultMaskMatrix。
var qmcDefaultMaskMatrix = [44]byte{
	0xde, 0x51, 0xfa, 0xc3, 0x4a, 0xd6, 0xca, 0x90,
	0x7e, 0x67, 0x5e, 0xf7, 0xd5, 0x52, 0x84, 0xd8,
	0x47, 0x95, 0xbb, 0xa1, 0xaa, 0xc6, 0x66, 0x23,
	0x92, 0x62, 0xf3, 0x74, 0xa1, 0x9f, 0xf4, 0xa0,
	0x1d, 0x3f, 0x5b, 0xf0, 0x13, 0x0e, 0x09, 0x3d,
	0xf9, 0xbc, 0x00, 0x11,
}

// qmcMatrix128 由默认 44 字节矩阵展开成 256 字节（每个索引 i 映射到 group，
// group 由 (i*i+27) % 256 决定，同一 group 共用同一个掩码字节）。
var qmcMatrix128 [256]byte

func init() {
	// 构建 realIdx -> []i 的映射
	mapping := map[int][]int{}
	for i := 0; i < 256; i++ {
		r := (i*i + 27) % 256
		mapping[r] = append(mapping[r], i)
	}
	// 按 realIdx 升序分配 group 序号（idx44）
	keys := make([]int, 0, len(mapping))
	for r := range mapping {
		keys = append(keys, r)
	}
	sort.Ints(keys)
	groupOf := make([]int, 256)
	for idx44, r := range keys {
		for _, i := range mapping[r] {
			groupOf[i] = idx44
		}
	}
	for i := 0; i < 256; i++ {
		qmcMatrix128[i] = qmcDefaultMaskMatrix[groupOf[i]]
	}
}

// decryptQMC 用默认掩码矩阵还原 QMC 音频。
// 偏移规则（来自 unlock-music 的 Decrypt）：每 0x8000 字节插入一个额外掩码步进，
// 掩码下标对 128 取模。
func decryptQMC(raw []byte) ([]byte, string, error) {
	if len(raw) < 16 {
		return nil, "", errors.New("文件过小，不是 QMC")
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	index := -1
	maskIdx := -1
	for cur := 0; cur < len(out); cur++ {
		index++
		maskIdx++
		if index == 0x8000 || (index > 0x8000 && (index+1)%0x8000 == 0) {
			index++
			maskIdx++
		}
		if maskIdx >= 128 {
			maskIdx -= 128
		}
		out[cur] ^= qmcMatrix128[maskIdx]
	}
	// 默认矩阵不会在开头就产生合法音频头，交由 sniffAudioExt 判定；
	// 若调用方发现结果非法（如 mflac/mgg），会在上层返回明确错误。
	return out, "", nil
}
