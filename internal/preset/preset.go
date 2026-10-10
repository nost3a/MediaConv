// Package preset 内置转码预设表
// Built-in transcoder presets
package preset

// Preset 一条转码预设：输入后缀 + 目标格式 + 音质 -> ffmpeg 参数
type Preset struct {
	Category    string
	InExts      []string
	OutFmt      string
	AudioCodec  string
	VideoCodec  string
	ExtraArgs   []string
	QualityArgs map[string][]string
	InputOnly   bool // 仅支持作为输入，不支持作为输出
}

// FormatItem 给前端列格式表
type FormatItem struct {
	Category  string   `json:"category"`
	OutFmt    string   `json:"out_fmt"`
	InExts    []string `json:"in_exts"`
	Qualities []string `json:"qualities"`
	Desc      string   `json:"desc"`
	InputOnly bool     `json:"input_only,omitempty"`
}

// 所有音频后缀（输入）
var audioIns = []string{"mp3", "flac", "wav", "aac", "m4a", "ogg", "wma", "opus", "ape", "aiff", "amr", "mka", "tta", "wv", "ac3", "dts"}

// 所有视频后缀（输入）
var videoIns = []string{"mp4", "mkv", "avi", "mov", "flv", "webm", "wmv", "mpg", "mpeg", "3gp", "ts", "mts", "vob", "m4v", "f4v", "ogv", "gif"}

// 视频抽音频用的输入
var avIns = []string{"mp4", "mkv", "avi", "mov", "flv", "webm", "3gp", "wmv", "mpg", "mpeg", "ts", "mts", "m4v", "f4v", "ogv"}

// 音频输出预设表
type audioOut struct {
	fmt        string
	codec      string
	extra      []string
	hasBitrate bool
}

var audioOuts = []audioOut{
	{"mp3", "libmp3lame", nil, true},
	{"aac", "aac", nil, true},
	{"m4a", "aac", nil, true},
	{"flac", "flac", nil, false},
	{"wav", "pcm_s16le", nil, false},
	{"ogg", "libvorbis", nil, true},
	{"opus", "libopus", nil, true},
	{"wma", "wmav2", nil, true},
	{"ape", "ape", nil, false},
	{"aiff", "pcm_s16le", []string{"-f", "aiff"}, false},
	{"amr", "amr_nb", []string{"-ar", "8000", "-ac", "1"}, false},
	{"mka", "copy", nil, false},
	{"mpc", "libmpc", nil, true},
	{"tta", "tta", nil, false},
	{"wv", "wavpack", nil, false},
	{"ac3", "ac3", nil, true},
	{"dts", "dca", nil, true},
}

// 视频输出预设表
type videoOut struct {
	fmt       string
	vcodec    string
	acodec    string
	extra     []string
}

var videoOuts = []videoOut{
	{"mp4", "libx264", "aac", []string{"-preset", "veryfast", "-crf", "23"}},
	{"m4v", "libx264", "aac", []string{"-preset", "veryfast", "-crf", "23"}},
	{"mkv", "libx264", "aac", []string{"-preset", "veryfast", "-crf", "23"}},
	{"avi", "mpeg4", "libmp3lame", []string{"-crf", "23"}},
	{"mov", "libx264", "aac", []string{"-preset", "veryfast", "-crf", "23"}},
	{"flv", "flv1", "mp3", nil},
	{"webm", "libvpx-vp9", "libopus", []string{"-deadline", "realtime", "-cpu-used", "4"}},
	{"wmv", "wmv2", "wmav2", nil},
	{"mpeg", "mpeg2video", "mp2", nil},
	{"mpg", "mpeg2video", "mp2", nil},
	{"3gp", "libx264", "aac", []string{"-f", "3gp"}},
	{"ts", "libx264", "aac", []string{"-f", "mpegts"}},
	{"mts", "libx264", "ac3", nil},
	{"vob", "mpeg2video", "ac3", nil},
	{"f4v", "libx264", "aac", []string{"-f", "flv"}},
	{"ogv", "libtheora", "libvorbis", nil},
	{"gif", "", "", []string{"-vf", "fps=10,scale=480:-1:flags=lanczos"}},
}

