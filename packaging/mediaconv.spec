Name:           mediaconv
Version:        1.0.0
Release:        1
Summary:        Local media transcoder
License:        MIT
URL:            https://github.com/nost3a/MediaConv
BuildArch:      x86_64

Requires:       ffmpeg
Requires:       libGL

%description
本地音视频转码工具，不联网不上传。
Local media transcoder, no cloud.

%install
mkdir -p %{buildroot}/usr/bin
install -m 755 bin/mediaconv-linux-amd64 %{buildroot}/usr/bin/mediaconv
install -m 755 bin/mconv-linux-amd64 %{buildroot}/usr/bin/mconv
install -m 755 bin/MediaConv-Linux-amd64-cli %{buildroot}/usr/bin/mediaconv-cli

%files
/usr/bin/mediaconv
/usr/bin/mconv
/usr/bin/mediaconv-cli

%changelog
* Wed Oct 07 2026 MediaConv Contributors - 1.0.0-1
- Initial release
