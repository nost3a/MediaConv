# MediaConv

本地音视频转换小工具，所有转码都在你自己的设备上完成，不联网、不上传。
A local media transcoder that runs entirely on your device — no cloud, no upload.

---

## 一键下载 / Quick Download

```bash
# Windows 10/11 64-bit
curl -L -o MediaConv-Win64.exe https://github.com/nost3a/MediaConv/releases/latest/download/MediaConv-Win64.exe

# Linux amd64 (command line)
curl -L -o mediaconv https://github.com/nost3a/MediaConv/releases/latest/download/MediaConv-Linux-amd64-cli
chmod +x mediaconv && ./mediaconv

# Debian / Ubuntu desktop
curl -L -o mediaconv.deb https://github.com/nost3a/MediaConv/releases/latest/download/MediaConv-Linux-amd64.deb
sudo dpkg -i mediaconv.deb

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
  mediaconv start                    启动 GUI 或本地 Web 服务
  mediaconv convert -i IN -o OUT     命令行直接转码
  mediaconv list-formats             列出支持格式
  mediaconv version                  打印版本
```

---

## 交互式命令行 / Interactive CLI

`mconv` 是一个带菜单的交互版，适合不想记命令的用户。
`mconv` is an interactive menu-driven version for users who don't want to memorize flags.

```
$ ./mconv-linux-amd64
MediaConv  v1.0.0
本地音视频转换工具 / Local media transcoder

输入 help 查看命令 / Type help for commands.

mconv> up song.mp4
已加载 song.mp4
Loaded song.mp4

可转换格式 / Available formats:
  "1"  mp4
  "2"  mkv
  "3"  avi
  ...

请输入编号 / Enter number: 1
转换为 mp4 / Converting to mp4
请输入保存路径 / Enter save path:
> /tmp/output.mp4
转换成功 / Conversion success!
已保存到 /tmp/output.mp4

mconv> up song.ncm
已加载 song.ncm
Loaded song.ncm
...（自动先还原再转 / auto-decrypts first）

mconv> help
mconv> exit
```

命令 / Commands:

| 命令 | 作用 |
|---|---|
| `up <文件>` | 加载文件并列出格式菜单 / Load file and show format menu |
| `dl` | 清理临时文件 / Clean temp files |
| `help` | 显示帮助 / Show help |
| `exit` | 退出 / Quit |

Windows 版 `mconv-windows-amd64.exe` 已内置 ffmpeg，双击即用。
The Windows `mconv-windows-amd64.exe` ships with ffmpeg bundled, double-click to run.

---

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
# Windows 交叉编译前，先拉一次内嵌用的 ffmpeg.exe（162M，不进 git）：
bash scripts/fetch-ffmpeg.sh

# 普通编译 / Normal build
go build -o mediaconv .
./mediaconv -h
```

Linux 桌面版需要 OpenGL 开发头文件（`libgl-dev`）。
The Linux desktop build needs OpenGL dev headers (`libgl-dev`).

> 仓库里不含 `bin/`（预编译二进制）、`internal/ffembed/ffmpeg.exe`（162M）、`test/*.mp3/mp4`（测试素材），
> 这些都通过 `.gitignore` 排除，二进制走 GitHub Releases 分发。

---

## License

[MIT](./LICENSE) © MediaConv Contributors.

---

> This tool is for personal backup of files you have rights to. You are responsible for complying with your local copyright law.
> 本工具仅供个人备份你有权使用的文件，请遵守当地著作权法。
