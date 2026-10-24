package com.mediaconv.app

import android.Manifest
import android.app.Activity
import android.app.AlertDialog
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.Environment
import android.provider.MediaStore
import android.webkit.JavascriptInterface
import android.webkit.ValueCallback
import android.webkit.WebChromeClient
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.Toast
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import androidx.core.content.FileProvider
import com.arthenica.ffmpegkit.FFmpegKit
import com.arthenica.ffmpegkit.ReturnCode
import com.mediaconv.app.decrypt.MusicDecrypt
import java.io.File
import java.io.FileOutputStream

// ============================================================================
// 待 SDK 验证 / 已知架构问题（已修复的代码级 bug 见下方标注）：
//
// 1. 前端 assets/index.html 走的是 `fetch('/api/upload'|'/api/convert'|...)`
//    服务端模式，但本 Activity 以 file:///android_asset 直接加载该页面，
//    WebView 内并没有本地 HTTP 服务 —— 该前端路径在 Android 上实际跑不通。
//    两种可行的收敛方向（二选一）：
//      (a) Bridge 模式：前端改为调用 NativeBridge.pickFile/convert/stageUri，
//          由本类直接驱动 ffmpeg-kit，不再依赖 /api/*。
//      (b) Server 模式：在 App 内嵌 MediaConv 本地服务（已交叉编译 android/arm64
//          的可执行 + 同一份 index.html），WebView 访问 http://127.0.0.1:端口。
//    目前 index.html 与 NativeBridge 两套方案并存、互不对齐，需统一后再上架。
//
// 2. 已修复的代码级 bug（无需改设计即可生效）：
//    - convert() 此前在 outFile.delete() 之后才读取 outFile.length()，
//      回调给前端的文件大小恒为 0；现已在 delete 前捕获 outSize。
//    - pickFile() 此前把 content:// URI 直接回传给 MediaConvOnPicked，
//      随后 convert() 用 File(contentUri) 打开必然失败；现已先落地为
//      真实缓存文件、回传绝对路径，并在转换后清理临时落地文件。
//
// 3. MusicDecrypt.kt 的 Kotlin 算法需与 internal/musicdecrypt 的 Go 版
//    （真实 NCM/AES-ECB+RC4、KGM 两阶段 XOR、KWM RC4、QMC 掩码）逐一对齐，
//    当前 Kotlin 版为早期简化版，可能不一致。
// ============================================================================

class MainActivity : Activity() {
    private lateinit var wv: WebView
    private var pickedUri: Uri? = null
    private var pickedName: String = ""
    // WebView <input type=file> 回调
    // WebView file chooser callback
    private var filePathCallback: ValueCallback<Array<Uri>>? = null

    inner class NativeBridge {
        @JavascriptInterface
        fun pickFile() {
            runOnUiThread {
                val i = Intent(Intent.ACTION_OPEN_DOCUMENT).apply {
                    addCategory(Intent.CATEGORY_OPENABLE)
                    type = "*/*"
                }
                startActivityForResult(i, 1)
            }
        }

        @JavascriptInterface
        fun convert(inputPath: String, outFmt: String, quality: String) {
            Thread {
                try {
                    // 若传入的是 content:// URI，先落地成真实文件再交给 ffmpeg
                    val inFile = if (inputPath.startsWith("content://")) {
                        val staged = File(cacheDir, "stage_in_${System.currentTimeMillis()}")
                        contentResolver.openInputStream(Uri.parse(inputPath))?.use { ins ->
                            FileOutputStream(staged).use { ins.copyTo(it) }
                        }
                        staged
                    } else {
                        File(inputPath)
                    }
                    val outFile = File(cacheDir, "out.$outFmt")
                    if (outFile.exists()) outFile.delete()

                    val args = buildFfmpegArgs(inFile.absolutePath, outFile.absolutePath, outFmt, quality)
                    val cmd = args.joinToString(" ")
                    val session = FFmpegKit.execute(cmd)
                    if (ReturnCode.isSuccess(session.returnCode)) {
                        val outSize = outFile.length() // 必须在 delete 之前读取
                        // 复制到公共 Download/MediaConv/
                        val savedPath = saveToPublicDownloads(outFile, outFmt)
                        outFile.delete()
                        if (inFile != File(inputPath)) inFile.delete() // 清理可能的临时落地文件
                        runOnUiThread {
                            wv.evaluateJavascript("window.MediaConvOnDone && MediaConvOnDone('$savedPath', $outSize)", null)
                            showResultDialog(savedPath, outFmt)
                        }
                    } else {
                        runOnUiThread {
                            wv.evaluateJavascript("window.MediaConvOnError && MediaConvOnError('转码失败')", null)
                            Toast.makeText(this@MainActivity, "转码失败", Toast.LENGTH_LONG).show()
                        }
                    }
                } catch (e: Exception) {
                    runOnUiThread {
                        wv.evaluateJavascript("window.MediaConvOnError && MediaConvOnError('${e.message}')", null)
                    }
                }
            }.start()
        }

        @JavascriptInterface
        fun stageUri(uriStr: String): String {
            val uri = Uri.parse(uriStr)
            val name = queryName(uri) ?: "in.dat"
            pickedName = name
            val ext = name.substringAfterLast('.', "")
            val dst = File(cacheDir, "stage_$name")
            contentResolver.openInputStream(uri).use { input ->
                FileOutputStream(dst).use { input!!.copyTo(it) }
            }
            if (MusicDecrypt.isPrivateExt(name)) {
                val raw = dst.readBytes()
                val decrypted = MusicDecrypt.decrypt(raw, ".$ext")
                val dec = File(cacheDir, "decrypted.mp3")
                dec.writeBytes(decrypted)
                return dec.absolutePath
            }
            return dst.absolutePath
        }
    }

