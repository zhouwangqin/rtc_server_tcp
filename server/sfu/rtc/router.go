package rtc

import (
	"errors"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"
	"github.com/zhuanxin-sz/go-protoo/logger"
)

const (
	liveCycle = 6 * time.Second
)

// Router 对象
type Router struct {
	Id         string
	stop       bool
	pub        *Pub
	subs       map[string]*Sub
	subsLock   sync.Mutex
	audioAlive time.Time
	videoAlive time.Time
}

// NewRouter 创建Router对象
func NewRouter(id string) *Router {
	router := &Router{
		Id:         id,
		stop:       false,
		pub:        nil,
		subs:       make(map[string]*Sub),
		audioAlive: time.Now().Add(liveCycle),
		videoAlive: time.Now().Add(liveCycle),
	}
	return router
}

// AddPub 增加Pub对象
func (router *Router) AddPub(mid, sdp string) (string, error) {
	pub, err := NewPub(mid)
	if err != nil {
		logger.Errorf("router add pub err=%v, id=%s, mid=%s", err, router.Id, mid)
		return "", err
	}

	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	answer, err := pub.Answer(offer)
	if err != nil {
		logger.Errorf("router pub answer err=%v, id=%s, mid=%s", err, router.Id, mid)
		pub.Close()
		return "", err
	}

	logger.Debugf("router add pub = %s", pub.Id)

	router.pub = pub
	// 启动RTP处理线程
	go router.DoAudioWork()
	go router.DoVideoWork()
	return answer.SDP, nil
}

// AddSub 增加Sub对象
func (router *Router) AddSub(sid, sdp string) (string, error) {
	sub, err := NewSub(sid)
	if err != nil {
		logger.Errorf("router add sub err=%v, id=%s, sid=%s", err, router.Id, sid)
		return "", err
	}

	if router.pub != nil {
		if router.pub.TrackAudio != nil {
			if router.pub.TrackAudio.Track() != nil {
				err = sub.AddTrack(router.pub.TrackAudio.Track())
				if err != nil {
					logger.Errorf("router sub add audio track err=%v, id=%s, sid=%s", err, router.Id, sid)
					sub.Close()
					return "", err
				}
			}
		}
	}

	if router.pub != nil {
		if router.pub.TrackVideo != nil {
			if router.pub.TrackVideo.Track() != nil {
				err = sub.AddTrack(router.pub.TrackVideo.Track())
				if err != nil {
					logger.Errorf("router sub add video track err=%v, id=%s, sid=%s", err, router.Id, sid)
					sub.Close()
					return "", err
				}
			}
		}
	}

	if sub.TrackAudio == nil && sub.TrackVideo == nil {
		sub.Close()
		return "", errors.New("router sub no audio and video track")
	}

	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}
	answer, err := sub.Answer(offer)
	if err != nil {
		logger.Errorf("router sub offer err=%v, id=%s, sid=%s", err, router.Id, sid)
		sub.Close()
		return "", err
	}

	logger.Debugf("router add sub = %s", sub.Id)

	router.subsLock.Lock()
	router.subs[sid] = sub
	router.subsLock.Unlock()

	// 启动RTCP处理线程
	go router.DoRTCPWork(sub)
	return answer.SDP, nil
}

// GetSub 获取Sub对象
func (router *Router) GetPub() *Pub {
	return router.pub
}

// GetSub 获取Sub对象
func (router *Router) GetSub(sid string) *Sub {
	router.subsLock.Lock()
	defer router.subsLock.Unlock()
	return router.subs[sid]
}

// DelSub 删除Sub对象
func (router *Router) DelSub(sid string) {
	router.subsLock.Lock()
	defer router.subsLock.Unlock()
	sub := router.subs[sid]
	if sub != nil {
		sub.Close()
		delete(router.subs, sid)
	}
}

// GetSubCount 获取subs数量
func (router *Router) GetSubs() map[string]*Sub {
	return router.subs
}

// Alive 判断Router状态
func (router *Router) Alive() bool {
	if router.stop {
		return false
	}
	if router.pub != nil {
		if router.pub.stop || !router.pub.alive {
			return false
		}

		bAudio := !router.audioAlive.Before(time.Now())
		bVideo := !router.videoAlive.Before(time.Now())
		return (bAudio || bVideo)
	}
	return true
}

// Close 关闭Router
func (router *Router) Close() {
	router.stop = true
	if router.pub != nil {
		router.pub.Close()
		router.pub = nil
	}
	router.subsLock.Lock()
	for sid, sub := range router.subs {
		sub.Close()
		delete(router.subs, sid)
	}
	router.subsLock.Unlock()
}

// DoAudioWork 处理音频
func (router *Router) DoAudioWork() {
	for {
		if router.stop || router.pub == nil || router.pub.stop || !router.pub.alive {
			return
		}

		if router.pub != nil && router.pub.TrackAudio != nil {
			pkt, err := router.pub.ReadAudioRTP()
			if err == nil {
				router.audioAlive = time.Now().Add(liveCycle)
				router.subsLock.Lock()
				for sid, sub := range router.subs {
					if sub.stop || !sub.alive {
						sub.Close()
						delete(router.subs, sid)
					} else {
						sub.WriteAudioRtp(pkt)
					}
				}
				router.subsLock.Unlock()
			}
		} else {
			time.Sleep(time.Second)
		}
	}
}

// DoVideoWork 处理视频
func (router *Router) DoVideoWork() {
	for {
		if router.stop || router.pub == nil || router.pub.stop || !router.pub.alive {
			return
		}

		if router.pub != nil && router.pub.TrackVideo != nil {
			pkt, err := router.pub.ReadVideoRTP()
			if err == nil {
				router.videoAlive = time.Now().Add(liveCycle)
				router.subsLock.Lock()
				for sid, sub := range router.subs {
					if sub.stop || !sub.alive {
						sub.Close()
						delete(router.subs, sid)
					} else {
						sub.WriteVideoRtp(pkt)
					}
				}
				router.subsLock.Unlock()
			}
		} else {
			time.Sleep(time.Second)
		}
	}
}

// DoRTCPWork 处理RTCP包,目前只用处理视频
func (router *Router) DoRTCPWork(sub *Sub) {
	for {
		if router.stop || sub.TrackVideo == nil || sub.stop || !sub.alive {
			return
		}

		if sub.TrackVideo != nil {
			pkt, err := sub.ReadVideoRTCP()
			if err == nil {
				switch (pkt).(type) {
				case *rtcp.PictureLossIndication:
					if router.pub != nil {
						router.pub.WriteVideoRtcp(pkt)
					}
				case *rtcp.TransportLayerNack:
					nack := (pkt).(*rtcp.TransportLayerNack)
					for _, nackPair := range nack.Nacks {
						nackpkt := &rtcp.TransportLayerNack{
							SenderSSRC: nack.SenderSSRC,
							MediaSSRC:  nack.MediaSSRC,
							Nacks:      []rtcp.NackPair{{PacketID: nackPair.PacketID}},
						}
						if router.pub != nil {
							router.pub.WriteVideoRtcp(nackpkt)
						}
					}
				default:
				}
			}
		}
	}
}
