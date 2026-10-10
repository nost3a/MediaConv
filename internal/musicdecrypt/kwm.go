package musicdecrypt

import (
	"encoding/binary"
	"errors"
)

var kwmMagic = []byte("KWM")

// kwmRC4Key 公开资料里酷我 kwm 用的固定 RC4 key
var kwmRC4Key = []byte("0gojrpv30vn30cvp0jrp30vn3")

func decryptKWM(raw []byte) ([]byte, string, error) {
	if len(raw) < 12 {
		return nil, "", errors.New("文件过短")
	}
	if string(raw[:3]) != string(kwmMagic) {
		return nil, "", errors.New("magic 不匹配，不是 kwm 文件")
	}
	// 布局: 3 "KWM" | 4 version LE | 4 audioOffset LE | header... | audio
	audioOff := int(binary.LittleEndian.Uint32(raw[7:11]))
	if audioOff <= 0 || audioOff > len(raw) {
		audioOff = 11
	}
	audio := rc4StreamXOR(kwmRC4Key, raw[audioOff:])
	return audio, ".mp3", nil
}