    // 把 cacheDir 里的临时产物写到 Download/MediaConv/
    private fun saveToPublicDownloads(tmp: File, outFmt: String): String {
        val baseName = pickedName.substringBeforeLast('.', "output")
        val outName = "${baseName}.$outFmt"
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            val values = android.content.ContentValues().apply {
                put(MediaStore.Downloads.DISPLAY_NAME, outName)
                put(MediaStore.Downloads.RELATIVE_PATH, "${Environment.DIRECTORY_DOWNLOADS}/MediaConv")
                put(MediaStore.Downloads.MIME_TYPE, mimeOf(outFmt))
            }
            val uri = contentResolver.insert(MediaStore.Downloads.EXTERNAL_CONTENT_URI, values)!!
            contentResolver.openOutputStream(uri)!!.use { out ->
                tmp.inputStream().use { it.copyTo(out) }
            }
            "Download/MediaConv/$outName"
        } else {
            @Suppress("DEPRECATION")
            val dir = File(Environment.getExternalStoragePublicDirectory(Environment.DIRECTORY_DOWNLOADS), "MediaConv")
            dir.mkdirs()
            val dst = File(dir, outName)
            tmp.inputStream().use { input ->
                FileOutputStream(dst).use { input.copyTo(it) }
            }
            "Download/MediaConv/$outName"
        }
    }

    private fun mimeOf(fmt: String): String = when (fmt) {
        "mp3" -> "audio/mpeg"
        "aac" -> "audio/aac"
        "flac" -> "audio/flac"
        "wav" -> "audio/wav"
        "ogg" -> "audio/ogg"
        "m4a" -> "audio/mp4"
        "opus" -> "audio/ogg"
        "mp4" -> "video/mp4"
        "webm" -> "video/webm"
        "mkv" -> "video/x-matroska"
        else -> "application/octet-stream"
    }

    private fun showResultDialog(savedPath: String, outFmt: String) {
        AlertDialog.Builder(this)
            .setTitle("转换完成")
            .setMessage("已保存到 $savedPath")
            .setPositiveButton("打开下载文件夹") { _, _ ->
                // 弹出文件管理器选择器 / Show file manager chooser
                val intent = Intent(Intent.ACTION_VIEW).apply {
                    setDataAndType(Uri.parse(Environment.DIRECTORY_DOWNLOADS), "resource/folder")
                    addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                }
                val chooser = Intent.createChooser(intent, "选择文件管理器 / Choose file manager")
                try {
                    startActivity(chooser)
                } catch (e: Exception) {
                    startActivity(Intent(android.app.DownloadManager.ACTION_VIEW_DOWNLOADS))
                }
            }
            .setNegativeButton("关闭", null)
            .show()
    }

    private fun queryName(uri: Uri): String? {
        contentResolver.query(uri, arrayOf(android.provider.OpenableColumns.DISPLAY_NAME), null, null, null)?.use {
            if (it.moveToFirst()) return it.getString(0)
        }
        return null
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // Android ≤9 需要运行时申请写公共目录权限
        if (Build.VERSION.SDK_INT <= Build.VERSION_CODES.P) {
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.WRITE_EXTERNAL_STORAGE)
                != PackageManager.PERMISSION_GRANTED) {
                ActivityCompat.requestPermissions(this, arrayOf(Manifest.permission.WRITE_EXTERNAL_STORAGE), 1)
            }
        }
        // Android 13+ 读媒体需要运行时权限
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            val need = mutableListOf<String>()
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.READ_MEDIA_AUDIO)
                != PackageManager.PERMISSION_GRANTED) need += Manifest.permission.READ_MEDIA_AUDIO
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.READ_MEDIA_VIDEO)
                != PackageManager.PERMISSION_GRANTED) need += Manifest.permission.READ_MEDIA_VIDEO
            if (need.isNotEmpty()) ActivityCompat.requestPermissions(this, need.toTypedArray(), 2)
        }
        wv = WebView(this)
        wv.settings.javaScriptEnabled = true
        wv.settings.domStorageEnabled = true
        wv.settings.allowFileAccess = true
        wv.settings.allowContentAccess = true
        wv.addJavascriptInterface(NativeBridge(), "NativeBridge")
        wv.webViewClient = object : WebViewClient() {
            override fun onPageFinished(view: WebView?, url: String?) {}
        }
        // 关键：必须设置 WebChromeClient，否则 <input type=file> 点了没反应
        // Must set WebChromeClient, otherwise <input type=file> does nothing
        wv.webChromeClient = object : WebChromeClient() {
            override fun onShowFileChooser(
                webView: WebView?,
                callback: ValueCallback<Array<Uri>>?,
                fileChooserParams: FileChooserParams?
            ): Boolean {
                filePathCallback?.onReceiveValue(null)
                filePathCallback = callback
                val i = Intent(Intent.ACTION_GET_CONTENT).apply {
                    addCategory(Intent.CATEGORY_OPENABLE)
                    type = "*/*"
                }
                try {
                    startActivityForResult(Intent.createChooser(i, "选择文件"), 100)
                } catch (e: Exception) {
                    filePathCallback = null
                    Toast.makeText(this@MainActivity, "无法打开文件选择器", Toast.LENGTH_SHORT).show()
                    return false
                }
                return true
            }
        }
        setContentView(wv)
        wv.loadUrl("file:///android_asset/index.html")
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        // <input type=file> 回调
        if (requestCode == 100) {
            val cb = filePathCallback ?: return
            filePathCallback = null
            val results: Array<Uri>? = if (resultCode == RESULT_OK && data?.data != null) {
                arrayOf(data.data!!)
            } else null
            cb.onReceiveValue(results)
            return
        }
        if (requestCode == 1 && resultCode == RESULT_OK) {
            data?.data?.let { uri ->
                pickedUri = uri
                val name = queryName(uri) ?: "unknown"
                pickedName = name
                // 落地成真实文件，回传路径而非 content URI（convert 需要真实路径）
                val staged = File(cacheDir, "stage_$name")
                contentResolver.openInputStream(uri)?.use { ins ->
                    FileOutputStream(staged).use { ins.copyTo(it) }
                }
                wv.evaluateJavascript("window.MediaConvOnPicked && MediaConvOnPicked('${staged.absolutePath}', '$name')", null)
            }
        }
    }

    private fun buildFfmpegArgs(inPath: String, outPath: String, outFmt: String, quality: String): List<String> {
        val a = mutableListOf("-y", "-i", inPath)
        when (outFmt) {
            "mp3" -> a += listOf("-vn", "-c:a", "libmp3lame", "-b:a", quality)
            "aac" -> a += listOf("-vn", "-c:a", "aac", "-b:a", quality)
            "m4a" -> a += listOf("-vn", "-c:a", "aac", "-b:a", quality)
            "flac" -> a += listOf("-vn", "-c:a", "flac")
            "wav" -> a += listOf("-vn", "-c:a", "pcm_s16le")
            "ogg" -> a += listOf("-vn", "-c:a", "libvorbis", "-b:a", quality)
            "opus" -> a += listOf("-vn", "-c:a", "libopus", "-b:a", quality)
            "wma" -> a += listOf("-vn", "-c:a", "wmav2", "-b:a", quality)
            "aiff" -> a += listOf("-vn", "-c:a", "pcm_s16le", "-f", "aiff")
            "ac3" -> a += listOf("-vn", "-c:a", "ac3", "-b:a", quality)
            "mp4" -> a += listOf("-c:v", "libx264", "-preset", "veryfast", "-crf", "23", "-c:a", "aac")
            "m4v" -> a += listOf("-c:v", "libx264", "-preset", "veryfast", "-crf", "23", "-c:a", "aac")
            "mkv" -> a += listOf("-c:v", "libx264", "-preset", "veryfast", "-crf", "-c:a", "aac")
            "avi" -> a += listOf("-c:v", "mpeg4", "-crf", "23", "-c:a", "libmp3lame")
            "mov" -> a += listOf("-c:v", "libx264", "-preset", "veryfast", "-crf", "23", "-c:a", "aac")
            "flv" -> a += listOf("-c:v", "flv1", "-c:a", "mp3")
            "webm" -> a += listOf("-c:v", "libvpx-vp9", "-deadline", "realtime", "-c:a", "libopus")
            "wmv" -> a += listOf("-c:v", "wmv2", "-c:a", "wmav2")
            "mpeg", "mpg" -> a += listOf("-c:v", "mpeg2video", "-c:a", "mp2")
            "ts" -> a += listOf("-c:v", "libx264", "-c:a", "aac", "-f", "mpegts")
            "gif" -> a += listOf("-vf", "fps=10,scale=480:-1:flags=lanczos")
            else -> a += listOf("-c", "copy")
        }
        a += outPath
        return a
    }
}
