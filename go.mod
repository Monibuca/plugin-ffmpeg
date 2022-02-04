module github.com/Monibuca/plugin-ffmpeg

go 1.13

require (
	github.com/Monibuca/engine/v3 v3.4.5
	github.com/Monibuca/utils/v3 v3.0.5
	github.com/charlestamz/goav v1.5.4
)

//replace github.com/charlestamz/goav => ../goav
//
//replace github.com/Monibuca/engine/v3 => ../engine
