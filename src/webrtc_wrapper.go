// file: webrtc_wrapper.go
package main

/*
#include <stdlib.h> // for C.free
#include <stdint.h> // for uint8_t

typedef enum {
	PionErrorCodeInvalid = -1
} PionErrorCode;

typedef enum {
	PionConnectionStateUnknown,

	// PeerConnectionStateNew indicates that any of the ICETransports or
	// DTLSTransports are in the "new" state and none of the transports are
	// in the "connecting", "checking", "failed" or "disconnected" state, or
	// all transports are in the "closed" state, or there are no transports.
	PionConnectionStateNew,

	// PeerConnectionStateConnecting indicates that any of the
	// ICETransports or DTLSTransports are in the "connecting" or
	// "checking" state and none of them is in the "failed" state.
	PionConnectionStateConnecting,

	// PeerConnectionStateConnected indicates that all ICETransports and
	// DTLSTransports are in the "connected", "completed" or "closed" state
	// and at least one of them is in the "connected" or "completed" state.
	PionConnectionStateConnected,

	// PeerConnectionStateDisconnected indicates that any of the
	// ICETransports or DTLSTransports are in the "disconnected" state
	// and none of them are in the "failed" or "connecting" or "checking" state.
	PionConnectionStateDisconnected,

	// PeerConnectionStateFailed indicates that any of the ICETransports
	// or DTLSTransports are in a "failed" state.
	PionConnectionStateFailed,

	// PeerConnectionStateClosed indicates the peer connection is closed
	PionConnectionStateClosed
} PionConnectionState;

typedef enum {
	PionDataChannelStateUnknown,

	// DataChannelStateConnecting indicates that the data channel is being
	// established. This is the initial state of DataChannel, whether created
	// with CreateDataChannel, or dispatched as a part of an DataChannelEvent.
	PionDataChannelStateConnecting,

	// DataChannelStateOpen indicates that the underlying data transport is
	// established and communication is possible.
	PionDataChannelStateOpen,

	// DataChannelStateClosing indicates that the procedure to close down the
	// underlying data transport has started.
	PionDataChannelStateClosing,

	// DataChannelStateClosed indicates that the underlying data transport
	// has been closed or could not be established.
	PionDataChannelStateClosed
} PionDataChannelState;

typedef enum {
	PionIceGatheringStateUnknown,

	// ICEGatheringStateNew indicates that any of the ICETransports are
	// in the "new" gathering state and none of the transports are in the
	// "gathering" state, or there are no transports.
	PionIceGatheringStateNew,

	// ICEGatheringStateGathering indicates that any of the ICETransports
	// are in the "gathering" state.
	PionIceGatheringStateGathering,

	// ICEGatheringStateComplete indicates that at least one ICETransport
	// exists, and all ICETransports are in the "completed" gathering state.
	PionIceGatheringStateComplete
} PionIceGatheringState;

typedef enum {
	PionSignalingStateUnknown,

	// SignalingStateStable indicates there is no offer/answer exchange in
	// progress. This is also the initial state, in which case the local and
	// remote descriptions are nil.
	PionSignalingStateStable,

	// SignalingStateHaveLocalOffer indicates that a local description, of
	// type "offer", has been successfully applied.
	PionSignalingStateHaveLocalOffer,

	// SignalingStateHaveRemoteOffer indicates that a remote description, of
	// type "offer", has been successfully applied.
	PionSignalingStateHaveRemoteOffer,

	// SignalingStateHaveLocalPranswer indicates that a remote description
	// of type "offer" has been successfully applied and a local description
	// of type "pranswer" has been successfully applied.
	PionSignalingStateHaveLocalPranswer,

	// SignalingStateHaveRemotePranswer indicates that a local description
	// of type "offer" has been successfully applied and a remote description
	// of type "pranswer" has been successfully applied.
	PionSignalingStateHaveRemotePranswer,

	// SignalingStateClosed indicates The PeerConnection has been closed.
	PionSignalingStateClosed
} PionSignalingState;

typedef struct {
    const char* hostname;
    const char* username;
	const char* credential;
    int credential_type;
} PionIceServer;

typedef struct {
	const PionIceServer* ice_servers;
	int num_servers;
} PionPeerConnectionConfiguration;

// Example of function declaration in C
extern void onMessage(uint8_t* msg, int len);
extern void onIceCandidate(const char* candidate);
extern void onDataChannelMessage(const char* msg, int len);

typedef void (*cb)(int);
static void helper(cb f, int x) { f(x); }

// Helper to call log callbacks
typedef void (*logcb)(const char*, int);
static void helper_log(logcb f, const char* msg, int level) { f(msg, level); }

// Helper to call ICE candidate callback
typedef void (*icecandidatecb)(const char*);
static void helper_ice_candidate(icecandidatecb f, const char* msg) { f(msg); }

// helper to call local description callback
typedef void (*localdescriptioncb)(int, const char*);
static void helper_local_description(localdescriptioncb f, int type, const char* sdp) { f(type, sdp); }

// helper to call new track callback
typedef void (*remotetrackcb)(int, unsigned int, const char*, unsigned int, unsigned short);
static void helper_remote_track(remotetrackcb f, int kind, unsigned int ssrc, const char* mime, unsigned int sample_rate, unsigned short channels) { f(kind, ssrc, mime, sample_rate, channels); }

// helper to call new track callback
typedef void (*trackdatacb)(unsigned int, const char*, unsigned int);
static void helper_track_data(trackdatacb f, unsigned int ssrc, const char* data, unsigned int length) { f(ssrc, data, length); }

typedef struct {
	logcb log_callback;
	icecandidatecb ice_candidate_callback;
	localdescriptioncb local_description_callback;
	remotetrackcb remote_track_callback;
	trackdatacb track_data_callback;
} PionCallbacks;
*/
import "C"
import (
	"pionc/connection"
	"unsafe"

	"github.com/pion/webrtc/v4"
)

