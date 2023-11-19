package rtapi_test

import (
	"testing"
	"time"

	"github.com/ardanlabs/service/business/ws"
	"github.com/ardanlabs/service/business/ws/schema/rtapi"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

const (
	success = "\u2713"
	failed  = "\u2717"
)

func TestNotification(t *testing.T) {
	jsonpbMarshaler := &protojson.MarshalOptions{
		UseEnumNumbers:  true,
		EmitUnpopulated: false,
		Indent:          "",
		UseProtoNames:   true,
	}
	jsonpbUnmarshaler := &protojson.UnmarshalOptions{
		DiscardUnknown: false,
	}

	tableTest := [2]ws.SessionFormat{ws.SessionFormatProtobuf, ws.SessionFormatJson}

	request := &rtapi.Envelope{Message: &rtapi.Envelope_Notifications{
		Notifications: &rtapi.Notifications{
			Notifications: []*rtapi.Notification{
				{
					Id:         uuid.NewString(),
					Subject:    "single_socket",
					Content:    "{}",
					Code:       ws.NotificationCodeSingleSocket,
					SenderId:   "",
					CreateTime: &timestamppb.Timestamp{Seconds: time.Now().Unix()},
					Persistent: false,
				},
			},
		},
	},
	}

	var err error
	var payload []byte
	data := &rtapi.Envelope{}
	for _, v := range tableTest {

		switch v {
		case ws.SessionFormatProtobuf:
			payload, err = proto.Marshal(request)
		case ws.SessionFormatJson:
			fallthrough
		default:
			payload, err = jsonpbMarshaler.Marshal(request)
		}
		if err != nil {
			t.Fatalf("Fail to Marshal")
		}

		switch v {
		case ws.SessionFormatProtobuf:
			err = proto.Unmarshal(payload, data)
		case ws.SessionFormatJson:
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
		incoming1 := request.GetNotifications()
		incoming2 := data.GetNotifications()

		if len(incoming1.Notifications) != len(incoming2.Notifications) {
			t.Fatalf("Fail to compare len")
		}

		Notifications1 := make(map[string]*rtapi.Notification)
		Notifications2 := make(map[string]*rtapi.Notification)

		for k, _ := range incoming1.Notifications {
			Notifications1[incoming1.Notifications[k].Id] = incoming1.Notifications[k]
		}

		for k, _ := range incoming2.Notifications {
			Notifications2[incoming2.Notifications[k].Id] = incoming2.Notifications[k]
		}

		ok := mapsHaveSameKeys(Notifications1, Notifications2)
		if !ok {
			t.Fatalf("Fail to Compare")
		}

		for k, _ := range Notifications1 {
			if Notifications1[k].Code != Notifications2[k].Code {
				t.Fatalf("Fail to compare Notifications[%s].Code", k)
			}
			if Notifications1[k].Subject != Notifications2[k].Subject {
				t.Fatalf("Fail to compare Notifications[%s].Subject", k)
			}
			if Notifications1[k].Content != Notifications2[k].Content {
				t.Fatalf("Fail to compare Notifications[%s].Content", k)
			}
			if Notifications1[k].Code != Notifications2[k].Code {
				t.Fatalf("Fail to compare Notifications[%s].Code", k)
			}
			if Notifications1[k].SenderId != Notifications2[k].SenderId {
				t.Fatalf("Fail to compare Notifications[%s].SenderId", k)
			}
			if Notifications1[k].CreateTime.Seconds != Notifications2[k].CreateTime.Seconds {
				t.Fatalf("Fail to compare Notifications[%s].CreateTime.Seconds", k)
			}
			if int64(Notifications1[k].CreateTime.Nanos) != int64(Notifications2[k].CreateTime.Nanos) {
				t.Fatalf("Fail to compare Notifications[%s].CreateTime.Nanos", k)
			}
			if Notifications1[k].Persistent != Notifications2[k].Persistent {
				t.Fatalf("Fail to compare Notifications[%s].Persistent", k)
			}
		}
	}
}

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

	tableTest := [2]ws.SessionFormat{ws.SessionFormatProtobuf, ws.SessionFormatJson}

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
		case ws.SessionFormatProtobuf:
			payload, err = proto.Marshal(request)
		case ws.SessionFormatJson:
			fallthrough
		default:
			payload, err = jsonpbMarshaler.Marshal(request)
		}
		if err != nil {
			t.Fatalf("Fail to Marshal")
		}

		switch v {
		case ws.SessionFormatProtobuf:
			err = proto.Unmarshal(payload, data)
		case ws.SessionFormatJson:
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
		incoming1 := request.GetGameServerCreateSucceed()
		incoming2 := data.GetGameServerCreateSucceed()

		if incoming1.IpAddress != incoming2.IpAddress {
			t.Fatalf("Fail to Compare")
		}
		if incoming1.Port != incoming2.Port {
			t.Fatalf("Fail to Compare")
		}

	}

}

func TestPing(t *testing.T) {
}

func TestPong(t *testing.T) {
}

func mapsHaveSameKeys(map1, map2 map[string]*rtapi.Notification) bool {
	// 检查 map1 的每个键是否都存在于 map2 中
	for key := range map1 {
		if _, exists := map2[key]; !exists {
			return false
		}
	}

	// 检查 map2 的每个键是否都存在于 map1 中
	for key := range map2 {
		if _, exists := map1[key]; !exists {
			return false
		}
	}

	// 如果上述两个循环都通过，说明两个 map 的键完全相同
	return true
}
