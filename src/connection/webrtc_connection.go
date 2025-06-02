// file: webrtc_connection.go

package connection

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

const USE_CUSTOM_TRACK = true

type logverbose func(string)

type callicecandidatecallback func(string)
type calllocaldescriptioncallback func(int, string)
type callremotetrackcallback func(int, uint32, string, uint32, uint16)
type calltrackdatacallback func(uint32, []byte, int)

type WebRTCCallbacks struct {
	IceCandidate     callicecandidatecallback
	LocalDescription calllocaldescriptioncallback
	RemoteTrackAdded callremotetrackcallback
	TrackData        calltrackdatacallback
	LogVerbose       logverbose
}

type TrackDataPacket struct {
	data []byte
}

type ReceiveDataStats struct {
	NumPackets uint64
	PacketRate int
}

type WebRTCDataChannel struct {
	Id          int32
	DataChannel *webrtc.DataChannel
}

type WebRTCConnection struct {
	peerConnection    *webrtc.PeerConnection
	dataChannel       *webrtc.DataChannel
	dataChannels      []*WebRTCDataChannel
	localSampleTrack  *webrtc.TrackLocalStaticSample
	customSampleTrack *TrackLocalSample
	localTrackChannel chan TrackDataPacket

	waitGroup    sync.WaitGroup
	callbacks    WebRTCCallbacks
	receiveStats ReceiveDataStats

	nextChannelId int32
}

func CreatePeerConnection(config webrtc.Configuration, callbacks WebRTCCallbacks) (*WebRTCConnection, error) {

	// Create a new WebRTC API object
	var err error

	var peerConnection *webrtc.PeerConnection = nil

	mediaEngine := webrtc.MediaEngine{}
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/opus", ClockRate: 48000, Channels: 1, SDPFmtpLine: "useinbandfec=1;stereo=1;sprop-stereo=1;maxaveragebitrate=96000", RTCPFeedback: nil},
		PayloadType:        111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		//LogError("mediaEngine contained no audio codecs: " + err.Error())
		return nil, err
	}

	// if err := mediaEngine.RegisterDefaultCodecs(); err != nil {
	// 	CallLogCallback("Failed to register default codecs", 0)
	// 	panic(err)
	// }

	api := webrtc.NewAPI(webrtc.WithMediaEngine(&mediaEngine))

	peerConnection, err = api.NewPeerConnection(config)

	if err != nil {
		//LogError("Failed to create new peer connection")
		return nil, err
	}

	callbacks.LogVerbose("Created peer connection")
	//LogInfo("peer connection created")

	return &WebRTCConnection{
		peerConnection: peerConnection,
		callbacks:      callbacks,
		nextChannelId:  1,
	}, nil
}

func (conn *WebRTCConnection) ConnectionState() webrtc.PeerConnectionState {
	return conn.peerConnection.ConnectionState()
}

func (conn *WebRTCConnection) ICEGatheringState() webrtc.ICEGatheringState {
	return conn.peerConnection.ICEGatheringState()
}

func (conn *WebRTCConnection) SignalingState() webrtc.SignalingState {
	return conn.peerConnection.SignalingState()
}

func (conn *WebRTCConnection) Init() (err error) {
	//conn.CreateDataChannel("data")

	// Set up ICE candidate handler
	conn.peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			conn.callbacks.LogVerbose("ice candidate: " + candidate.String())
			conn.callbacks.IceCandidate(candidate.String())
		}
	})

	// Set up data channel handler
	conn.peerConnection.OnDataChannel(conn.dataChannelHandler)

	conn.peerConnection.OnSignalingStateChange(func(s webrtc.SignalingState) {
		conn.callbacks.LogVerbose("signaling state changed to " + s.String())
	})

	conn.peerConnection.OnICEGatheringStateChange(func(s webrtc.ICEGatheringState) {
		conn.callbacks.LogVerbose("ICE gathering state changed to " + s.String())
	})

	conn.peerConnection.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		conn.callbacks.LogVerbose("ICE connection state changed to " + s.String())
	})

	conn.peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		conn.callbacks.LogVerbose("connection state changed to " + s.String())

		// if s == webrtc.PeerConnectionStateConnected {
		// 	conn.callbacks.LogVerbose("connection established. creating data channel...")
		// 	conn.CreateDataChannel("data")
		// }
	})

	conn.peerConnection.OnTrack(conn.trackHandler)

	if USE_CUSTOM_TRACK {
		err = conn.AddLocalCustomTrack(webrtc.RTPCodecCapability{MimeType: "audio/opus"}, "test", "stream")
		conn.callbacks.LogVerbose("Added custom sample track")
	} else {
		err = conn.AddLocalSampleTrack()
		conn.callbacks.LogVerbose("Added local sample track")
	}

	return err
}

