# MediaConv

本地音视频转换小工具，所有转码都在你自己的设备上完成，不联网、不上传。
A local media transcoder that runs entirely on your device — no cloud, no upload.

---

## 一键下载 / Quick Download

```bash
# Windows 10/11 64-bit（内嵌 ffmpeg，开箱即用）
curl -L -o mediaconv.exe https://github.com/nost3a/MediaConv/releases/latest/download/mediaconv-windows-amd64.exe

# Linux amd64 (command line)
curl -L -o mediaconv https://github.com/nost3a/MediaConv/releases/latest/download/mediaconv-linux-amd64
chmod +x mediaconv && ./mediaconv

# Linux 桌面 GUI 版
curl -L -o mediaconv-gui https://github.com/nost3a/MediaConv/releases/latest/download/mediaconv-linux-amd64-gui
chmod +x mediaconv-gui && ./mediaconv-gui

# Android
# 浏览器打开 https://github.com/nost3a/MediaConv/releases/latest 下载 APK
# Open the releases page in your browser and download MediaConv-Android.apk
```

> Windows 首次打开如被 SmartScreen 拦截，选择 **More info → Run anyway** 即可。
> On first launch, if Windows blocks the app, click **More info → Run anyway**.

---

## 各平台用法 / Usage

### Windows

双击 `MediaConv-Win64.exe` 打开主窗口，拖文件进去，选格式，点开始。
Double-click `MediaConv-Win64.exe`, drag a file in, pick a format, hit start.

ffmpeg 已内置，无需单独安装。Windows 10/11 双击即用。
ffmpeg is bundled inside, no separate install needed. Works out of the box on Windows 10/11.

### Linux (Debian / Ubuntu)

**通用 tar.gz（推荐，所有发行版可用）**
下载 `MediaConv-Linux-amd64.tar.gz`，解压后跑 `scripts/install.sh`。
Download the tarball and run `scripts/install.sh`.

**Debian/Ubuntu deb 包**
```
sudo dpkg -i mediaconv.deb
sudo apt install -f    # 自动补依赖 / auto-fix deps
```
装完后从应用菜单点 MediaConv 图标启动。
Launch MediaConv from your application menu after install.

依赖：`ffmpeg` 和 `libgl1`，发行版仓库里都有。
Depends on `ffmpeg` and `libgl1`, both in your distro's repos.

**Fedora / RHEL / CentOS / Arch / openSUSE / Alpine**
用通用 tar.gz 即可。`packaging/` 目录里有 `.spec`、`PKGBUILD`、`APKBUILD` 模板可参考。
Use the generic tarball. See `packaging/` for rpm/AUR/alpine templates.

### Android

手机浏览器打开下面链接下载 APK：
Download the APK on your phone:

https://github.com/nost3a/MediaConv/releases/latest/download/MediaConv-Android.apk

或扫描二维码 / Or scan QR code:

![download](assets/qrcode-apk.png)

装上后点图标打开，选文件 → 选格式 → 转换。完成后去系统"下载"App 的 `Download/MediaConv/` 目录拿结果。
Open the app, pick a file, choose a format, convert. Output lands in `Download/MediaConv/` on your device.

### Termux / 无桌面服务器 / CLI

```bash
# Termux
pkg install ffmpeg
chmod +x mediaconv && ./mediaconv

# 直接命令行转码 / One-shot transcode
./mediaconv convert -i video.mp4 -o out.mp3 -q 192k
./mediaconv convert -i song.ncm  -o song.mp3

# 起本地 Web 服务 / start local web UI
./mediaconv start
# 然后浏览器打开 http://127.0.0.1:8090
```

```
用法 / Usage:
  mediaconv start [--host 127.0.0.1] [--port 8090]   启动本地 Web 服务（无桌面首选）
  mediaconv up [-f 格式] [-q 音质]                     TUI 选文件→转码→TUI 选输出目录
  mediaconv convert -i IN [-f 格式] [-q 音质] [-o OUT]  命令行单次转码
  mediaconv batch -d 目录 -f 格式 [-q 音质] [--recursive] 批量转码目录
  mediaconv info <文件>                                查看媒体信息（时长/编码/分辨率）
  mediaconv list                                       列出支持的格式与音质
  mediaconv decrypt <文件.ncm/kgm/qmc/kwm>             仅还原私有音乐格式（不重编码）
  mediaconv clean                                      清理临时工作目录
  mediaconv ffmpeg                                     查看 ffmpeg/ffprobe 状态与安装提示
  mediaconv gui                                        启动原生图形界面（需 -tags gui 构建）
  mediaconv repl                                       进入交互式命令行
  mediaconv version                                    打印版本
```

> **Linux / Windows 共用同一套命令库**（源码在 `internal/cli`）。
> 行为完全一致，只有产物形态不同：Windows 版把 ffmpeg 内嵌进 exe，开箱即用；
> Linux 版使用系统 PATH 里的 ffmpeg / ffprobe。
> The same command library is shared by both Linux and Windows CLI builds.

---

## 交互式命令行 / Interactive CLI

无参运行 `mediaconv` 会打印帮助。想进交互循环用 `repl`（别名 `shell` / `sh`）：
Run `mediaconv repl` to enter an interactive loop.

