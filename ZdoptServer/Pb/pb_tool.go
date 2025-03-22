package Pb

import (
	"errors"
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"sync"
)

var (
	typeRegistry   = new(sync.Map)
	ErrInvalidType = errors.New("invalid protobuf message type")
)

// RegisterType 注册协议类型（线程安全）
func RegisterType[T proto.Message]() {
	var zero T
	desc := zero.ProtoReflect().Descriptor()
	typeRegistry.Store(desc.FullName(), zero.ProtoReflect().Type())
}

// Serialize 安全序列化（带类型校验）
func Serialize(msg proto.Message) ([]byte, error) {
	if err := validateMessage(msg); err != nil {
		return nil, fmt.Errorf("serialize validation failed: %w", err)
	}
	return proto.Marshal(msg)
}

// Deserialize 安全反序列化（带类型校验）
func Deserialize[T proto.Message](data []byte) (T, error) {
	var zero T
	desc := zero.ProtoReflect().Descriptor()

	typ, ok := typeRegistry.Load(desc.FullName())
	if !ok {
		return zero, fmt.Errorf("type %s not registered", desc.FullName())
	}

	msg := typ.(protoreflect.MessageType).New().Interface()
	if err := proto.Unmarshal(data, msg); err != nil {
		return zero, fmt.Errorf("deserialize failed: %w", err)
	}

	return msg.(T), nil
}

func validateMessage(msg proto.Message) error {
	desc := msg.ProtoReflect().Descriptor()
	_, ok := typeRegistry.Load(desc.FullName())
	if !ok {
		return fmt.Errorf("%w: %s", ErrInvalidType, desc.FullName())
	}
	return nil
}
