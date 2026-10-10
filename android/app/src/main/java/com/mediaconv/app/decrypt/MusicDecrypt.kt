package com.mediaconv.app.decrypt

import java.io.ByteArrayOutputStream
import javax.crypto.Cipher
import javax.crypto.spec.SecretKeySpec

// Kotlin 移植 internal/musicdecrypt/，算法与 Go 版一致
object MusicDecrypt {

    private val ncmAESKey = "hzHRz0r4z5!\$eX".toByteArray()
    private val kgmXorTable = byteArrayOf(
        0x6a, 0x48, 0xb3.toByte(), 0x4a, 0xc4.toByte(), 0x80.toByte(), 0x8c.toByte(), 0x80.toByte(),
        0x9f.toByte(), 0xc3.toByte(), 0x8a.toByte(), 0x64, 0x98.toByte(), 0x19, 0x20, 0x48,
        0x96.toByte(), 0x48, 0x3b, 0x48, 0x98.toByte(), 0x19, 0x20, 0x48,
        0x9f.toByte(), 0xc3.toByte(), 0x8a.toByte(), 0x64, 0x98.toByte(), 0x19, 0x20, 0x48,
        0x6a, 0x48, 0xb3.toByte(), 0x4a, 0xc4.toByte(), 0x80.toByte(), 0x8c.toByte(), 0x80.toByte(),
        0x9f.toByte(), 0xc3.toByte(), 0x8a.toByte(), 0x64, 0x98.toByte(), 0x19, 0x20, 0x48,
        0x96.toByte(), 0x48, 0x3b, 0x48, 0x98.toByte(), 0x19, 0x20, 0x48,
        0x9f.toByte(), 0xc3.toByte(), 0x8a.toByte(), 0x64, 0x98.toByte(), 0x19, 0x20, 0x48,
        0x6a, 0x48, 0xb3.toByte(), 0x4a, 0xc4.toByte(), 0x80.toByte(), 0x8c.toByte(), 0x80.toByte(),
        0x9f.toByte(), 0xc3.toByte(), 0x8a.toByte(), 0x64, 0x98.toByte(), 0x19, 0x20, 0x48,
        0x96.toByte(), 0x48, 0x3b, 0x48, 0x98.toByte(), 0x19, 0x20, 0x48,
        0x9f.toByte(), 0xc3.toByte(), 0x8a.toByte(), 0x64, 0x98.toByte(), 0x19, 0x20, 0x48,
        0x6a, 0x48, 0xb3.toByte(), 0x4a, 0xc4.toByte(), 0x80.toByte(), 0x8c.toByte(), 0x80.toByte(),
        0x9f.toByte(), 0xc3.toByte(), 0x8a.toByte(), 0x64, 0x98.toByte(), 0x19, 0x20, 0x48,
        0x96.toByte(), 0x48, 0x3b, 0x48, 0x98.toByte(), 0x19, 0x20, 0x48,
        0x9f.toByte(), 0xc3.toByte(), 0x8a.toByte(), 0x64, 0x98.toByte(), 0x19, 0x20, 0x48,
    )
    private val qmcXorKey = byteArrayOf(
        0x5a, 0x8b.toByte(), 0xa5.toByte(), 0xd2.toByte(), 0x33, 0x9c.toByte(), 0x6e, 0xf1.toByte(),
        0x47, 0x2a, 0xb8.toByte(), 0x09, 0xd5.toByte(), 0x6c, 0x1e, 0x83.toByte(),
    )
    private val kwmRC4Key = "0gojrpv30vn30cvp0jrp30vn3".toByteArray()

    fun isPrivateExt(name: String): Boolean {
        val e = name.lowercase().substringAfterLast('.', "")
        return e in setOf("ncm", "kgm", "kgma", "qmc", "qmcflac", "qmc0", "mflac", "mgg", "kwm")
    }

    fun decrypt(raw: ByteArray, ext: String): ByteArray {
        return when (ext.lowercase()) {
            ".ncm" -> decryptNCM(raw)
            ".kgm", ".kgma" -> decryptKGM(raw)
            ".qmc", ".qmcflac", ".qmc0", ".mflac", ".mgg" -> decryptQMC(raw)
            ".kwm" -> decryptKWM(raw)
            else -> raw
        }
    }

