package plugin_ffmpeg

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/dwdcth/ffmpeg-go/v6/ffcommon"
	"go.uber.org/zap"
	. "m7s.live/engine/v4"
)

type FFmpegConfig struct {
	LibPath string `desc:"ffmpeg lib path"`
}

var conf FFmpegConfig

var FFmpegPlugin = InstallPlugin(&conf)

func (conf *FFmpegConfig) OnEvent(event any) {
	switch event.(type) {
	case FirstConfig:
		err := SetAvLib()
		if err != nil {
			FFmpegPlugin.Error("set av lib error", zap.Error(err))
		}
	}
}

func SetAvLib() error {
	fnMap := map[string]func(path string){
		"avutil":     ffcommon.SetAvutilPath,
		"avcodec":    ffcommon.SetAvcodecPath,
		"avdevice":   ffcommon.SetAvdevicePath,
		"avfilter":   ffcommon.SetAvfilterPath,
		"avformat":   ffcommon.SetAvformatPath,
		"postproc":   ffcommon.SetAvpostprocPath,
		"swresample": ffcommon.SetAvswresamplePath,
		"swscale":    ffcommon.SetAvswscalePath,
	}
	libNum := len(fnMap)
	var searchDirs []string
	if conf.LibPath != "" {
		searchDirs = append(searchDirs, conf.LibPath)
	}
	switch runtime.GOOS {
	case "darwin":
		searchDirs = append(searchDirs, ".", "/usr/local/lib", "/usr/lib")
	case "windows":
		searchDirs = append(searchDirs, ".", "C:\\Windows\\System32")
	default:
		searchDirs = append(searchDirs, ".", "/usr/local/lib", "/usr/lib", "/usr/lib/x86_64-linux-gnu")
	}

	for i, dir := range searchDirs {
		var founds int
		for k, f := range fnMap {
			var exp *regexp.Regexp
			switch runtime.GOOS {
			case "darwin":
				exp = regexp.MustCompile("lib" + k + `(\.\d+)?\.dylib`)
			case "windows":
				exp = regexp.MustCompile(k + `(-\d+)?\.dll`)
			default:
				exp = regexp.MustCompile(k + `(\.\d+)?\.so`)
			}
			if i == 0 {
				filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
					if strings.HasPrefix(path, ".git") {
						return nil
					}
					FFmpegPlugin.Debug("search lib", zap.String("path", path))
					if err != nil {
						FFmpegPlugin.Error("search lib error", zap.Error(err))
						return nil
					}
					if !info.IsDir() && exp.MatchString(info.Name()) {
						FFmpegPlugin.Info("load lib", zap.String("path", path))
						f(path)
						founds++
					}
					return nil
				})
			} else {
				// 遍历一层目录
				files, err := os.ReadDir(dir)
				if err != nil {
					FFmpegPlugin.Error("read dir error", zap.Error(err))
					continue
				}
				for _, file := range files {
					if !file.IsDir() && exp.MatchString(file.Name()) {
						FFmpegPlugin.Info("load lib", zap.String("path", filepath.Join(dir, file.Name())))
						f(filepath.Join(dir, file.Name()))
						founds++
					}
				}
			}
		}
		if founds == libNum {
			return nil
		}
	}
	return nil
}
