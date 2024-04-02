package ptrie

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"sync"
	"unsafe"
)

// KeyProvider represents entity key provider
type KeyProvider interface {
	Key() interface{}
}

// Decoder decoder
type Decoder interface {
	Decode(reader io.Reader) error
}

// Encoder encoder
type Encoder interface {
	Encode(writer io.Writer) error
}

type values[T any] struct {
	Type  reflect.Type
	vType interface{}
	data  []T
	*sync.RWMutex
	registry map[interface{}]uint32
}

func (v *values[T]) useType(Type reflect.Type) {
	v.Type = Type
	v.vType = reflect.New(Type).Elem().Interface()
}

func (v *values[T]) put(value T) (uint32, error) {
	var key interface{}
	switch val := any(value).(type) {
	case []byte:
		key = string(val)
	case int, string, bool, uint8, uint16, uint32, uint64, int8, int16, int32, int64, float32, float64:
		key = val
	default:
		keyProvider, ok := any(value).(KeyProvider)
		if !ok {
			return 0, fmt.Errorf("unhashable type %T, consifer implementing Hash() int", value)
		}
		key = keyProvider.Key()
	}
	v.RLock()
	result, ok := v.registry[key]
	v.RUnlock()
	if ok {
		return result, nil
	}
	v.Lock()
	defer v.Unlock()
	if v.vType == nil {
		v.useType(reflect.TypeOf(value))
	}
	result = uint32(len(v.registry))
	v.registry[key] = result
	v.data = append(v.data, value)
	return result, nil
}

func (v *values[T]) Decode(reader io.Reader) error {
	var err error
	control := uint8(0)
	if err = binary.Read(reader, binary.LittleEndian, &control); err == nil {
		length := uint32(0)
		if err = binary.Read(reader, binary.LittleEndian, &length); err == nil {
			v.data = make([]T, length)
			if len(v.data) == 0 {
				return nil
			}
			switch v.vType.(type) {
			case string:
				return v.decodeStrings(reader)
			case []byte:
				return v.decodeBytes(reader)
			case int, bool, uint8, uint16, uint32, uint64, int8, int16, int32, int64, float32, float64:
				return v.decodeScalar(reader)
			default:
				return v.decodeCustom(reader)
			}
		}
	}
	return nil
}

func (v *values[T]) Encode(writer io.Writer) error {
	var err error
	if err = binary.Write(writer, binary.LittleEndian, controlByte); err == nil {
		if err = binary.Write(writer, binary.LittleEndian, uint32(len(v.data))); err == nil {
			if len(v.data) == 0 {
				return nil
			}
			switch v.vType.(type) {
			case string:
				return v.encodeStrings(writer)
			case []byte:
				return v.encodeBytes(writer)
			case int, bool, uint8, uint16, uint32, uint64, int8, int16, int32, int64, float32, float64:
				return v.encodeScalar(writer)
			default:
				return v.encodeCustom(writer)
			}
		}
	}
	return nil
}

func (v *values[T]) value(index uint32) T {
	v.RLock()
	defer v.RUnlock()
	return v.data[index]
}

func (v *values[T]) encodeScalar(writer io.Writer) error {
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(&v.data[0])), len(v.data)*int(v.Type.Size()))
	err := binary.Write(writer, binary.LittleEndian, bytes)
	return err
}

func (v *values[T]) encodeStrings(writer io.Writer) error {
	var err error
	for i := range v.data {
		item := any(v.data[i]).(string)
		if err = binary.Write(writer, binary.LittleEndian, uint32(len(item))); err == nil {
			err = binary.Write(writer, binary.LittleEndian, []byte(item))
		}
		if err != nil {
			return err
		}
	}
	return err
}

func (v *values[T]) encodeBytes(writer io.Writer) error {
	var err error
	for i := range v.data {
		item := any(v.data[i]).([]byte)
		if err = binary.Write(writer, binary.LittleEndian, uint32(len(item))); err == nil {
			err = binary.Write(writer, binary.LittleEndian, item)
		}
		if err != nil {
			return err
		}
	}
	return err
}

func (v *values[T]) encodeCustom(writer io.Writer) error {
	for i := range v.data {
		encoder, ok := any(v.data[i]).(Encoder)
		if !ok {
			return fmt.Errorf("unable to cast Encoder from %T", v.data[i])
		}
		if err := encoder.Encode(writer); err != nil {
			return err
		}
	}
	return nil
}

func (v *values[T]) decodeStrings(reader io.Reader) error {
	var err error
	for i := range v.data {
		var length uint32
		err = binary.Read(reader, binary.LittleEndian, &length)
		if err == nil {
			var item = make([]byte, length)
			if err = binary.Read(reader, binary.LittleEndian, item); err == nil {
				v.data[i] = any(string(item)).(T)
			}
		}
		if err != nil {
			return err
		}
	}
	return err
}

func (v *values[T]) decodeBytes(reader io.Reader) error {
	var err error
	for i := range v.data {
		length := uint32(0)
		if err = binary.Read(reader, binary.LittleEndian, &length); err == nil {
			var item = make([]byte, length)
			if err = binary.Read(reader, binary.LittleEndian, item); err == nil {
				v.data[i] = any(item).(T)
			}
		}
		if err != nil {
			return err
		}
	}
	return err
}

func (v *values[T]) decodeScalar(reader io.Reader) error {
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(&v.data[0])), len(v.data)*int(v.Type.Size()))
	err := binary.Read(reader, binary.LittleEndian, bytes)
	return err
}

func (v *values[T]) decodeCustom(reader io.Reader) error {
	for i := range v.data {
		newType := v.Type
		if v.Type.Kind() == reflect.Ptr {
			newType = newType.Elem()
		}
		newItem := reflect.New(newType)
		item := newItem.Interface()
		decoder, ok := item.(Decoder)
		if !ok {
			return fmt.Errorf("unable to cast Decoder from %T", item)
		}
		if err := decoder.Decode(reader); err != nil {
			return err
		}
		v.data[i] = item.(T)
	}
	return nil
}

func newValues[T any]() *values[T] {
	return &values[T]{
		data:     make([]T, 0),
		registry: make(map[interface{}]uint32),
		RWMutex:  &sync.RWMutex{},
	}
}
