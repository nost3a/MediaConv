# 发布到 GitHub 操作手册（Edge 翻译中文版）

你用 Edge 浏览器，页面会自动翻译成中文。下面所有按钮名**按中文界面写**，括号里标英文以防翻译抽风。

---

## 第 0 步：为什么不能一个 zip 全传？

GitHub 硬规定：**单个文件超过 100MB，git push 直接被拒**。
你之前"推到一半断了"就是这个原因，不是网络问题。

所以分两步：
- **git push** 只传源码（639KB，秒传）
- **发行版（Release）** 上传 exe/apk/deb，就在你仓库页面右边，不是另一个网站

---

## 第 1 步：建空仓库

1. 打开 https://github.com/new （Edge 会翻译成"新建仓库"）
2. **仓库名称** 填：`MediaConv`
3. 下面三个勾选框**全部不勾**（添加 README、添加 .gitignore、添加许可协议）
4. 点绿色按钮 **Create repository**（创建仓库）

> 如果你的 GitHub 用户名不是 `nost3a`，先告诉我，我把 README 里的网址改成你的。

---

## 第 2 步：push 源码

在本地 `mediaconv/` 文件夹里打开终端，执行：

```bash
cd mediaconv

git init
git add .
git commit -m "v1.0.0"

# 下面这行把 nost3a 换成你自己的 GitHub 用户名
git remote add origin https://github.com/nost3a/MediaConv.git

git branch -M main
git push -u origin main
```

这一步只传 639KB，几秒钟完事。

---

## 第 3 步：上传二进制文件（你问的"直链放哪"就在这）

### 3.1 找到入口

1. 打开你的仓库页面，网址大概长这样：
   `https://github.com/你的用户名/MediaConv`
2. 看页面**右边那一边**，有一列小字链接，找到 **发行版**（英文 Releases）
   - 位置大概在：仓库主页 → 右侧栏 → 有个 "About" 框下面，或者看页面最右边
   - 如果找不到，直接在网址后面加 `/releases` 回车，比如：
     `https://github.com/你的用户名/MediaConv/releases`
3. 进去后点右上角的绿色按钮 **草拟新发行版**（英文 Draft a new release）
   - 或者可能写着 **创建新发布** / **发布新版本**

### 3.2 选标签

1. 页面上有个输入框写着 **选择标记**（英文 Choose a tag），点它
2. 在弹出的输入框里输入：`v1.0.0`
3. 点下面出现的选项 **创建新标记：v1.0.0**（英文 Create new tag: v1.0.0 on publish）

### 3.3 填标题

- **发行版标题**（Release title）填：`MediaConv v1.0.0`
- 下面大框留空就行，或者随便写两句

### 3.4 上传文件（关键步骤）

1. 拉到页面最下面，有个虚线框写着：
   **通过将二进制文件拖放到此处或选择它们来附加文件**
   （英文 Attach binaries by dropping them here or selecting them）
2. 打开你电脑上的 `mediaconv/bin/` 文件夹
3. 把下面这些文件**用鼠标拖进那个虚线框**：

**必传（README 里的下载按钮指向它们）：**

| 文件 | 大概大小 |
|---|---|
| `MediaConv-Win64.exe` | 171 MB |
| `MediaConv-Linux-amd64-cli` | 8.6 MB |
| `MediaConv-Linux-amd64.deb` | 16 MB |
| `MediaConv-Android.apk` | 107 MB |

**建议也传（其他平台）：**

| 文件 | 大概大小 |
|---|---|
| `MediaConv-Win32.exe` | 171 MB |
| `MediaConv-v1.0.0-linux-amd64.tar.gz` | 27 MB |
| `mconv-windows-amd64.exe` | 171 MB |
| `MediaConv-Termux-arm64` | 8.3 MB |

4. 等上传进度条走完（107MB 的 APK 和 171MB 的 exe 要传几分钟，别急）

### 3.5 发布

1. 拉到页面最下面，点绿色按钮 **发布发行版**（英文 Publish release）
   - 旁边有个小框 "Set as pre-release"（设为预发行版）**不要勾**

---

## 第 4 步：验证直链能用

发布完后，把下面网址里的"你的用户名"换成你自己的，在浏览器打开，应该直接开始下载：

```
https://github.com/你的用户名/MediaConv/releases/latest/download/MediaConv-Win64.exe
https://github.com/你的用户名/MediaConv/releases/latest/download/MediaConv-Android.apk
https://github.com/你的用户名/MediaConv/releases/latest/download/MediaConv-Linux-amd64.deb
```

能下载就说明 README 里的直链对用户生效了。

---

## 找不到按钮怎么办？

Edge 偶尔翻译抽风，按钮名对不上。两个办法：

1. **临时关翻译**：点地址栏右边那个翻译图标（一般是个"译"字或地球图标）→ 点 **显示原始版本** / **Show original**，页面就变回英文，按括号里的英文找
2. **看图找位置**：不管文字是什么，找这些视觉特征：
   - 绿色按钮 = 主操作按钮
   - 虚线框 = 上传区域
   - 页面右侧栏 = 发行版入口

---

## 以后更新版本怎么办？

改完代码 push 后，再点 **草拟新发行版**，tag 填 `v1.0.1`，重新拖文件。
README 里写的是 `releases/latest/download/`，永远指向最新版，**不用改 README**。
