package rtapi_test

import (
	"testing"

	"github.com/ardanlabs/service/business/ws/schema/rtapi"
	"github.com/ardanlabs/service/business/ws/sessionws"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	success = "\u2713"
	failed  = "\u2717"
)

func TestGameServerCreateSucceed(t *testing.T) {
	// Shared utility components.
	jsonpbMarshaler := &protojson.MarshalOptions{
		UseEnumNumbers:  true,
		EmitUnpopulated: false,
		Indent:          "",
		UseProtoNames:   true,
	}
	jsonpbUnmarshaler := &protojson.UnmarshalOptions{
		DiscardUnknown: false,
	}

	tableTest := [2]sessionws.SessionFormat{sessionws.SessionFormatProtobuf, sessionws.SessionFormatJson}

	request := &rtapi.Envelope{
		Cid: "1",
		Message: &rtapi.Envelope_GameServerCreateSucceed{
			GameServerCreateSucceed: &rtapi.GameServerCreateSucceed{
				IpAddress: "198.168.71.178",
				Port:      45545,
			},
		},
	}
	var err error
	var payload []byte
	data := &rtapi.Envelope{}
	for _, v := range tableTest {

		switch v {
		case sessionws.SessionFormatProtobuf:
			payload, err = proto.Marshal(request)
		case sessionws.SessionFormatJson:
			fallthrough
		default:
			payload, err = jsonpbMarshaler.Marshal(request)
		}
		if err != nil {
			t.Fatalf("Fail to Marshal")
		}

		switch v {
		case sessionws.SessionFormatProtobuf:
			err = proto.Unmarshal(payload, data)
		case sessionws.SessionFormatJson:
			fallthrough
		default:
			err = jsonpbUnmarshaler.Unmarshal(payload, data)
		}
		if err != nil {
			t.Fatalf("Fail to Marshal")
		}
		if request.Cid != data.Cid {
			t.Fatalf("Fail to Compare")
		}
		if request.Message.(*rtapi.Envelope_GameServerCreateSucceed).GameServerCreateSucceed.IpAddress != data.Message.(*rtapi.Envelope_GameServerCreateSucceed).GameServerCreateSucceed.IpAddress {
			t.Fatalf("Fail to Compare")
		}
		if request.Message.(*rtapi.Envelope_GameServerCreateSucceed).GameServerCreateSucceed.Port != data.Message.(*rtapi.Envelope_GameServerCreateSucceed).GameServerCreateSucceed.Port {
			t.Fatalf("Fail to Compare")
		}

	}

}

func TestPing(t *testing.T) {
}

func TestPong(t *testing.T) {
}
