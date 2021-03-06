package plugin_ffmpeg

//#cgo pkg-config: libavformat libavcodec libavutil libswresample
//#include <stdio.h>
//#include <stdlib.h>
//#include <inttypes.h>
//#include <stdint.h>
//#include <string.h>
//#include <libavformat/avformat.h>
//#include <libavcodec/avcodec.h>
//#include <libavutil/avutil.h>
import "C"
import (
	"reflect"
	"unsafe"

	. "github.com/Monibuca/engine/v3"
	"github.com/Monibuca/utils/v3"
	"github.com/giorgisio/goav/avcodec"
	"github.com/giorgisio/goav/avformat"
	"github.com/giorgisio/goav/avutil"
	"github.com/giorgisio/goav/swresample"
)

/*
enum AVSampleFormat {
    AV_SAMPLE_FMT_NONE = -1,
    AV_SAMPLE_FMT_U8,          ///< unsigned 8 bits
    AV_SAMPLE_FMT_S16,         ///< signed 16 bits
    AV_SAMPLE_FMT_S32,         ///< signed 32 bits
    AV_SAMPLE_FMT_FLT,         ///< float
    AV_SAMPLE_FMT_DBL,         ///< double

    AV_SAMPLE_FMT_U8P,         ///< unsigned 8 bits, planar
    AV_SAMPLE_FMT_S16P,        ///< signed 16 bits, planar
    AV_SAMPLE_FMT_S32P,        ///< signed 32 bits, planar
    AV_SAMPLE_FMT_FLTP,        ///< float, planar
    AV_SAMPLE_FMT_DBLP,        ///< double, planar
    AV_SAMPLE_FMT_S64,         ///< signed 64 bits
    AV_SAMPLE_FMT_S64P,        ///< signed 64 bits, planar

    AV_SAMPLE_FMT_NB           ///< Number of sample formats. DO NOT USE if linking dynamically
};
*/
var (
	config struct {
	}
	decCodecs map[string]*avcodec.Codec
	encCodecs map[string]*avcodec.Codec
	reqs      = make(map[string]map[*Stream]*TransCoder)
)

type (
	Frame   C.struct_AVFrame
	Context C.struct_AVCodecContext
)

func (c *Frame) SetSample(fmt int, rate int, cl int64) {
	c.format = C.int(fmt)
	c.sample_rate = C.int(rate)
	c.channel_layout = C.uint64_t(cl)
}
func (c *Context) SetSample(fmt int32, rate int, channels int) {
	c.sample_fmt = fmt
	c.sample_rate = C.int(rate)
	c.channels = C.int(channels)
}
func (ctxt *Context) AvcodecReceivePacket(packet *avcodec.Packet) int {
	return (int)(C.avcodec_receive_packet((*C.struct_AVCodecContext)(ctxt), (*C.struct_AVPacket)(unsafe.Pointer(packet))))
}

func (ctxt *Context) AvcodecSendFrame(frame *avcodec.Frame) int {
	return (int)(C.avcodec_send_frame((*C.struct_AVCodecContext)(ctxt), (*C.struct_AVFrame)(unsafe.Pointer(frame))))
}

func init() {
	avformat.AvRegisterAll()
	avcodec.AvcodecRegisterAll()
	decCodecs = map[string]*avcodec.Codec{
		"aac":  avcodec.AvcodecFindDecoder(avcodec.CodecId(avcodec.AV_CODEC_ID_AAC)),
		"pcma": avcodec.AvcodecFindDecoder(avcodec.CodecId(avcodec.AV_CODEC_ID_PCM_ALAW)),
		"pcmu": avcodec.AvcodecFindDecoder(avcodec.CodecId(avcodec.AV_CODEC_ID_PCM_MULAW)),
	}
	encCodecs = map[string]*avcodec.Codec{
		"aac":  avcodec.AvcodecFindEncoder(avcodec.CodecId(avcodec.AV_CODEC_ID_AAC)),
		"pcma": avcodec.AvcodecFindEncoder(avcodec.CodecId(avcodec.AV_CODEC_ID_PCM_ALAW)),
		"pcmu": avcodec.AvcodecFindEncoder(avcodec.CodecId(avcodec.AV_CODEC_ID_PCM_MULAW)),
	}
	HasTranscoder = true
	InstallPlugin(&PluginConfig{
		Name:   "FFMPEG",
		Config: &config,
		Run:    run,
	})
}