```
$ ./mediaconv repl
MediaConv 1.2.0 交互模式（输入 help 查看命令，exit 退出）
mediaconv> help
mediaconv> convert -i song.wav -f mp3 -o song.mp3
mediaconv> batch -d ./videos -f mp4 -q 192k --recursive
mediaconv> exit
```

`up` 命令（别名 `upload`）专为非桌面 / 无文件管理器的环境设计，内置 **bubbletea TUI 文件选择器**：
The `up` command embeds a bubbletea TUI file picker:

1. 启动后弹出 TUI 文件管理器，让你浏览并**选择原始文件**；
2. 选好格式/音质后开始转码；
3. 转码完成再弹出 TUI，让你**选择输出目录**并命名。

```
$ ./mediaconv up -f mp3 -q 320k
[ TUI 文件管理器：选择要转换的原文件 ]
[ 转码中 ... 100% ]
[ TUI 文件管理器：选择输出保存位置 ]
已保存到 /path/to/out.mp3
```

Windows 版 `mediaconv-windows-amd64.exe` 已内置 ffmpeg，双击即用。
The Windows `mediaconv-windows-amd64.exe` ships with ffmpeg bundled, double-click to run.

---

## 音乐格式还原 / Music Format Recovery

支持把下面这些容器格式还原为通用音频（MP3 / FLAC）：
Recovers these container formats back to plain MP3 / FLAC:

| 后缀 | 来源 |
|---|---|
| `.ncm` | 网易云音乐 / NetEase Cloud Music |
| `.kgm` / `.kgma` | 酷狗 / KuGou |
| `.qmc` / `.qmcflac` / `.mflac` / `.mgg` | QQ 音乐 / QQ Music |
| `.kwm` | 酷我 / KuWo |

直接当普通文件上传或作为 `-i` 输入即可，工具会自动先还原再转。
Just pass them in like any other file — recovery happens automatically.

> 还原算法按各平台真实封装实现（仅本地、不联网）：
> Recovery follows each platform's real container spec, fully offline:
> - **NCM**：AES-128-ECB 解密钥 → RC4 解密音频流
> - **KGM / KGMA**：固定密钥 + 两段式 XOR 掩码
> - **QMC / QMCFLAC / MFLAC / MGG**：位置掩码矩阵异或
> - **KWM**：RC4 流解密
>
> 解密相关代码在 `internal/musicdecrypt/`（Go，含 round-trip 测试）；
> Android 端另有一份 Kotlin 移植在 `android/app/.../decrypt/MusicDecrypt.kt`，
> 上架前需与该 Go 版逐一对齐（见该文件注释）。

---

## 支持的格式 / Formats

音频输出：`mp3 aac m4a flac wav ogg opus wma aiff amr mka ac3`
视频输出：`mp4 m4v mkv avi mov flv webm wmv mpeg mpg 3gp ts mts vob f4v ogv gif`

任意音视频输入都可以转到上面这些输出格式。
Any audio/video input can be converted to any output listed above.

仅作为输入读取（不支持转成这些格式）：`ape tta wv mid midi ra rm rmvb asf mpc dts`
Read-only input formats (cannot be output): `ape tta wv mid midi ra rm rmvb asf mpc dts`.

---

## 安全 / Security

本工具只监听 `127.0.0.1`，仅本机访问。**有意设计为不支持对外暴露**，不要拿去公网部署。
The server binds to loopback only by design. Don't expose it to the public internet.

---

## 从源码构建 / Build from Source

需要 Go 1.23+。Linux/macOS 用系统 ffmpeg；Windows 版会把 ffmpeg 内嵌进 exe。
Requires Go 1.23+. Linux/macOS use system ffmpeg; the Windows build embeds ffmpeg.

```bash
# Linux 纯 CLI（无桌面依赖，CGO 关闭）
bash scripts/build-cli.sh          # → bin/mediaconv-linux-amd64

# Windows CLI 交叉编译（自动下载内嵌 ffmpeg.exe，162M 不进 git）
bash scripts/build-windows-cli.sh  # → bin/mediaconv-windows-amd64.exe

# 桌面 GUI 版（可选）
bash scripts/build-linux-webview.sh   # Linux + webkit2gtk-4.1
bash scripts/build-windows.sh         # Windows GUI（需 mingw-w64）
```

也可以手动编译 / Or build by hand:

```bash
# Windows 交叉编译前，先拉一次内嵌用的 ffmpeg.exe（162M，不进 git）：
bash scripts/fetch-ffmpeg.sh

# 普通编译 / Normal build
CGO_ENABLED=0 go build -o mediaconv .
./mediaconv -h
```

Linux 桌面版需要 OpenGL / webkit 开发头文件（`libgl-dev` / `libwebkit2gtk-4.1-dev`）。
The Linux desktop build needs OpenGL / webkit dev headers.

> 仓库里不含 `bin/`（预编译二进制）、`internal/ffembed/ffmpeg.exe`（162M）、`test/*.mp3/mp4`（测试素材），
> 这些都通过 `.gitignore` 排除，二进制走 GitHub Releases 分发。

---

## License

[MIT](./LICENSE) © MediaConv Contributors.

---

> This tool is for personal backup of files you have rights to. You are responsible for complying with your local copyright law.
> 本工具仅供个人备份你有权使用的文件，请遵守当地著作权法。