var pion_callbacks C.PionCallbacks

var pionConnection *connection.WebRTCConnection = nil

//export pionInit
func pionInit() {
}

//export pionSetCallbacks
func pionSetCallbacks(cb C.PionCallbacks) {
	pion_callbacks = cb
}

// func pionCallback(f C.cb, x C.int) {
// 	pion_callback = f
// 	C.helper(f, x)
// }

type LogLevel int

const (
	LogLevelError LogLevel = iota
	LogLevelWarning
	LogLevelInfo
)

// levels: 0 - error, 1 - warnning, 2 - info
func CallLogCallback(msg string, level LogLevel) {
	var cmsg = C.CString(msg)
	C.helper_log(pion_callbacks.log_callback, cmsg, C.int(level))
	C.free(unsafe.Pointer(cmsg))
}

func LogError(msg string) {
	CallLogCallback(msg, LogLevelError)
}

func LogWarning(msg string) {
	CallLogCallback(msg, LogLevelWarning)
}

func LogInfo(msg string) {
	CallLogCallback(msg, LogLevelInfo)
}

func CallIceCandidateCallback(msg string) {
	var cmsg = C.CString(msg)
	C.helper_ice_candidate(pion_callbacks.ice_candidate_callback, cmsg)
	C.free(unsafe.Pointer(cmsg))
}

func CallLocalDescriptionCallback(desc_type int, sdp string) {
	var csdp = C.CString(sdp)
	C.helper_local_description(pion_callbacks.local_description_callback, C.int(desc_type), csdp)
	C.free(unsafe.Pointer(csdp))
}

// CallRemoteTrackCallback(C.int(track.Kind()), track.SSRC(), mimeType, freq, channels)
func CallRemoteTrackCallback(kind int, ssrc uint32, mime string, freq uint32, channels uint16) {
	var cmime_type = C.CString(mime)
	C.helper_remote_track(pion_callbacks.remote_track_callback, C.int(kind), C.uint(ssrc), cmime_type, C.uint(freq), C.ushort(channels))
	C.free(unsafe.Pointer(cmime_type))
}

func CallTrackDataCallback(ssrc uint32, data []byte, len int) {
	//var cdata = C.CString(string(data))
	C.helper_track_data(pion_callbacks.track_data_callback, C.uint(ssrc), (*C.char)(unsafe.Pointer(&data[0])), C.uint(len))
	//C.free(unsafe.Pointer(cdata))
}

// ============================================================================
// Go-to-C interface
// ============================================================================

//export pionClosePeerConnection
func pionClosePeerConnection() {
	if pionConnection != nil {
		pionConnection.Close()
	}
}

func logger(message string) {
	LogInfo(message)
}

//export pionCreatePeerConnection
func pionCreatePeerConnection(config *C.PionPeerConnectionConfiguration) int {

	pionClosePeerConnection()

	var err error
	pionConnection, err = connection.CreatePeerConnection(createPeerConnectionConfig(config), connection.WebRTCCallbacks{
		IceCandidate:     CallIceCandidateCallback,
		LocalDescription: CallLocalDescriptionCallback,
		RemoteTrackAdded: CallRemoteTrackCallback,
		TrackData:        CallTrackDataCallback,
		LogVerbose:       logger})
	if err != nil {
		LogError("Failed to create peer connection: " + err.Error())
		return -1
	}

	pionConnection.Init()

	return 1
}