func (conn *WebRTCConnection) Close() (err error) {

	if conn.dataChannel != nil {
		conn.callbacks.LogVerbose("closing data channel...")
		conn.dataChannel.Close()
	}

	for _, v := range conn.dataChannels {
		if v.DataChannel != nil {
			conn.callbacks.LogVerbose("closing data channel " + v.DataChannel.Label())
			v.DataChannel.Close()
		}
	}
	conn.dataChannels = nil

	if conn.peerConnection != nil {
		conn.callbacks.LogVerbose("closing connection...")

		err = conn.peerConnection.Close()

		if err != nil {
			//CallLogCallback("Failed to close connection: "+err.Error(), 0)
			return err
		}

		time.Sleep(1 * time.Second)

		conn.callbacks.LogVerbose("waiting for workers...")
		close(conn.localTrackChannel)
		conn.waitGroup.Wait()
		conn.callbacks.LogVerbose("workes stopped")

		conn.callbacks.LogVerbose("connection closed")
		conn.peerConnection = nil
	}

	return err
}

func (conn *WebRTCConnection) AddLocalCustomTrack(c webrtc.RTPCodecCapability, id, streamID string) (err error) {

	// Create an audio track using Opus codec with NewTrackLocalStaticSample
	rtpTrack, err := webrtc.NewTrackLocalStaticRTP(c, id, streamID)
	if err != nil {
		//LogError("Failed to create static sample")
		return err
	}

	conn.customSampleTrack = &TrackLocalSample{
		rtpTrack: rtpTrack,
	}

	// Add the media stream and start it
	_, err = conn.peerConnection.AddTrack(conn.customSampleTrack)
	if err != nil {
		//LogError("Failed to add track")
		return err
	}

	conn.waitGroup.Add(1)
	// Create a channel for the remote tracks and start reading from it
	conn.localTrackChannel = make(chan TrackDataPacket, 64)
	go conn.TrackDataSender(conn.localTrackChannel)

	return err
}

func (conn *WebRTCConnection) AddLocalSampleTrack() (err error) {

	// Create an audio track using Opus codec with NewTrackLocalStaticSample
	conn.localSampleTrack, err = webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
		MimeType:  webrtc.MimeTypeOpus,
		ClockRate: 48000,
	}, "audio", "pion")
	if err != nil {
		//LogError("Failed to create static sample")
		return err
	}

	// Add the media stream and start it
	_, err = conn.peerConnection.AddTrack(conn.localSampleTrack)
	if err != nil {
		//LogError("Failed to add track")
		return err
	}

	conn.waitGroup.Add(1)
	// Create a channel for the remote tracks and start reading from it
	conn.localTrackChannel = make(chan TrackDataPacket, 64)
	go conn.TrackDataSender(conn.localTrackChannel)

	return err
}

func (conn *WebRTCConnection) SendLocalTrackPacket(packet TrackDataPacket) (err error) {
	if conn.customSampleTrack != nil {
		err = conn.customSampleTrack.WriteSample(media.Sample{
			Data:     packet.data,
			Duration: 20 * time.Millisecond,
		})
	} else {
		err = conn.localSampleTrack.WriteSample(media.Sample{
			Data:     packet.data,
			Duration: 20 * time.Millisecond,
		})
	}
	return err
}

