# test/ — 测试素材目录

本目录下的媒体 fixture（mp3 / mp4 / ncm / kgm / qmc / kwm 等）**不进 git 仓库**：
体积大、属测试素材，推上去会把仓库撑爆。

## 本地要跑真实解密测试时

```bash
# 1. 随便拿一段几秒钟的短 mp3 放到这里，命名为 sample.mp3
#    例如：cp ~/Music/some-short.mp3 test/sample.mp3

# 2. 用项目自带的一次性工具反向加密出 4 种私有格式样本
go run ./cmd/genfixtures
# 会在本目录生成：sample.ncm / sample.kgm / sample.qmc / sample.kwm

# 3. 跑测试
go test ./internal/musicdecrypt/
```

仓库里没有这些文件时，`TestRealFixtures` 会自动 `t.Skip`，不影响 CI。