//export pionCreateDataChannel
func pionCreateDataChannel(label *C.char) int32 {
	if pionConnection != nil {
		goLabel := C.GoString(label)
		ch, err := pionConnection.CreateDataChannel(goLabel)
		if err != nil {
			LogError("Failed to create data channel: " + err.Error())
		}

		LogInfo("Created data channel: " + goLabel)

		return ch.Id
	} else {
		LogError("Failed to create data channel: pionConnection is nil")
	}

	return -1
}

//export pionGetConnectionState
func pionGetConnectionState() C.PionConnectionState {
	if pionConnection == nil {
		return C.PionConnectionStateClosed
	}

	state := pionConnection.ConnectionState()

	return C.PionConnectionState(state)
}

//export pionGetIceGatheringState
func pionGetIceGatheringState() C.PionIceGatheringState {
	if pionConnection == nil {
		return C.PionIceGatheringStateNew
	}

	state := pionConnection.ICEGatheringState()

	return C.PionIceGatheringState(state)
}

//export pionGetSignalingState
func pionGetSignalingState() C.PionSignalingState {
	if pionConnection == nil {
		return C.PionSignalingStateClosed
	}

	state := pionConnection.SignalingState()

	return C.PionSignalingState(state)
}

//export pionCreateOffer
func pionCreateOffer() {
	if pionConnection != nil {
		pionConnection.CreateOffer()
	}
}

//export pionSetRemoteDescription
func pionSetRemoteDescription(sdp *C.char) {
	if pionConnection != nil {
		goSdp := C.GoString(sdp)
		err := pionConnection.SetRemoteDescription(goSdp)
		if err != nil {
			LogError("Failed to set remote description: " + err.Error())
		}
	}
}

//export pionAddICECandidate
func pionAddICECandidate(candidate *C.char) {
	if pionConnection != nil {
		goString := C.GoString(candidate)
		err := pionConnection.AddICECandidate(goString)
		if err != nil {
			LogError("Failed to add ICE candidate")
		}
	}
}

//export pionSendDataChannelText
func pionSendDataChannelText(channel int32, msg *C.char) {
	if pionConnection != nil {
		goString := C.GoString(msg)
		err := pionConnection.SendDataChannelText(channel, goString)
		if err != nil {
			LogError("Failed to send data channel messsage " + err.Error())
		}
	}
}

//export pionGetDataChannelReadyState
func pionGetDataChannelReadyState(channel int32) C.PionDataChannelState {
	if pionConnection != nil {
		state := pionConnection.GetDataChannelReadyState(channel)
		return C.PionDataChannelState(state)
	}

	return C.PionDataChannelStateUnknown
}

//export pionSendTrackDataPacket
func pionSendTrackDataPacket(data *C.char, length C.int) {
	if pionConnection != nil {
		goBytes := C.GoBytes(unsafe.Pointer(data), length)
		err := pionConnection.SendTrackDataPacket(goBytes)
		if err != nil {
			LogError("Failed to send track data packet")
		}
	}
}

// ============================================================================
// Go implementation
// ============================================================================

func createPeerConnectionConfig(config *C.PionPeerConnectionConfiguration) webrtc.Configuration {
	if config == nil {
		return webrtc.Configuration{}
	}

	struct_size := unsafe.Sizeof(*config.ice_servers)
	num_servers := int(config.num_servers)

	pion_servers := []webrtc.ICEServer{}

	var hostname string
	var username string
	var credential string

	urls := []string{}

	for i := 0; i < num_servers; i++ {
		server := (*C.PionIceServer)(unsafe.Pointer(uintptr(unsafe.Pointer(config.ice_servers)) + uintptr(i)*struct_size))
		if server != nil {
			hostname = C.GoString(server.hostname)
			username = C.GoString(server.username)
			credential = C.GoString(server.credential)

			//LogInfo("ICE server: " + hostname + " username: " + username + " password: " + credential)

			urls = append(urls, hostname)
			pion_servers = append(pion_servers, webrtc.ICEServer{
				URLs:           urls,
				Username:       username,
				Credential:     credential,
				CredentialType: webrtc.ICECredentialTypePassword,
			})
		}
	}

	return webrtc.Configuration{ICEServers: pion_servers}
}

func main() {

}
