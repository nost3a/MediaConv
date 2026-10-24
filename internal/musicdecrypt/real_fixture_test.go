package musicdecrypt

import (
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"
)

// TestRealFixtures 依赖 test/ 目录下的 sample.mp3 及其派生加密样本
// 这些 fixture 不进 git 仓库（体积大、属测试素材）；本地要跑时：
//
//  1. 往 test/sample.mp3 放任意一段短 mp3
//  2. go run ./cmd/genfixtures   # 生成 sample.ncm/kgm/qmc/kwm
//
// 仓库里没有 fixture 时本测试自动 skip，不影响 CI。
func TestRealFixtures(t *testing.T) {
	orig, err := ioutil.ReadFile("../../test/sample.mp3")
	if err != nil {
		t.Skipf("缺少 test/sample.mp3，跳过真实 fixture 测试：%v", err)
	}
	origMD5 := md5hex(orig)
	t.Logf("原始 sample.mp3 MD5=%s size=%d", origMD5, len(orig))
	for _, ext := range []string{".ncm", ".kgm", ".qmc", ".kwm"} {
		samplePath := "../../test/sample" + ext
		if _, err := os.Stat(samplePath); err != nil {
			t.Logf("缺少 %s，跳过该格式", samplePath)
			continue
		}
		out, _, err := Decrypt(samplePath)
		if err != nil {
			t.Errorf("%s: %v", ext, err)
			continue
		}
		got, _ := ioutil.ReadFile(out)
		sum := md5.Sum(got)
		gotMD5 := hex.EncodeToString(sum[:])
		match := gotMD5 == origMD5
		t.Logf("%s: out=%s size=%d MD5=%s match=%v", ext, out, len(got), gotMD5, match)
		if !match {
			t.Errorf("%s MD5 不一致", ext)
		}
	}
}
