package plugin_ffmpeg

import (
	"github.com/charlestamz/goav/avcodec"
	. "m7s.live/engine/v4"
)

type FFmpegConfig struct {
}

var (
	config    FFmpegConfig
	decCodecs map[string]*avcodec.Codec
	encCodecs map[string]*avcodec.Codec
	// reqs      = make(map[string]map[*Stream]*TransCoder)
)

func init() {

	//ffmpeg version below 3.0
	avcodec.AvcodecRegisterAll()
	decCodecs = map[string]*avcodec.Codec{
		"aac":  avcodec.AvcodecFindDecoder(avcodec.CodecId(avcodec.AV_CODEC_ID_AAC)),
		"pcma": avcodec.AvcodecFindDecoder(avcodec.CodecId(avcodec.AV_CODEC_ID_PCM_ALAW)),
		"pcmu": avcodec.AvcodecFindDecoder(avcodec.CodecId(avcodec.AV_CODEC_ID_PCM_MULAW)),
		"h264": avcodec.AvcodecFindDecoder(avcodec.CodecId(avcodec.AV_CODEC_ID_H264)),
		"h265": avcodec.AvcodecFindDecoder(avcodec.CodecId(avcodec.AV_CODEC_ID_HEVC)),
	}
	encCodecs = map[string]*avcodec.Codec{
		"aac":  avcodec.AvcodecFindEncoder(avcodec.CodecId(avcodec.AV_CODEC_ID_AAC)),
		"pcma": avcodec.AvcodecFindEncoder(avcodec.CodecId(avcodec.AV_CODEC_ID_PCM_ALAW)),
		"pcmu": avcodec.AvcodecFindEncoder(avcodec.CodecId(avcodec.AV_CODEC_ID_PCM_MULAW)),
		"h264": avcodec.AvcodecFindEncoder(avcodec.CodecId(avcodec.AV_CODEC_ID_H264)),
		"h265": avcodec.AvcodecFindEncoder(avcodec.CodecId(avcodec.AV_CODEC_ID_HEVC)),
	}
}

var FFmpegPlugin = InstallPlugin(&config)

func (conf *FFmpegConfig) OnEvent(event any) {
	switch event.(type) {
	case SEpublish:

	}
}

// type TransCoder struct {
// 	requests map[*Subscriber]*struct{}
// 	*Subscriber
// }

// func (tc *TransCoder) request(subscriber *Subscriber) (needRun bool) {
// 	if len(tc.requests) == 0 {
// 		needRun = true
// 	}
// 	tc.requests[subscriber] = nil
// 	return
// }

// func (tc *TransCoder) leave(subscriber *Subscriber) {
// 	delete(tc.requests, subscriber)
// 	if len(tc.requests) == 0 {
// 		tc.Close()
// 	}
// }
// func onRequest(tc *TransCodeReq) {
// 	subscriber := tc.Subscriber
// 	if _, ok := reqs[tc.RequestCodec]; !ok {
// 		reqs[tc.RequestCodec] = make(map[*Stream]*TransCoder)
// 	}
// 	reqmap := reqs[tc.RequestCodec]
// 	if _, ok := reqmap[subscriber.Stream]; !ok {
// 		reqmap[subscriber.Stream] = &TransCoder{make(map[*Subscriber]*struct{}), nil}
// 	}
// 	if t := reqmap[subscriber.Stream]; t.request(subscriber) {
// 		t.transcode(subscriber.Stream, tc.RequestCodec)
// 	}
// }
// func onUnsubscribe(sub *Subscriber, count int) {
// 	for _, req := range reqs {
// 		if tc, ok := req[sub.Stream]; ok {
// 			tc.leave(sub)
// 		}
// 	}
// }
// func run() {
// 	AddHookGo(HOOK_REQUEST_TRANSAUDIO, onRequest)
// 	AddHook(HOOK_UNSUBSCRIBE, onUnsubscribe)
// }
// func soundType2Layout(st byte) uint64 {
// 	if st == 1 {
// 		return 1
// 	} else {
// 		return 3
// 	}
// }

// func (tc *TransCoder) transcode(s *Stream, encCodec string) {
// 	tc.Subscriber = &Subscriber{
// 		ID: "ffmpeg",
// 	}
// 	s.Subscribe(tc.Subscriber)
// 	utils.Println("ffmpeg transcode", s.StreamPath, encCodec)
// 	pSwrCtx := swresample.SwrAlloc()
// 	defer pSwrCtx.SwrFree()
// 	var decCtx *avcodec.Context
// 	var ret int
// 	audioTrack := s.WaitAudioTrack()
// 	switch audioTrack.CodecID {
// 	case 10:
// 		decCtx = decCodecs["aac"].AvcodecAllocContext3()
// 		(*avformat.CodecContext)(unsafe.Pointer(decCtx)).SetExtraData(audioTrack.ExtraData[2:])
// 		ret = decCtx.AvcodecOpen2(decCodecs["aac"], nil)
// 	case 7:
// 		decCtx = decCodecs["pcma"].AvcodecAllocContext3()
// 		ret = decCtx.AvcodecOpen2(decCodecs["pcma"], nil)
// 	case 8:
// 		decCtx = decCodecs["pcmu"].AvcodecAllocContext3()
// 		ret = decCtx.AvcodecOpen2(decCodecs["pcmu"], nil)
// 	default:
// 		utils.Printf("transCode not support CodecID :%d", audioTrack.CodecID)
// 		tc.Close()
// 		return
// 	}
// 	rational := avutil.NewRational(1, audioTrack.SoundRate)
// 	decCtx.SetSampleRate(audioTrack.SoundRate)
// 	decCtx.SetChannels(int(audioTrack.Channels))
// 	decCtx.SetChannelLayout(soundType2Layout(audioTrack.Channels))
// 	decCtx.SetTimeBase(rational)
// 	decCtx.SetDebug(0xFFFFFFFF)
// 	defer decCtx.AvcodecClose()
// 	if ret != 0 {
// 		utils.Println("decCtx.AvcodecOpen2", ret)
// 		return
// 	}
// 	encCtx := encCodecs[encCodec].AvcodecAllocContext3()
// 	//defer encCtx.AvcodecFreeContext()

