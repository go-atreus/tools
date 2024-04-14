package delayqueue

import "github.com/vmihailenco/msgpack"

// Serializer defines body serialization interface.
type Serializer interface {
	// Unmarshal deserialize the in bytes into body
	Unmarshal(in []byte, body interface{}) error

	// Marshal returns the bytes serialized from body.
	Marshal(body interface{}) (out []byte, err error)
}

var defaultSerializer Serializer

func setDefaultSerializer(s Serializer) {
	defaultSerializer = s
}

func getSerializer() Serializer {
	if defaultSerializer == nil {
		defaultSerializer = MsgPackSerializer{}
	}
	return defaultSerializer
}

func Marshal(body interface{}) (out []byte, err error) {
	return getSerializer().Marshal(body)
}

func Unmarshal(in []byte, body interface{}) error {
	return getSerializer().Unmarshal(in, body)
}

type MsgPackSerializer struct{}

func (s MsgPackSerializer) Marshal(body interface{}) (out []byte, err error) {
	return msgpack.Marshal(body)
}

func (s MsgPackSerializer) Unmarshal(in []byte, body interface{}) error {
	return msgpack.Unmarshal(in, body)
}
