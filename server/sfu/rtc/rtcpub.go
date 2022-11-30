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
	maxRTPChanSize = 100
)

type Pub struct {
	Id    string
	stop  bool
	alive bool
	pc    *webrtc.PeerConnection

	TrackAudio *webrtc.RTPReceiver
	TrackVideo *webrtc.RTPReceiver
	RtpAudioCh chan *rtp.Packet
	RtpVideoCh chan *rtp.Packet
}

func NewPub(pid string) (*Pub, error) {
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
		logger.Errorf("pub new peer err=%v, pubid=%s", err, pid)
		return nil, err
	}

	_, err = pcnew.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	if err != nil {
		logger.Errorf("pub add audio recv err=%v, pubid=%s", err, pid)
		pcnew.Close()
		return nil, err
	}

	_, err = pcnew.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RtpTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly})
	if err != nil {
		logger.Errorf("pub add video recv err=%v, pubid=%s", err, pid)
		pcnew.Close()
		return nil, err
	}

	pub := &Pub{
		Id:         pid,
		pc:         pcnew,
		stop:       false,
		alive:      true,
		TrackAudio: nil,
		TrackVideo: nil,
		RtpAudioCh: make(chan *rtp.Packet, maxRTPChanSize),
		RtpVideoCh: make(chan *rtp.Packet, maxRTPChanSize),
	}

	pcnew.OnConnectionStateChange(pub.OnPeerConnect)
	pcnew.OnTrack(pub.OnTrackRemote)
	return pub, nil
}

// OnPeerConnect Pub连接状态回调
func (pub *Pub) OnPeerConnect(state webrtc.PeerConnectionState) {
	if state == webrtc.PeerConnectionStateConnected {
		logger.Debugf("pub peer connected = %s", pub.Id)
		pub.alive = true
	}
	if state == webrtc.PeerConnectionStateDisconnected {
		logger.Debugf("pub peer disconnected = %s", pub.Id)
		pub.alive = false
	}
	if state == webrtc.PeerConnectionStateFailed {
		logger.Debugf("pub peer failed = %s", pub.Id)
		pub.alive = false
	}
}

// OnTrackRemote 接受到track回调
func (pub *Pub) OnTrackRemote(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
	if track.Kind() == webrtc.RTPCodecTypeAudio {
		pub.TrackAudio = receiver
		logger.Debugf("OnTrackRemote pub audio = %s", pub.Id)
		go pub.DoAudioRtp()
	}
	if track.Kind() == webrtc.RTPCodecTypeVideo {
		pub.TrackVideo = receiver
		logger.Debugf("OnTrackRemote pub video = %s", pub.Id)
		go pub.DoVideoRtp()
	}
}

// Close 关闭连接
func (pub *Pub) Close() {
	logger.Debugf("pub close = %s", pub.Id)
	pub.stop = true
	pub.pc.Close()
	close(pub.RtpAudioCh)
	close(pub.RtpVideoCh)
}

// Answer SDP交换
func (pub *Pub) Answer(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	err := pub.pc.SetRemoteDescription(offer)
	if err != nil {
		logger.Errorf("pub set offer err=%v, pubid=%s", err, pub.Id)
		return webrtc.SessionDescription{}, err
	}

	answer, err := pub.pc.CreateAnswer(nil)
	if err != nil {
		logger.Errorf("pub create answer err=%v, pubid=%s", err, pub.Id)
		return webrtc.SessionDescription{}, err
	}

	err = pub.pc.SetLocalDescription(answer)
	if err != nil {
		logger.Errorf("pub set answer err=%v, pubid=%s", err, pub.Id)
		return webrtc.SessionDescription{}, err
	}
	return answer, err
}

// DoAudioRtp 处理音频RTP包
func (pub *Pub) DoAudioRtp() {
	if pub.TrackAudio != nil && pub.TrackAudio.Track() != nil {
		for {
			if pub.stop || !pub.alive {
				return
			}

			rtp, err := pub.TrackAudio.Track().ReadRTP()
			if err != nil {
				if err == io.EOF {
					pub.alive = false
					logger.Errorf("pub.TrackAudio ReadRTP error io.EOF")
				}
			} else {
				if pub.stop || !pub.alive {
					return
				}
				pub.RtpAudioCh <- rtp
			}
		}
	}
}

// DoVideoRtp 处理视频RTP包
func (pub *Pub) DoVideoRtp() {
	if pub.TrackVideo != nil && pub.TrackVideo.Track() != nil {
		for {
			if pub.stop || !pub.alive {
				return
			}

			rtp, err := pub.TrackVideo.Track().ReadRTP()
			if err != nil {
				if err == io.EOF {
					pub.alive = false
					logger.Errorf("pub.TrackVideo ReadRTP error io.EOF")
				}
			} else {
				if pub.stop || !pub.alive {
					return
				}
				pub.RtpVideoCh <- rtp
			}
		}
	}
}

// ReadAudioRTP 读音频RTP包
func (pub *Pub) ReadAudioRTP() (*rtp.Packet, error) {
	rtp, ok := <-pub.RtpAudioCh
	if !ok {
		return nil, errors.New("pub audio rtp chan close")
	}
	return rtp, nil
}

// ReadVideoRTP 读视频RTP包
func (pub *Pub) ReadVideoRTP() (*rtp.Packet, error) {
	rtp, ok := <-pub.RtpVideoCh
	if !ok {
		return nil, errors.New("pub video rtp chan close")
	}
	return rtp, nil
}

// WriteVideoRtcp 发RTCP包
func (pub *Pub) WriteVideoRtcp(pkg rtcp.Packet) error {
	if pub.pc != nil {
		return pub.pc.WriteRTCP([]rtcp.Packet{pkg})
	}
	return errors.New("pub pc is nil")
}