func (conn *WebRTCConnection) TrackDataSender(ch <-chan TrackDataPacket) {
	conn.callbacks.LogVerbose("Starting writing to local track")
	defer conn.waitGroup.Done()
	var packetBuffer []TrackDataPacket
	var mu sync.Mutex = sync.Mutex{}
	timeout := 20 * time.Millisecond
	buffered := false
	//timer := time.NewTimer(timeout)
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()
	noDataTimeBegin := time.Now()
	noDataTimeEnd := noDataTimeBegin
	lastSendTime := time.Now()

	for {
		select {
		case packet, open := <-ch:
			if !open {
				conn.callbacks.LogVerbose("Track data channel closed. Exiting...")
				return
			} else {
				mu.Lock()
				packetBuffer = append(packetBuffer, packet)
				//conn.callbacks.LogVerbose(fmt.Sprintf("received %d, buffered %d packets", len(packet.data), len(packetBuffer)))
				if len(packetBuffer) > 1 {
					buffered = true
				}
				mu.Unlock()

				if buffered {
					mu.Lock()
					p := packetBuffer[0]
					packetBuffer = packetBuffer[1:]
					mu.Unlock()
					err := conn.SendLocalTrackPacket(p)
					if err != nil {
						conn.callbacks.LogVerbose("Error writing to track: " + err.Error())
					}
				}
			}
		case <-ticker.C:
			var err error
			if len(packetBuffer) > 0 && buffered {
				t := time.Now()
				if noDataTimeBegin.After(noDataTimeEnd) {
					noDataTimeEnd = t
					conn.callbacks.LogVerbose("no data period: " + fmt.Sprint(noDataTimeEnd.Sub(noDataTimeBegin).Seconds()))
				}
				// mu.Lock()
				// p := packetBuffer[0]
				// packetBuffer = packetBuffer[1:]
				// mu.Unlock()
				// err = conn.SendLocalTrackPacket(p)
				sendTime := time.Since(t)
				timeSinceLastSend := time.Since(lastSendTime) //+ 500*time.Microsecond
				if timeSinceLastSend.Milliseconds() < 40 {
					timeout = 40*time.Millisecond - timeSinceLastSend - sendTime
				} else {
					timeout = 20*time.Millisecond - sendTime
				}
				//conn.callbacks.LogVerbose(fmt.Sprintf("sending time: %du, time since last send: %dms, setting timeout to: %dms", sendTime.Microseconds(), timeSinceLastSend.Milliseconds(), timeout.Milliseconds()))
				lastSendTime = t

				//err = nil
			} else {
				t := time.Now()
				if t.After(noDataTimeEnd) && noDataTimeEnd.After(noDataTimeBegin) {
					noDataTimeBegin = t
					conn.callbacks.LogVerbose("No data to send. Sending empty buffer")
				}
				err = conn.SendLocalTrackPacket(TrackDataPacket{data: []byte{0x00, 0x00}})
			}

			if err != nil {
				conn.callbacks.LogVerbose("Error writing to track: " + err.Error())
			}

			ticker.Reset(max(timeout, 1*time.Millisecond))
		}
	}
}

// CreateDataChannel creates a new data channel for the WebRTC connection.
func (conn *WebRTCConnection) CreateDataChannel(label string) (*WebRTCDataChannel, error) {
	// Create a new data channel
	dataChannel, err := conn.peerConnection.CreateDataChannel(label, nil)
	if err != nil {
		conn.callbacks.LogVerbose("Failed to create data channel: " + err.Error())
		return nil, err
	}

	// Set up event handlers for the data channel
	dataChannel.OnOpen(func() {
		conn.callbacks.LogVerbose("Data channel is open! Label: " + dataChannel.Label() + " ID: " + fmt.Sprint(dataChannel.ID()))
		//fmt.Println("Data channel is open!")
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		//conn.callbacks.LogVerbose("Received message: " + string(msg.Data))
	})

	newDC := &WebRTCDataChannel{
		Id:          atomic.AddInt32(&conn.nextChannelId, 1),
		DataChannel: dataChannel,
	}

	conn.dataChannels = append(conn.dataChannels, newDC)

	return newDC, nil
}

