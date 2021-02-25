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
	"log"
	"reflect"
	"unsafe"

	. "github.com/Monibuca/engine/v3"
	"github.com/Monibuca/utils/v3"
	"github.com/giorgisio/goav/avcodec"
	"github.com/giorgisio/goav/avformat"
	"github.com/giorgisio/goav/avutil"
)

var (
	config struct {
	}
	decCodecs map[string]*avcodec.Codec
	encCodecs map[string]*avcodec.Codec
	reqs      = make(map[string]map[*Stream]TransCoder)
)

type Context C.struct_AVCodecContext

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
		reqs[tc.RequestCodec] = make(map[*Stream]TransCoder)
	}
	reqmap := reqs[tc.RequestCodec]
	if _, ok := reqmap[subscriber.Stream]; !ok {
		reqmap[subscriber.Stream] = TransCoder{make(map[*Subscriber]*struct{}), nil}
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
func (tc *TransCoder) transcode(s *Stream, encCodec string) {
	tc.Subscriber = &Subscriber{
		ID: "ffmpeg",
	}
	s.Subscribe(tc.Subscriber)
	var decCtx *avcodec.Context
	var ret int
	// encCtx2:=(*avformat.CodecContext)(unsafe.Pointer(encCtx))
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
	utils.Println("decCtx.AvcodecOpen2", ret)
	encCtx := encCodecs[encCodec].AvcodecAllocContext3()
	encCtx2 := (*Context)(unsafe.Pointer(encCtx))
	encCtx2.SetSample(int32(decCtx.SampleFmt()), 8000, 1)
	ret = encCtx.AvcodecOpen2(encCodecs[encCodec], nil)
	utils.Println("encCtx.AvcodecOpen2", ret, encCtx.SampleFmt(), encCtx.SampleRate(), encCtx.Channels())
	if ret != 0 {
		log.Fatal(ret)
	}
	p := avcodec.AvPacketAlloc()
	p2 := avcodec.AvPacketAlloc()
	p2.AvInitPacket()
	frame0 := avutil.AvFrameAlloc()
	frame := (*avcodec.Frame)(unsafe.Pointer(frame0))
	// var got_output int
	at := tc.GetAudioTrack(encCodec)
	at.SoundRate = 44000
	at.SoundSize = tc.OriginAudioTrack.SoundSize
	at.SoundType = tc.OriginAudioTrack.SoundType
	switch encCodec {
	case "aac":
		at.SoundFormat = 10
		config1 := (1 << 3) | ((4 & 0xe) >> 1)
		config2 := ((4 & 0x1) << 7) | (tc.OriginAudioTrack.SoundType << 3)
		at.RtmpTag = []byte{0xAF, 0x00, byte(config1), byte(config2)}
	case "pcma":
		at.SoundFormat = 7
		tmp := at.SoundFormat<<4 | byte(3)<<3 | at.SoundSize<<1 | at.SoundType
		at.RtmpTag = []byte{tmp}
	case "pcmu":
		at.SoundFormat = 8
		tmp := at.SoundFormat<<4 | byte(3)<<3 | at.SoundSize<<1 | at.SoundType
		at.RtmpTag = []byte{tmp}
	}
	tc.OnAudio = func(pack AudioPack) {
		header := (*reflect.SliceHeader)(unsafe.Pointer(&pack.Payload))
		p.AvPacketFromData((*uint8)(unsafe.Pointer(header.Data)), header.Len)
		ret := decCtx.AvcodecSendPacket(p)
		for ret >= 0 {
			ret = decCtx.AvcodecReceiveFrame(frame)
			utils.Println("decode ret:", ret)
			ret = encCtx2.AvcodecSendFrame(frame)
			//ret = encCtx.AvcodecEncodeAudio2(p2, frame, &got_output)
			utils.Println("encode ret:", ret)
			ret = encCtx2.AvcodecReceivePacket(p2)
			utils.Println("AvcodecReceivePacket ret:", ret)
			// if got_output > 0 {
			// 	at.Push(pack.Timestamp, cdata2go(p2.Data(), p2.Size()))
			// }
			at.Push(pack.Timestamp, cdata2go(p2.Data(), p2.Size()))
		}
	}
	tc.PlayAudio(tc.OriginAudioTrack)
	p.AvFreePacket()
	p2.AvFreePacket()
}
func cdata2go(data *uint8, size int) (payload []byte) {
	p := uintptr(unsafe.Pointer(data))
	for i := 0; i < size; i++ {
		payload = append(payload, *(*byte)(unsafe.Pointer(p)))
		p++
	}
	return
}
