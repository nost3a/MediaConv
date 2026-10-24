// 一次性工具：把 test/sample.mp3 反向加密成 4 种私有格式样本，放到 test/ 下
package main

import (
	"crypto/aes"
	"encoding/binary"
	"io/ioutil"
	"os"
)

var ncmMagic = []byte("CTENFDAM")
var ncmAESKey = []byte("hzHRz0r4z5!Z3$eX")
var kgmMagic1 = []byte("#KGM")
var kgmXorTable = []byte{
	0x6a, 0x48, 0xb3, 0x4a, 0xc4, 0x80, 0x8c, 0x80,
	0x9f, 0xc3, 0x8a, 0x64, 0x98, 0x19, 0x20, 0x48,
	0x96, 0x48, 0x3b, 0x48, 0x98, 0x19, 0x20, 0x48,
	0x9f, 0xc3, 0x8a, 0x64, 0x98, 0x19, 0x20, 0x48,
	0x6a, 0x48, 0xb3, 0x4a, 0xc4, 0x80, 0x8c, 0x80,
	0x9f, 0xc3, 0x8a, 0x64, 0x98, 0x19, 0x20, 0x48,
	0x96, 0x48, 0x3b, 0x48, 0x98, 0x19, 0x20, 0x48,
	0x9f, 0xc3, 0x8a, 0x64, 0x98, 0x19, 0x20, 0x48,
	0x6a, 0x48, 0xb3, 0x4a, 0xc4, 0x80, 0x8c, 0x80,
	0x9f, 0xc3, 0x8a, 0x64, 0x98, 0x19, 0x20, 0x48,
	0x96, 0x48, 0x3b, 0x48, 0x98, 0x19, 0x20, 0x48,
	0x9f, 0xc3, 0x8a, 0x64, 0x98, 0x19, 0x20, 0x48,
	0x6a, 0x48, 0xb3, 0x4a, 0xc4, 0x80, 0x8c, 0x80,
	0x9f, 0xc3, 0x8a, 0x64, 0x98, 0x19, 0x20, 0x48,
	0x96, 0x48, 0x3b, 0x48, 0x98, 0x19, 0x20, 0x48,
	0x9f, 0xc3, 0x8a, 0x64, 0x98, 0x19, 0x20, 0x48,
}
var qmcMagic = []byte("QMC2")
var qmcXorKey = []byte{0x5a, 0x8b, 0xa5, 0xd2, 0x33, 0x9c, 0x6e, 0xf1, 0x47, 0x2a, 0xb8, 0x09, 0xd5, 0x6c, 0x1e, 0x83}
var kwmMagic = []byte("KWM")
var kwmRC4Key = []byte("0gojrpv30vn30cvp0jrp30vn3")

func rc4XOR(key, data []byte) []byte {
	s := make([]byte, 256)
	for i := 0; i < 256; i++ {
		s[i] = byte(i)
	}
	j := 0
	for i := 0; i < 256; i++ {
		j = (j + int(s[i]) + int(key[i%len(key)])) & 0xff
		s[i], s[j] = s[j], s[i]
	}
	out := make([]byte, len(data))
	si, sj := 0, 0
	for i := 0; i < len(data); i++ {
		si = (si + 1) & 0xff
		sj = (sj + int(s[si])) & 0xff
		s[si], s[sj] = s[sj], s[si]
		out[i] = data[i] ^ s[(int(s[si])+int(s[sj]))&0xff]
	}
	return out
}

func main() {
	mp3, _ := ioutil.ReadFile("test/sample.mp3")
	rc4Key := []byte("1234567890abcdef")

	// ncm
	fullKey := append([]byte("neteasecloudmusic"), rc4Key...)
	pad := aes.BlockSize - len(fullKey)%aes.BlockSize
	padded := make([]byte, len(fullKey)+pad)
	copy(padded, fullKey)
	block, _ := aes.NewCipher(ncmAESKey)
	enc := make([]byte, len(padded))
	for i := 0; i < len(padded); i += aes.BlockSize {
		block.Encrypt(enc[i:i+aes.BlockSize], padded[i:i+aes.BlockSize])
	}
	for i := range enc {
		enc[i] ^= 0x64
	}
	ncmOut := append([]byte{}, ncmMagic...)
	ncmOut = append(ncmOut, 1, 0)
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(len(enc)))
	ncmOut = append(ncmOut, buf...)
	ncmOut = append(ncmOut, enc...)
	ncmOut = append(ncmOut, rc4XOR(rc4Key, mp3)...)
	ioutil.WriteFile("test/sample.ncm", ncmOut, 0644)

	// kgm
	kgmOut := append([]byte{}, kgmMagic1...)
	kgmOut = append(kgmOut, make([]byte, 0x40-4)...)
	for i, b := range mp3 {
		kgmOut = append(kgmOut, b^kgmXorTable[i%len(kgmXorTable)])
	}
	ioutil.WriteFile("test/sample.kgm", kgmOut, 0644)

	// qmc
	qmcOut := append([]byte{}, qmcMagic...)
	binary.LittleEndian.PutUint32(buf, uint32(len(mp3)))
	qmcOut = append(qmcOut, buf...)
	for i, b := range mp3 {
		qmcOut = append(qmcOut, b^qmcXorKey[i%len(qmcXorKey)])
	}
	ioutil.WriteFile("test/sample.qmc", qmcOut, 0644)

	// kwm
	kwmOut := append([]byte{}, kwmMagic...)
	ver := make([]byte, 4)
	binary.LittleEndian.PutUint32(ver, 2)
	kwmOut = append(kwmOut, ver...)
	off := make([]byte, 4)
	binary.LittleEndian.PutUint32(off, 11)
	kwmOut = append(kwmOut, off...)
	kwmOut = append(kwmOut, rc4XOR(kwmRC4Key, mp3)...)
	ioutil.WriteFile("test/sample.kwm", kwmOut, 0644)

	os.Chdir("test")
	for _, f := range []string{"sample.ncm", "sample.kgm", "sample.qmc", "sample.kwm"} {
		st, _ := os.Stat(f)
		println(f, st.Size())
	}
}