func (conn *WebRTCConnection) dataChannelHandler(dc *webrtc.DataChannel) {
	// Handle data channel events here
	dc.OnOpen(func() {
		conn.callbacks.LogVerbose("Data channel is open! Label: " + dc.Label() + " ID: " + fmt.Sprint(dc.ID()))
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		//conn.callbacks.LogVerbose("Received message: " + string(msg.Data))
	})

	dc.OnClose(func() {
		conn.callbacks.LogVerbose("Data channel closed")
	})

	dc.OnError(func(err error) {
		conn.callbacks.LogVerbose("Data channel error: " + err.Error())
	})
}

// func addSDPOptions(sdp string) string {
// 	modifiedSDP := sdp + "a=ptime:20\n"

// 	return modifiedSDP
// }

func (conn *WebRTCConnection) CreateOffer() error {
	offer, err := conn.peerConnection.CreateOffer(nil)
	if err != nil {
		conn.callbacks.LogVerbose("Failed to create an offer: " + err.Error())
		return err
	}

	gatherComplete := webrtc.GatheringCompletePromise(conn.peerConnection)

	// modifiedSDP := addSDPOptions(offer.SDP)
	// offer = webrtc.SessionDescription{
	// 	Type: webrtc.SDPTypeOffer,
	// 	SDP:  modifiedSDP,
	// }

	err = conn.peerConnection.SetLocalDescription(offer)
	if err != nil {
		conn.callbacks.LogVerbose("Failed to set the offer: " + err.Error())
		return err
	}
	conn.callbacks.LogVerbose("offer created: " + offer.SDP)
	<-gatherComplete

	// Get the local description
	localDescription := conn.peerConnection.LocalDescription()

	// Convert the local description to JSON
	localDescriptionJSON, err := json.Marshal(localDescription)
	if err != nil {
		conn.callbacks.LogVerbose("Failed to serialize to JSON: " + err.Error())
		return err
	}
	conn.callbacks.LogVerbose("local description: " + string(localDescriptionJSON))

	conn.callbacks.LocalDescription(1, offer.SDP)

	return err
}

func (conn *WebRTCConnection) SetRemoteDescription(sdpString string) error {
	remoteSDP := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdpString,
	}

	err := conn.peerConnection.SetRemoteDescription(remoteSDP)
	if err != nil {
		conn.callbacks.LogVerbose("Failed to set remote description: " + err.Error())
	}

	conn.callbacks.LogVerbose("remote description set to: " + sdpString)

	return err
}

// SetLocalDescription sets the local SDP (Session Description Protocol).
func (conn *WebRTCConnection) SetLocalDescription(sdp webrtc.SessionDescription) error {
	return conn.peerConnection.SetLocalDescription(sdp)
}

func (conn *WebRTCConnection) AddICECandidate(candidate string) error {
	err := conn.peerConnection.AddICECandidate(webrtc.ICECandidateInit{
		Candidate: candidate,
	})
	if err != nil {
		conn.callbacks.LogVerbose("AddICECandidate failed: " + err.Error())

	} else {
		conn.callbacks.LogVerbose("added ICE candidate: " + candidate)
	}

	return err
}

func (conn *WebRTCConnection) SendDataChannelText(channel int32, msg string) (err error) {
	for _, v := range conn.dataChannels {
		if v.Id == channel {
			return v.DataChannel.SendText(msg)
		}
	}

	return fmt.Errorf("channel %d not found", channel)
}

func (conn *WebRTCConnection) GetDataChannelReadyState(channel int32) (state webrtc.DataChannelState) {
	for _, v := range conn.dataChannels {
		if v.Id == channel {
			return v.DataChannel.ReadyState()
		}
	}

	return webrtc.DataChannelStateUnknown
}

func (conn *WebRTCConnection) SendTrackDataPacket(packet []byte) (err error) {
	conn.localTrackChannel <- TrackDataPacket{data: packet}
	return nil
}

