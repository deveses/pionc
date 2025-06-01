// file: track_local_static.go

package connection

import (
	"errors"
	"strings"
	"sync"

	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

const const_rtpOutboundMTU = 1200

type TrackLocalSample struct {
	packetizer rtp.Packetizer
	sequencer  rtp.Sequencer
	rtpTrack   *webrtc.TrackLocalStaticRTP
	clockRate  float64
	mu         sync.RWMutex
}

// ID is the unique identifier for this Track. This should be unique for the
// stream, but doesn't have to globally unique. A common example would be 'audio' or 'video'
// and StreamID would be 'desktop' or 'webcam'
func (s *TrackLocalSample) ID() string { return s.rtpTrack.ID() }

// StreamID is the group this track belongs too. This must be unique
func (s *TrackLocalSample) StreamID() string { return s.rtpTrack.StreamID() }

func (s *TrackLocalSample) RID() string { return s.rtpTrack.RID() }

// Kind controls if this TrackLocal is audio or video
func (s *TrackLocalSample) Kind() webrtc.RTPCodecType { return s.rtpTrack.Kind() }

// Codec gets the Codec of the track
func (s *TrackLocalSample) Codec() webrtc.RTPCodecCapability {
	return s.rtpTrack.Codec()
}

// Bind is called by the PeerConnection after negotiation is complete
// This asserts that the code requested is supported by the remote peer.
// If so it setups all the state (SSRC and PayloadType) to have a call
func (s *TrackLocalSample) Bind(t webrtc.TrackLocalContext) (webrtc.RTPCodecParameters, error) {
	codec, err := s.rtpTrack.Bind(t)
	if err != nil {
		return codec, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// We only need one packetizer
	if s.packetizer != nil {
		return codec, nil
	}

	payloader, err := payloaderForCodecMime(codec.RTPCodecCapability.MimeType)
	if err != nil {
		return codec, err
	}

	s.sequencer = rtp.NewRandomSequencer()
	s.packetizer = rtp.NewPacketizer(
		const_rtpOutboundMTU,
		0, // Value is handled when writing
		0, // Value is handled when writing
		payloader,
		s.sequencer,
		codec.ClockRate,
	)

	// Most of the reason to implement this TrackHMCSample class is so I can add these absolute time stamps out.
	s.packetizer.EnableAbsSendTime(1)

	s.clockRate = float64(codec.RTPCodecCapability.ClockRate)
	return codec, nil
}

// Unbind implements the teardown logic when the track is no longer needed. This happens
// because a track has been stopped.
func (s *TrackLocalSample) Unbind(t webrtc.TrackLocalContext) error {
	return s.rtpTrack.Unbind(t)
}

// WriteSample writes a Sample to the TrackLocalSample
// If one PeerConnection fails the packets will still be sent to
// all PeerConnections. The error message will contain the ID of the failed
// PeerConnections so you can remove them
func (s *TrackLocalSample) WriteSample(sample media.Sample) error {
	s.mu.RLock()
	p := s.packetizer
	clockRate := s.clockRate
	s.mu.RUnlock()

	if p == nil {
		return nil
	}

	// skip packets by the number of previously dropped packets
	for i := uint16(0); i < sample.PrevDroppedPackets; i++ {
		s.sequencer.NextSequenceNumber()
	}

	samples := uint32(sample.Duration.Seconds() * clockRate)
	if sample.PrevDroppedPackets > 0 {
		p.SkipSamples(samples * uint32(sample.PrevDroppedPackets))
	}
	packets := p.Packetize(sample.Data, samples)

	writeErrs := []error{}
	for _, p := range packets {
		if err := s.rtpTrack.WriteRTP(p); err != nil {
			writeErrs = append(writeErrs, err)
		}
	}

	return FlattenErrs(writeErrs)
}

// GeneratePadding writes padding-only samples to the TrackLocalStaticSample
// If one PeerConnection fails the packets will still be sent to
// all PeerConnections. The error message will contain the ID of the failed
// PeerConnections so you can remove them
func (s *TrackLocalSample) GeneratePadding(samples uint32) error {
	s.mu.RLock()
	p := s.packetizer
	s.mu.RUnlock()

	if p == nil {
		return nil
	}

	packets := p.GeneratePadding(samples)

	writeErrs := []error{}
	for _, p := range packets {
		if err := s.rtpTrack.WriteRTP(p); err != nil {
			writeErrs = append(writeErrs, err)
		}
	}

	return FlattenErrs(writeErrs)
}

func payloaderForCodecMime(codec string) (rtp.Payloader, error) {
	switch strings.ToLower(codec) {
	case strings.ToLower(webrtc.MimeTypeH264):
		return &codecs.H264Payloader{}, nil
	case strings.ToLower(webrtc.MimeTypeOpus):
		return &codecs.OpusPayloader{}, nil
	case strings.ToLower(webrtc.MimeTypeVP8):
		return &codecs.VP8Payloader{}, nil
	case strings.ToLower(webrtc.MimeTypeVP9):
		return &codecs.VP9Payloader{}, nil
	case strings.ToLower(webrtc.MimeTypeG722):
		return &codecs.G722Payloader{}, nil
	case strings.ToLower(webrtc.MimeTypePCMU), strings.ToLower(webrtc.MimeTypePCMA):
		return &codecs.G711Payloader{}, nil
	default:
		return nil, webrtc.ErrNoPayloaderForCodec
	}
}

// FlattenErrs flattens multiple errors into one
func FlattenErrs(errs []error) error {
	errs2 := []error{}
	for _, e := range errs {
		if e != nil {
			errs2 = append(errs2, e)
		}
	}
	if len(errs2) == 0 {
		return nil
	}
	return multiError(errs2)
}

type multiError []error

func (me multiError) Error() string {
	var errstrings []string

	for _, err := range me {
		if err != nil {
			errstrings = append(errstrings, err.Error())
		}
	}

	if len(errstrings) == 0 {
		return "multiError must contain multiple error but is empty"
	}

	return strings.Join(errstrings, "\n")
}

func (me multiError) Is(err error) bool {
	for _, e := range me {
		if errors.Is(e, err) {
			return true
		}
		if me2, ok := e.(multiError); ok {
			if me2.Is(err) {
				return true
			}
		}
	}
	return false
}
