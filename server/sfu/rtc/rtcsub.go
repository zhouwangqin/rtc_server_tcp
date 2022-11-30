package rtc

import (
	"errors"
	"io"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
	"github.com/zhuanxin-sz/go-protoo/logger"
)

const (
	maxRTCPChanSize = 100
)

// Sub 拉流对象
type Sub struct {
	Id    string
	stop  bool
	alive bool
	pc    *webrtc.PeerConnection

	writeErrCnt int
	TrackAudio  *webrtc.RTPSender
	TrackVideo  *webrtc.RTPSender
	RtcpAudioCh chan rtcp.Packet
	RtcpVideoCh chan rtcp.Packet
}

// NewSub 新建Sub对象
func NewSub(sid string) (*Sub, error) {
	cfg := webrtc.Configuration{
		ICEServers:         iceServers,
		ICETransportPolicy: webrtc.ICETransportPolicyAll,
		SDPSemantics:       webrtc.SDPSemanticsUnifiedPlanWithFallback,
	}

	engine := webrtc.MediaEngine{}
	engine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
	engine.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))

	setting := webrtc.SettingEngine{}
	if icePortStart != 0 && icePortEnd != 0 {
		setting.SetEphemeralUDPPortRange(icePortStart, icePortEnd)
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(engine), webrtc.WithSettingEngine(setting))
	pcnew, err := api.NewPeerConnection(cfg)
	if err != nil {
		logger.Errorf("sub new peer err=%v, sid=%s", err, sid)
		return nil, err
	}

	sub := &Sub{
		Id:          sid,
		pc:          pcnew,
		stop:        false,
		alive:       true,
		writeErrCnt: 0,
		TrackAudio:  nil,
		TrackVideo:  nil,
		RtcpAudioCh: make(chan rtcp.Packet, maxRTCPChanSize),
		RtcpVideoCh: make(chan rtcp.Packet, maxRTCPChanSize),
	}

	pcnew.OnConnectionStateChange(sub.OnPeerConnect)
	return sub, nil
}

// OnPeerConnect Sub连接状态回调
func (sub *Sub) OnPeerConnect(state webrtc.PeerConnectionState) {
	if state == webrtc.PeerConnectionStateConnected {
		logger.Debugf("sub peer connected = %s", sub.Id)
		sub.alive = true
		go sub.DoVideoRtcp()
	}
	if state == webrtc.PeerConnectionStateDisconnected {
		logger.Debugf("sub peer disconnected = %s", sub.Id)
		sub.alive = false
	}
	if state == webrtc.PeerConnectionStateFailed {
		logger.Debugf("sub peer failed = %s", sub.Id)
		sub.alive = false
	}
}

// Close 关闭Sub
func (sub *Sub) Close() {
	logger.Debugf("sub close = %s", sub.Id)
	sub.stop = true
	sub.pc.Close()
	close(sub.RtcpAudioCh)
	close(sub.RtcpVideoCh)
}

// AddTrack 增加Track
func (sub *Sub) AddTrack(remoteTrack *webrtc.Track) error {
	track, err := sub.pc.NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), remoteTrack.ID(), remoteTrack.Label())
	if err != nil {
		logger.Errorf("sub new track err=%v, sid=%s", err, sub.Id)
		return err
	}

	sender, err := sub.pc.AddTrack(track)
	if err != nil {
		logger.Errorf("sub add track err=%v, sid=%s", err, sub.Id)
		return err
	}

	if remoteTrack.Kind() == webrtc.RTPCodecTypeAudio {
		sub.TrackAudio = sender
	}
	if remoteTrack.Kind() == webrtc.RTPCodecTypeVideo {
		sub.TrackVideo = sender
	}
	return nil
}

// Answer SDP交换
func (sub *Sub) Answer(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	err := sub.pc.SetRemoteDescription(offer)
	if err != nil {
		logger.Errorf("sub set answer err=%v, sid=%s", err, sub.Id)
		return webrtc.SessionDescription{}, err
	}

	sdp, err := sub.pc.CreateAnswer(nil)
	if err != nil {
		logger.Errorf("sub create answer err=%v, sid=%s", err, sub.Id)
		return webrtc.SessionDescription{}, err
	}

	err = sub.pc.SetLocalDescription(sdp)
	if err != nil {
		logger.Errorf("sub set answer err=%v, sid=%s", err, sub.Id)
		return webrtc.SessionDescription{}, err
	}
	return sdp, nil
}

// DoAudioRtcp 接收音频RTCP包
func (sub *Sub) DoAudioRtcp() {
	if sub.TrackAudio != nil {
		for {
			if sub.stop || !sub.alive {
				return
			}

			rtcps, err := sub.TrackAudio.ReadRTCP()
			if err != nil {
				if err == io.EOF {
					sub.alive = false
				}
			} else {
				for _, rtcp := range rtcps {
					if sub.stop || !sub.alive {
						return
					}
					sub.RtcpAudioCh <- rtcp
				}
			}
		}
	}
}

// DoVideoRtcp 接收视频RTCP包
func (sub *Sub) DoVideoRtcp() {
	if sub.TrackVideo != nil {
		for {
			if sub.stop || !sub.alive {
				return
			}

			rtcps, err := sub.TrackVideo.ReadRTCP()
			if err != nil {
				if err == io.EOF {
					sub.alive = false
				}
			} else {
				for _, rtcp := range rtcps {
					if sub.stop || !sub.alive {
						return
					}
					sub.RtcpVideoCh <- rtcp
				}
			}
		}
	}
}

// ReadAudioRTCP 读音频RTCP包
func (sub *Sub) ReadAudioRTCP() (rtcp.Packet, error) {
	pkt, ok := <-sub.RtcpAudioCh
	if !ok {
		return nil, errors.New("audio rtcp chan close")
	}
	return pkt, nil
}

// ReadVideoRTCP 读视频RTCP包
func (sub *Sub) ReadVideoRTCP() (rtcp.Packet, error) {
	pkt, ok := <-sub.RtcpVideoCh
	if !ok {
		return nil, errors.New("video rtcp chan close")
	}
	return pkt, nil
}

// WriteAudioRtp 写音频包
func (sub *Sub) WriteAudioRtp(pkt *rtp.Packet) error {
	if sub.TrackAudio != nil && sub.TrackAudio.Track() != nil && !sub.stop && sub.alive {
		return sub.TrackAudio.Track().WriteRTP(pkt)
	}
	return errors.New("sub audio track is nil or peer not connect")
}

// WriteVideoRtp 写视频包
func (sub *Sub) WriteVideoRtp(pkt *rtp.Packet) error {
	if sub.TrackVideo != nil && sub.TrackVideo.Track() != nil && !sub.stop && sub.alive {
		return sub.TrackVideo.Track().WriteRTP(pkt)
	}
	return errors.New("sub video track is nil or peer not connect")
}

// WriteErrTotal return write error
func (sub *Sub) WriteErrTotal() int {
	return sub.writeErrCnt
}

// WriteErrReset reset write error
func (sub *Sub) WriteErrReset() {
	sub.writeErrCnt = 0
}

// WriteErrAdd write error
func (sub *Sub) WriteErrAdd() {
	sub.writeErrCnt++
}
