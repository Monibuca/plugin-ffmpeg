package plugin_ffmpeg

//#cgo pkg-config: libavformat libavcodec libavutil libswresample
////#cgo LDFLAGS: -lavformat -lavutil -lavcodec -lswresample
////#cgo CFLAGS: -Wno-deprecated -I
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
	"github.com/charlestamz/goav/avcodec"
	"github.com/charlestamz/goav/avformat"
	"github.com/charlestamz/goav/avutil"
	"github.com/charlestamz/goav/swresample"
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

	//ffmpeg version below 3.0
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
	plugin := PluginConfig{
		Name:   "FFMPEG",
		Config: &config,
	}
	plugin.Install(run)
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
func onRequest(tc *TransCodeReq) {
	subscriber := tc.Subscriber
	if _, ok := reqs[tc.RequestCodec]; !ok {
		reqs[tc.RequestCodec] = make(map[*Stream]*TransCoder)
	}
	reqmap := reqs[tc.RequestCodec]
	if _, ok := reqmap[subscriber.Stream]; !ok {
		reqmap[subscriber.Stream] = &TransCoder{make(map[*Subscriber]*struct{}), nil}
	}
	if t := reqmap[subscriber.Stream]; t.request(subscriber) {
		t.transcode(subscriber.Stream, tc.RequestCodec)
	}
}
func onUnsubscribe(sub *Subscriber, count int) {
	for _, req := range reqs {
		if tc, ok := req[sub.Stream]; ok {
			tc.leave(sub)
		}
	}
}
func run() {
	go AddHookGo(HOOK_REQUEST_TRANSAUDIO, onRequest)
	AddHook(HOOK_UNSUBSCRIBE, onUnsubscribe)
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
	utils.Println("ffmpeg transcode", s.StreamPath, encCodec)
	pSwrCtx := swresample.SwrAlloc()
	//defer pSwrCtx.SwrFree()
	var decCtx *avcodec.Context
	var ret int
	audioTrack := s.WaitAudioTrack()
	switch audioTrack.CodecID {
	case 10:
		decCtx = decCodecs["aac"].AvcodecAllocContext3()
		(*avformat.CodecContext)(unsafe.Pointer(decCtx)).SetExtraData(audioTrack.ExtraData[2:])
		ret = decCtx.AvcodecOpen2(decCodecs["aac"], nil)
	case 7:
		decCtx = decCodecs["pcma"].AvcodecAllocContext3()
		ret = decCtx.AvcodecOpen2(decCodecs["pcma"], nil)
	case 8:
		decCtx = decCodecs["pcmu"].AvcodecAllocContext3()
		ret = decCtx.AvcodecOpen2(decCodecs["pcmu"], nil)
	default:
		utils.Printf("transCode not support CodecID :%d", audioTrack.CodecID)
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

	at := tc.NewAudioTrack(0)
	at.SoundRate = audioTrack.SoundRate
	at.SoundSize = audioTrack.SoundSize
	at.Channels = audioTrack.Channels

	var encodeFormat int32
	switch encCodec {
	case "aac":
		at.CodecID = 10
		config1 := (1 << 3) | ((4 & 0xe) >> 1)
		config2 := ((4 & 0x1) << 7) | (at.Channels << 3)
		at.ExtraData = []byte{0xAF, 0x00, byte(config1), byte(config2)}
	case "pcma":
		at.CodecID = 7
		at.SoundRate = 8000
		at.SoundSize = 1
		at.Channels = 1
		encodeFormat = 1
		tmp := at.CodecID<<4 | byte(3)<<2 | at.SoundSize<<1 | (at.Channels - 1)
		at.ExtraData = []byte{tmp}
	case "pcmu":
		at.CodecID = 8
		at.SoundRate = 8000
		at.SoundSize = 1
		at.Channels = 1
		encodeFormat = 1
		tmp := at.CodecID<<4 | byte(3)<<2 | at.SoundSize<<1 | (at.Channels - 1)
		at.ExtraData = []byte{tmp}
	}
	rational := avutil.NewRational(1, at.SoundRate)
	encCtx.SetTimeBase(rational)
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
	utils.Println("swr:", soundType2Layout(at.Channels), swresample.AvSampleFormat(encCtx.SampleFmt()), encCtx.SampleRate(), soundType2Layout(audioTrack.Channels), swresample.AvSampleFormat(decCtx.SampleFmt()), audioTrack.SoundRate)
	pSwrCtx.SwrAllocSetOpts(soundType2Layout(at.Channels), swresample.AvSampleFormat(encCtx.SampleFmt()), encCtx.SampleRate(), soundType2Layout(audioTrack.Channels), swresample.AvSampleFormat(decCtx.SampleFmt()), audioTrack.SoundRate, 0, 0)
	if ret = pSwrCtx.SwrInit(); ret != 0 {
		utils.Println("pSwrCtx.SwrInit:", ret)
		return
	}
	frame0 := avutil.AvFrameAlloc()
	defer avutil.AvFrameFree(frame0)
	frame := (*avutil.Frame)(unsafe.Pointer(frame0))
	frame1 := (*swresample.Frame)(unsafe.Pointer(frame0))
	frameOut0 := avutil.AvFrameAlloc()
	defer avutil.AvFrameFree(frameOut0)
	(*Frame)(unsafe.Pointer(frameOut0)).SetSample(int(encodeFormat), encCtx.SampleRate(), soundType2Layout(at.Channels))
	frameOut := (*avutil.Frame)(unsafe.Pointer(frameOut0))
	frameOut1 := (*swresample.Frame)(unsafe.Pointer(frameOut0))
	tc.OnAudio = func(timestamp uint32, pack *AudioPack) {
		header := (*reflect.SliceHeader)(unsafe.Pointer(&pack.Payload))
		p.AvPacketFromData((*uint8)(unsafe.Pointer(header.Data)), header.Len)
		var gp int
		for ret := decCtx.SendPacket(p); ret >= 0; {
			if ret = decCtx.ReceiveFrame(frame); ret < 0 {
				utils.Println("Decoding error ret:", ret)
				return
			}
			if ret = pSwrCtx.SwrConvertFrame(frameOut1, frame1); ret < 0 {
				utils.Println("SwrConvertFrame ret:", ret)
			}
			if ret = encCtx.SendFrame(frameOut); ret < 0 {
				utils.Println("Encoding error ret:", ret)
			}

			ret = encCtx.ReceivePacket(p2)
			if gp != 0 {
				at.PushRaw(timestamp, cdata2go(p2.Data(), p2.Size()))
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
	tc.PlayAudio(audioTrack)

}
func cdata2go(data *uint8, size int) (payload []byte) {
	p := uintptr(unsafe.Pointer(data))
	for i := 0; i < size; i++ {
		payload = append(payload, *(*byte)(unsafe.Pointer(p)))
		p++
	}
	return
}
