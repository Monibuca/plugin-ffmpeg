package plugin_ffmpeg

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/dwdcth/ffmpeg-go/ffcommon"
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

	for _, dir := range searchDirs {
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
			filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					fmt.Println(err)
					return nil
				}
				if !info.IsDir() && exp.MatchString(info.Name()) {
					FFmpegPlugin.Info("load lib", zap.String("path", path))
					f(path)
					founds++
				}
				return nil
			})
		}
		if founds == libNum {
			return nil
		}
	}
	return nil
}