type TransCoder struct {
	requests map[*Subscriber]*struct{}
	*Subscriber
}

func (tc *TransCoder) request(subscriber *Subscriber) (needRun bool) {
	if len(tc.requests) == 0 {
		needRun = true
	}
	tc.requests[subscriber] = nil
	return
}

func (tc *TransCoder) leave(subscriber *Subscriber) {
	delete(tc.requests, subscriber)
	if len(tc.requests) == 0 {
		tc.Close()
	}
}
func onRequest(req interface{}) {
	tc := req.(*TransCodeReq)
	subscriber := tc.Subscriber
	if _, ok := reqs[tc.RequestCodec]; !ok {
		reqs[tc.RequestCodec] = make(map[*Stream]*TransCoder)
	}
	reqmap := reqs[tc.RequestCodec]
	if _, ok := reqmap[subscriber.Stream]; !ok {
		reqmap[subscriber.Stream] = &TransCoder{make(map[*Subscriber]*struct{}), nil}
	}
	t := reqmap[subscriber.Stream]
	if t.request(subscriber) {
		go t.transcode(subscriber.Stream, tc.RequestCodec)
	}
}
func onUnsubscribe(sub interface{}) {
	s := sub.(*Subscriber)
	for _, req := range reqs {
		if tc, ok := req[s.Stream]; ok {
			tc.leave(s)
		}
	}
}
func run() {
	go AddHook(HOOK_REQUEST_TRANSAUDIO, onRequest)
	go AddHook(HOOK_UNSUBSCRIBE, onUnsubscribe)
}
func soundType2Layout(st byte) int64 {
	if st == 1 {
		return 1
	} else {
		return 3
	}
}
func (tc *TransCoder) transcode(s *Stream, encCodec string) {
	tc.Subscriber = &Subscriber{
		ID: "ffmpeg",
	}
	s.Subscribe(tc.Subscriber)
	pSwrCtx := swresample.SwrAlloc()
	//defer pSwrCtx.SwrFree()
	var decCtx *avcodec.Context
	var ret int
	switch s.OriginAudioTrack.SoundFormat {
	case 10:
		decCtx = decCodecs["aac"].AvcodecAllocContext3()
		(*avformat.CodecContext)(unsafe.Pointer(decCtx)).SetExtraData(s.OriginAudioTrack.RtmpTag[2:])
		ret = decCtx.AvcodecOpen2(decCodecs["aac"], nil)
	case 7:
		decCtx = decCodecs["pcma"].AvcodecAllocContext3()
		ret = decCtx.AvcodecOpen2(decCodecs["pcma"], nil)
	case 8:
		decCtx = decCodecs["pcmu"].AvcodecAllocContext3()
		ret = decCtx.AvcodecOpen2(decCodecs["pcmu"], nil)
	default:
		utils.Printf("transCode not support soundformat :%d", s.OriginAudioTrack.SoundFormat)
		tc.Close()
		return
	}
	//defer decCtx.AvcodecFreeContext()
	defer decCtx.AvcodecClose()
	if ret != 0 {
		utils.Println("decCtx.AvcodecOpen2", ret)
		return
	}
	encCtx := encCodecs[encCodec].AvcodecAllocContext3()
	//defer encCtx.AvcodecFreeContext()

	p := avcodec.AvPacketAlloc()
	p2 := avcodec.AvPacketAlloc()
	p2.AvInitPacket()
	//defer p.AvFreePacket()
	//defer p2.AvFreePacket()
	at := tc.GetAudioTrack(encCodec)
	at.SoundRate = tc.OriginAudioTrack.SoundRate
	at.SoundSize = tc.OriginAudioTrack.SoundSize
	at.Channels = tc.OriginAudioTrack.Channels
	var encodeFormat int32
	switch encCodec {
	case "aac":
		at.SoundFormat = 10
		config1 := (1 << 3) | ((4 & 0xe) >> 1)
		config2 := ((4 & 0x1) << 7) | (at.Channels << 3)
		at.RtmpTag = []byte{0xAF, 0x00, byte(config1), byte(config2)}
	case "pcma":
		at.SoundFormat = 7
		at.SoundRate = 8000
		at.SoundSize = 1
		at.Channels = 1
		encodeFormat = 1
		tmp := at.SoundFormat<<4 | byte(3)<<2 | at.SoundSize<<1 | (at.Channels - 1)
		at.RtmpTag = []byte{tmp}
	case "pcmu":
		at.SoundFormat = 8
		at.SoundRate = 8000
		at.SoundSize = 1
		at.Channels = 1
		encodeFormat = 1
		tmp := at.SoundFormat<<4 | byte(3)<<2 | at.SoundSize<<1 | (at.Channels - 1)
		at.RtmpTag = []byte{tmp}
	}
	encCtx.SetTimebase(1, at.SoundRate)
	// decCtx2 := (*Context)(unsafe.Pointer(decCtx))
	// decCtx2.SetSample(encodeFormat, tc.OriginAudioTrack.SoundRate, int(tc.OriginAudioTrack.SoundType+1))
	encCtx2 := (*Context)(unsafe.Pointer(encCtx))
	encCtx2.SetSample(encodeFormat, at.SoundRate, int(at.Channels))
	ret = encCtx.AvcodecOpen2(encCodecs[encCodec], nil)
	if ret != 0 {
		utils.Println("encCtx.AvcodecOpen2:", ret)
		return
	}
	defer encCtx.AvcodecClose()
	utils.Println("swr:", soundType2Layout(at.Channels), swresample.AvSampleFormat(encCtx.SampleFmt()), encCtx.SampleRate(), soundType2Layout(tc.OriginAudioTrack.Channels), swresample.AvSampleFormat(decCtx.SampleFmt()), tc.OriginAudioTrack.SoundRate)
	pSwrCtx.SwrAllocSetOpts(soundType2Layout(at.Channels), swresample.AvSampleFormat(encCtx.SampleFmt()), encCtx.SampleRate(), soundType2Layout(tc.OriginAudioTrack.Channels), swresample.AvSampleFormat(decCtx.SampleFmt()), tc.OriginAudioTrack.SoundRate, 0, 0)
	ret = pSwrCtx.SwrInit()
	if ret != 0 {
		utils.Println("pSwrCtx.SwrInit:", ret)
		return
	}
	frame0 := avutil.AvFrameAlloc()
	defer avutil.AvFrameFree(frame0)
	frame := (*avcodec.Frame)(unsafe.Pointer(frame0))
	frame1 := (*swresample.Frame)(unsafe.Pointer(frame0))
	frameOut0 := avutil.AvFrameAlloc()
	defer avutil.AvFrameFree(frameOut0)
	(*Frame)(unsafe.Pointer(frameOut0)).SetSample(int(encodeFormat), encCtx.SampleRate(), soundType2Layout(at.Channels))
	frameOut := (*avcodec.Frame)(unsafe.Pointer(frameOut0))
	frameOut1 := (*swresample.Frame)(unsafe.Pointer(frameOut0))
	tc.OnAudio = func(pack AudioPack) {
		header := (*reflect.SliceHeader)(unsafe.Pointer(&pack.Payload))
		p.AvPacketFromData((*uint8)(unsafe.Pointer(header.Data)), header.Len)
		var gp int
		for ret := decCtx.AvcodecSendPacket(p); ret >= 0; {
			if ret = decCtx.AvcodecReceiveFrame(frame); ret < 0 {
				return
			}
			ret2 := pSwrCtx.SwrConvertFrame(frameOut1, frame1)
			if ret2 != 0 {
				utils.Println("SwrConvertFrame ret:", ret2)
			}
			ret2 = encCtx.AvcodecEncodeAudio2(p2, frameOut, &gp)
			if gp != 0 {

				at.Push(pack.Timestamp, cdata2go(p2.Data(), p2.Size()))
			}
			// if ret2 = encCtx2.AvcodecSendFrame(frameOut); ret2 != 0 {
			// 	utils.Println("encode ret:", ret2)
			// }
			// for ret2 >= 0 {
			// 	if ret2 = encCtx2.AvcodecReceivePacket(p2); ret2 < 0 {
			// 		break
			// 	}
			// }
		}
	}
	tc.PlayAudio(tc.OriginAudioTrack)

}
func cdata2go(data *uint8, size int) (payload []byte) {
	p := uintptr(unsafe.Pointer(data))
	for i := 0; i < size; i++ {
		payload = append(payload, *(*byte)(unsafe.Pointer(p)))
		p++
	}
	return
}