func (conn *WebRTCConnection) trackHandler(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
	trackKind := track.Kind()

	if trackKind == webrtc.RTPCodecTypeAudio {
		conn.audioTrackHandler(track, receiver)
	}
}

func (conn *WebRTCConnection) audioTrackHandler(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
	channels := track.Codec().Channels
	freq := track.Codec().ClockRate
	mimeType := track.Codec().MimeType
	formattedString := fmt.Sprintf("received track %s ssrc %d type %s freq %d channels %d payload type %d.", track.Kind().String(), track.SSRC(), mimeType, freq, channels, track.PayloadType())
	conn.callbacks.LogVerbose(formattedString)
	conn.callbacks.RemoteTrackAdded(int(track.Kind()), uint32(track.SSRC()), mimeType, freq, channels)

	conn.callbacks.LogVerbose("Starting reading from remote track")
	//bufferSize := freq * 10 / 1000
	//buffer := make([]byte, 1500)
	ssrc := uint32(track.SSRC())
	packetSeen := time.Now()
	lastPacketCounterCheck := time.Now()
	numPackets := 0
	packetRate := 0
	var lastPacket *rtp.Packet = nil
	var underrun = false
	for {
		audioPacket, _, readErr := track.ReadRTP()
		//len, _, err := track.Read(buffer)
		// if err != nil {
		// 	CallLogCallback("Error reading from track: "+err.Error(), 2)
		// 	break
		// }
		if readErr != nil {
			conn.callbacks.LogVerbose("Error reading from track: " + readErr.Error())
			break
		}
		if audioPacket == nil {
			if time.Since(packetSeen) > time.Second*10 {
				conn.callbacks.LogVerbose("audioTrackHandler: Nothing received for 10 seconds. Aborting...")
				return
			}
		}

		now := time.Now()
		delta := now.Sub(packetSeen)
		packetSeen = now

		if lastPacket != nil && audioPacket != nil {
			frames := audioPacket.Timestamp - lastPacket.Timestamp
			period := float32(frames) * 1000 / float32(freq)
			too_slow := float32(delta.Milliseconds()) > period
			if !underrun && too_slow {
				conn.callbacks.LogVerbose(fmt.Sprintf("audioTrackHandler: buffer underrun delta: %d period: %f", delta.Milliseconds(), period))
				underrun = true
			}
		}

		if lastPacket != nil && lastPacket.SequenceNumber+1 != audioPacket.SequenceNumber {
			formattedString := fmt.Sprintf("audioTrackHandler: Missing packet! sequence number %d and previous one %d", audioPacket.SequenceNumber, lastPacket.SequenceNumber)
			conn.callbacks.LogVerbose(formattedString)
		}

		// formattedString := fmt.Sprintf("OnTrack: sequence number %d len %d", audioPacket.SequenceNumber, len(audioPacket.Payload))
		// CallLogCallback(formattedString, 2)
		if lastPacket != nil {
			numPackets++
			conn.receiveStats.NumPackets = uint64(numPackets)
			packetCountingDuration := time.Since(lastPacketCounterCheck)
			if packetCountingDuration > time.Second {
				lastPacketCounterCheck = time.Now()
				packetRate = numPackets * 1000 / int(packetCountingDuration.Milliseconds())
				numPackets = 0
				conn.receiveStats.PacketRate = packetRate

				//timestampDelta := audioPacket.Timestamp - lastPacket.Timestamp
				//conn.callbacks.LogVerbose(fmt.Sprintf("receive delta: %d timestamp delta: %d packet rate := %d/s", delta.Milliseconds(), timestampDelta, packetRate))
			}
		}

		payload := audioPacket.Payload
		conn.callbacks.TrackData(ssrc, payload, len(payload))

		lastPacket = audioPacket

		//time.Sleep(10 * time.Millisecond)
		//CallLogCallback("received "+strconv.Itoa(len), 2)
		//CallTrackDataCallback(buffer, len(buffer))
	}
}