// 	p := avcodec.AvPacketAlloc()
// 	p2 := avcodec.AvPacketAlloc()
// 	p2.AvInitPacket()
// 	defer avcodec.AvPacketFree(p)
// 	defer avcodec.AvPacketFree(p2)

// 	at := tc.NewAudioTrack(0)
// 	at.SoundRate = audioTrack.SoundRate
// 	at.SoundSize = audioTrack.SoundSize
// 	at.Channels = audioTrack.Channels

// 	var encodeFormat int32
// 	switch encCodec {
// 	case "aac":
// 		at.CodecID = 10
// 		config1 := (1 << 3) | ((4 & 0xe) >> 1)
// 		config2 := ((4 & 0x1) << 7) | (at.Channels << 3)
// 		at.ExtraData = []byte{0xAF, 0x00, byte(config1), byte(config2)}
// 	case "pcma":
// 		at.CodecID = 7
// 		at.SoundRate = 8000
// 		at.SoundSize = 1
// 		at.Channels = 1
// 		encodeFormat = 1
// 		tmp := at.CodecID<<4 | byte(3)<<2 | at.SoundSize<<1 | (at.Channels - 1)
// 		at.ExtraData = []byte{tmp}
// 	case "pcmu":
// 		at.CodecID = 8
// 		at.SoundRate = 8000
// 		at.SoundSize = 1
// 		at.Channels = 1
// 		encodeFormat = 1
// 		tmp := at.CodecID<<4 | byte(3)<<2 | at.SoundSize<<1 | (at.Channels - 1)
// 		at.ExtraData = []byte{tmp}
// 	}
// 	encCtx.SetTimeBase(rational)
// 	// decCtx2 := (*Context)(unsafe.Pointer(decCtx))
// 	// decCtx2.SetSample(encodeFormat, tc.OriginAudioTrack.SoundRate, int(tc.OriginAudioTrack.SoundType+1))
// 	encCtx.SetSampleRate(at.SoundRate)
// 	//fmt := swresample.AvSampleFormat(decCtx.SampleFmt())
// 	encCtx.SetSampleFmt(*(*avcodec.AvSampleFormat)(unsafe.Pointer(&encodeFormat)))
// 	encCtx.SetChannels(int(at.Channels))
// 	encCtx.SetChannelLayout(soundType2Layout(at.Channels))

// 	if ret = encCtx.AvcodecOpen2(encCodecs[encCodec], nil); ret != 0 {
// 		utils.Println("encCtx.AvcodecOpen2 error:", ret)
// 		return
// 	}
// 	defer encCtx.AvcodecClose()

// 	pSwrCtx.SwrAllocSetOpts(int64(soundType2Layout(at.Channels)), swresample.AvSampleFormat(encCtx.SampleFmt()), at.SoundRate,
// 		int64(soundType2Layout(audioTrack.Channels)), swresample.AvSampleFormat(decCtx.SampleFmt()), audioTrack.SoundRate, 0, 0)
// 	if ret = pSwrCtx.SwrInit(); ret != 0 {
// 		utils.Println("pSwrCtx.SwrInit error:", ret)
// 		return
// 	}
// 	frameIn := avutil.AvFrameAlloc()
// 	defer avutil.AvFrameFree(frameIn)
// 	frameInSw := (*swresample.Frame)(unsafe.Pointer(frameIn))
// 	frameOut := avutil.AvFrameAlloc()
// 	defer avutil.AvFrameFree(frameOut)
// 	frameOut.SetSampleRate(encCtx.SampleRate())
// 	frameOut.SetChannelLayout(soundType2Layout(at.Channels))
// 	frameOut.SetFormat(int(encodeFormat))
// 	frameOutSw := (*swresample.Frame)(unsafe.Pointer(frameOut))
// 	tc.OnAudio = func(timestamp uint32, pack *AudioPack) {
// 		header := (*reflect.SliceHeader)(unsafe.Pointer(&pack.Raw))
// 		p.AvPacketFromData((*uint8)(unsafe.Pointer(header.Data)), header.Len)
// 		for ret := decCtx.SendPacket(p); ret >= 0; {
// 			if ret = decCtx.ReceiveFrame(frameIn); ret < 0 {
// 				if ret == avutil.AVERROR_EAGAIN {
// 					//utils.Println("EAGAIN", ret)
// 				}
// 				return
// 			}

// 			if ret = pSwrCtx.SwrConvertFrame(frameOutSw, frameInSw); ret < 0 {
// 				utils.Println("SwrConvertFrame ret:", ret)
// 				return
// 			}

// 			//utils.Println("SwrConvertFrame  ret:", ret)
// 			if ret = encCtx.SendFrame(frameOut); ret < 0 {
// 				utils.Println("Encoding error ret:", ret)
// 				return
// 			}

// 			if ret = encCtx.ReceivePacket(p2); ret == 0 {
// 				at.PushRaw(timestamp, cdata2go(p2.Data(), p2.Size()))
// 			}

// 		}
// 	}
// 	tc.PlayAudio(audioTrack)

// }
// func cdata2go(data *uint8, size int) (payload []byte) {
// 	// slice example
// 	const vmax = math.MaxInt32 / unsafe.Sizeof(payload[0])
// 	payload = (*[vmax]byte)(unsafe.Pointer(data))[:size:size]
// 	return
// }