// 仅输入格式（不支持作为输出）
var inputOnly = []string{"mid", "midi", "ra", "rm", "rmvb", "asf"}

// 构建预设表
func buildPresets() []Preset {
	out := []Preset{}
	qualities := []string{"128k", "192k", "320k"}

	for _, a := range audioOuts {
		qa := map[string][]string{}
		if a.hasBitrate {
			for _, q := range qualities {
				qa[q] = []string{"-b:a", q}
			}
		}
		// 视频 -> 音频
		out = append(out, Preset{
			Category: "video2audio", InExts: avIns, OutFmt: a.fmt,
			AudioCodec: a.codec, ExtraArgs: a.extra, QualityArgs: qa,
		})
		// 音频 -> 音频
		out = append(out, Preset{
			Category: "audio2audio", InExts: audioIns, OutFmt: a.fmt,
			AudioCodec: a.codec, ExtraArgs: a.extra, QualityArgs: qa,
		})
	}

	for _, v := range videoOuts {
		out = append(out, Preset{
			Category: "video2video", InExts: videoIns, OutFmt: v.fmt,
			AudioCodec: v.acodec, VideoCodec: v.vcodec, ExtraArgs: v.extra,
			QualityArgs: map[string][]string{
				"original": {},
				"720p":     {"-vf", "scale=1280:-2"},
				"1080p":    {"-vf", "scale=1920:-2"},
			},
		})
	}

	// 仅输入格式（占位，让 Find 识别为不支持输出）
	for _, f := range inputOnly {
		out = append(out, Preset{
			Category: "input-only", OutFmt: f, InputOnly: true,
		})
	}
	return out
}

var presets = buildPresets()

// Find 找匹配预设
func Find(inExt, outFmt, quality string) (*Preset, bool) {
	for i := range presets {
		p := &presets[i]
		if p.OutFmt != outFmt || p.InputOnly {
			continue
		}
		ok := false
		for _, e := range p.InExts {
			if e == inExt {
				ok = true
				break
			}
		}
		if !ok {
			continue
		}
		if len(p.QualityArgs) > 0 {
			if _, hit := p.QualityArgs[quality]; !hit {
				for k := range p.QualityArgs {
					quality = k
					break
				}
			}
		}
		return p, true
	}
	return nil, false
}

// BuildArgs 翻译为 ffmpeg 参数
func (p *Preset) BuildArgs(quality string, outPath string) []string {
	args := []string{}
	if p.VideoCodec == "" && p.Category != "video2audio" && p.Category != "audio2audio" {
		// 视频->视频 但没指定 vcodec（比如 gif），走 extra
		if len(p.ExtraArgs) > 0 {
			args = append(args, p.ExtraArgs...)
		}
	} else if p.VideoCodec == "" {
		args = append(args, "-vn")
	} else {
		args = append(args, "-c:v", p.VideoCodec)
		args = append(args, p.ExtraArgs...)
	}
	if p.AudioCodec != "" {
		args = append(args, "-c:a", p.AudioCodec)
	}
	if qa, ok := p.QualityArgs[quality]; ok {
		args = append(args, qa...)
	}
	args = append(args, outPath)
	return args
}

// ListFormats 给前端列全部格式
func ListFormats() []FormatItem {
	out := []FormatItem{}
	seen := map[string]bool{}
	for i := range presets {
		p := &presets[i]
		if p.InputOnly {
			out = append(out, FormatItem{
				Category: "input-only", OutFmt: p.OutFmt, InputOnly: true,
				Desc: p.OutFmt + " (仅支持作为输入)",
			})
			continue
		}
		key := p.Category + "|" + p.OutFmt
		if seen[key] {
			continue
		}
		seen[key] = true
		q := []string{}
		for k := range p.QualityArgs {
			q = append(q, k)
		}
		out = append(out, FormatItem{
			Category: p.Category, OutFmt: p.OutFmt, InExts: p.InExts,
			Qualities: q, Desc: p.Category + " -> " + p.OutFmt,
		})
	}
	return out
}
