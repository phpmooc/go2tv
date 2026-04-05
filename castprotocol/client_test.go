package castprotocol

import (
	"encoding/json"
	"testing"

	"go2tv.app/go2tv/v2/castprotocol/v2/application"
	"go2tv.app/go2tv/v2/castprotocol/v2/cast"
	pb "go2tv.app/go2tv/v2/castprotocol/v2/cast/proto"
)

type stubCastConn struct {
	recvChan chan *pb.CastMessage
	sendFn   func(requestID int, payload cast.Payload, sourceID, destinationID, namespace string) error
}

func (s *stubCastConn) Start(addr string, port int) error { return nil }

func (s *stubCastConn) MsgChan() chan *pb.CastMessage { return s.recvChan }

func (s *stubCastConn) Close() error {
	close(s.recvChan)
	return nil
}

func (s *stubCastConn) SetDebug(debug bool) {}

func (s *stubCastConn) LocalAddr() (string, error) { return "127.0.0.1", nil }

func (s *stubCastConn) RemoteAddr() (string, error) { return "127.0.0.1", nil }

func (s *stubCastConn) RemotePort() (string, error) { return "8009", nil }

func (s *stubCastConn) Send(requestID int, payload cast.Payload, sourceID, destinationID, namespace string) error {
	if s.sendFn != nil {
		return s.sendFn(requestID, payload, sourceID, destinationID, namespace)
	}
	return nil
}

func TestLoadWarmsReceiverBeforeFirstStandardLoad(t *testing.T) {
	recvChan := make(chan *pb.CastMessage, 8)

	var launchCalls int
	var primeLoadCalls int
	var realLoadCalls int
	var stopCalls int
	var receiverStatusCalls int
	var mediaLoaded bool
	var primed bool

	conn := &stubCastConn{recvChan: recvChan}
	conn.sendFn = func(requestID int, payload cast.Payload, sourceID, destinationID, namespace string) error {
		switch namespace {
		case CastNamespaceReceiver:
			switch p := payload.(type) {
			case *cast.PayloadHeader:
				if p.Type != "GET_STATUS" {
					break
				}

				receiverStatusCalls++
				resp := cast.ReceiverStatusResponse{}
				resp.PayloadHeader = cast.PayloadHeader{Type: "RECEIVER_STATUS", RequestId: requestID}
				if receiverStatusCalls > 1 {
					resp.Status.Applications = []cast.Application{{
						AppId:       cast.DefaultMediaReceiverAppID,
						TransportId: "web-1",
					}}
				}
				recvChan <- mustCastMessage(t, resp)
			case *LaunchRequest:
				launchCalls++
			}
		case "urn:x-cast:com.google.cast.media":
			switch p := payload.(type) {
			case *cast.PayloadHeader:
				if p.Type != "GET_STATUS" {
					break
				}

				resp := cast.MediaStatusResponse{}
				resp.PayloadHeader = cast.PayloadHeader{Type: "MEDIA_STATUS", RequestId: requestID}
				if mediaLoaded {
					resp.Status = []cast.Media{{
						MediaSessionId: 7,
						PlayerState:    "PAUSED",
						Media: cast.MediaItem{
							ContentId:   "http://example/media.mp4",
							ContentType: "video/mp4",
							StreamType:  "BUFFERED",
						},
					}}
				}
				recvChan <- mustCastMessage(t, resp)
			case *CustomLoadPayload:
				primeLoadCalls++
				if p.Autoplay {
					t.Fatalf("prime Autoplay = true, want false")
				}
				mediaLoaded = true
			case *cast.MediaHeader:
				switch p.Type {
				case "STOP":
					stopCalls++
					mediaLoaded = false
					primed = true
				default:
					break
				}
			case *cast.LoadMediaCommand:
				realLoadCalls++
				resp := cast.MediaStatusResponse{}
				resp.PayloadHeader = cast.PayloadHeader{Type: "MEDIA_STATUS"}
				resp.Status = []cast.Media{{IdleReason: "FINISHED"}}
				recvChan <- mustCastMessage(t, resp)
			}
		}

		return nil
	}

	client := &CastClient{
		app:       application.NewApplication(application.WithConnection(conn)),
		conn:      conn,
		connected: true,
	}

	if err := client.Load("http://example/media.mp4", "video/mp4", 0, 0, "", false); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if launchCalls != 1 {
		t.Fatalf("launchCalls = %d, want 1", launchCalls)
	}

	if !primed {
		t.Fatal("expected primed initial media session")
	}

	if primeLoadCalls != 1 {
		t.Fatalf("primeLoadCalls = %d, want 1", primeLoadCalls)
	}

	if realLoadCalls != 1 {
		t.Fatalf("realLoadCalls = %d, want 1", realLoadCalls)
	}

	if stopCalls != 1 {
		t.Fatalf("stopCalls = %d, want 1", stopCalls)
	}

	if err := client.Close(false); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func mustCastMessage(t *testing.T, payload any) *pb.CastMessage {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	payloadString := string(payloadBytes)
	protocolVersion := pb.CastMessage_CASTV2_1_0
	payloadType := pb.CastMessage_STRING

	return &pb.CastMessage{
		ProtocolVersion: &protocolVersion,
		PayloadType:     &payloadType,
		PayloadUtf8:     &payloadString,
		PayloadBinary:   payloadBytes,
	}
}