    private fun aesECBDecrypt(key: ByteArray, data: ByteArray): ByteArray {
        val cipher = Cipher.getInstance("AES/ECB/NoPadding")
        cipher.init(Cipher.DECRYPT_MODE, SecretKeySpec(key, "AES"))
        val out = cipher.doFinal(data)
        // 去尾部 0
        var n = out.size
        while (n > 0 && out[n-1] == 0.toByte()) n--
        return out.copyOf(n)
    }

    private fun rc4XOR(key: ByteArray, data: ByteArray): ByteArray {
        val s = ByteArray(256) { it.toByte() }
        var j = 0
        for (i in 0 until 256) {
            j = (j + s[i].toInt() + key[i % key.size].toInt()) and 0xff
            val t = s[i]; s[i] = s[j]; s[j] = t
        }
        val out = ByteArray(data.size)
        var si = 0
        var sj = 0
        for (i in data.indices) {
            si = (si + 1) and 0xff
            sj = (sj + s[si].toInt()) and 0xff
            val t = s[si]; s[si] = s[sj]; s[sj] = t
            val k = s[(s[si].toInt() + s[sj].toInt()) and 0xff]
            out[i] = (data[i].toInt() xor k.toInt()).toByte()
        }
        return out
    }

    private fun decryptNCM(raw: ByteArray): ByteArray {
        require(raw.size >= 16) { "文件过短" }
        require(String(raw.copyOfRange(0, 8)) == "CTENFDAM") { "magic 不匹配" }
        var off = 10
        val keyLen = (raw[off].toInt() and 0xff) or
                ((raw[off+1].toInt() and 0xff) shl 8) or
                ((raw[off+2].toInt() and 0xff) shl 16) or
                ((raw[off+3].toInt() and 0xff) shl 24)
        off += 4
        val encKey = raw.copyOfRange(off, off + keyLen)
        off += keyLen
        val xored = ByteArray(encKey.size) { (encKey[it].toInt() xor 0x64).toByte() }
        val rc4Key = aesECBDecrypt(ncmAESKey, xored)
        // 去掉前面 "neteasecloudmusic" 17 字节
        val key = if (rc4Key.size > 17) rc4Key.copyOfRange(17, rc4Key.size) else rc4Key
        return rc4XOR(key, raw.copyOfRange(off, raw.size))
    }

    private fun decryptKGM(raw: ByteArray): ByteArray {
        require(raw.size >= 4) { "文件过短" }
        val head = String(raw.copyOfRange(0, 4))
        require(head == "#KGM" || head == "KGMA") { "magic 不匹配" }
        val off = if (raw.size >= 0x40) 0x40 else 4
        val out = ByteArray(raw.size - off)
        for (i in off until raw.size) {
            out[i - off] = (raw[i].toInt() xor kgmXorTable[(i - off) % kgmXorTable.size].toInt()).toByte()
        }
        return out
    }

    private fun decryptQMC(raw: ByteArray): ByteArray {
        require(raw.size >= 8) { "文件过短" }
        require(String(raw.copyOfRange(0, 4)) == "QMC2") { "magic 不匹配" }
        val out = ByteArray(raw.size - 8)
        for (i in 8 until raw.size) {
            out[i - 8] = (raw[i].toInt() xor qmcXorKey[(i - 8) % qmcXorKey.size].toInt()).toByte()
        }
        return out
    }

    private fun decryptKWM(raw: ByteArray): ByteArray {
        require(raw.size >= 12) { "文件过短" }
        require(String(raw.copyOfRange(0, 3)) == "KWM") { "magic 不匹配" }
        val off = (raw[7].toInt() and 0xff) or
                ((raw[8].toInt() and 0xff) shl 8) or
                ((raw[9].toInt() and 0xff) shl 16) or
                ((raw[10].toInt() and 0xff) shl 24)
        val o = if (off in 11..raw.size) off else 11
        return rc4XOR(kwmRC4Key, raw.copyOfRange(o, raw.size))
    }
}
