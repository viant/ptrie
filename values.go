package ptrie

import (
	"encoding/binary"
	"fmt"
	"github.com/viant/toolbox"
	"io"
	"reflect"
	"sync"
)

//Hasher represents value hasher
type Hasher interface {
	Hash() int
}

//Decoder decoder
type Decoder interface {
	Decode(reader io.Reader) error
}

//Encoder encoder
type Encoder interface {
	Encode(writer io.Writer) error
}

type values struct {
	Type  reflect.Type
	vType interface{}
	data  []interface{}
	*sync.RWMutex
	registry map[interface{}]uint32
}

func (v *values) useType(Type reflect.Type) {
	v.Type = Type
	v.vType = reflect.New(Type).Elem().Interface()
}

func (v *values) put(value interface{}) (uint32, error) {
	v.RLock()
	hashedValue := value
	switch val := value.(type) {
	case []byte:
		hashedValue = string(val)
	case int, string, bool, uint8, uint16, uint32, uint64, int8, int16, int32, int64, float32, float64:
	default:
		hasher, ok := value.(Hasher)
		if !ok {
			return 0, fmt.Errorf("unhashable type %T, consifer implementing Hash() int", value)
		}
		hashedValue = hasher.Hash()
	}
	result, ok := v.registry[hashedValue]
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
	v.registry[hashedValue] = result
	v.data = append(v.data, value)
	return result, nil
}

func (v *values) Decode(reader io.Reader) error {
	var err error
	control := uint8(0)
	if err = binary.Read(reader, binary.LittleEndian, &control); err == nil {
		length := uint32(0)
		if err = binary.Read(reader, binary.LittleEndian, &length); err == nil {
			v.data = make([]interface{}, length)
			if len(v.data) == 0 {
				return nil
			}
			switch v.vType.(type) {
			case string:
				return v.decodeStrings(reader)
			case []byte:
				return v.decodeBytes(reader)
			case int:
				return v.decodeInts(reader)
			case bool:
				return v.decodeBooleans(reader)
			case uint8, uint16, uint32, uint64, int8, int16, int32, int64, float32, float64:

				return v.decodeNumbers(reader)
			default:
				return v.decodeCustom(reader)
			}
		}
	}
	return nil
}

func (v *values) Encode(writer io.Writer) error {
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
			case int:
				return v.encodeInts(writer)
			case bool:
				return v.encodeBooleans(writer)
			case uint8, uint16, uint32, uint64, int8, int16, int32, int64, float32, float64:
				return v.encodeNumbers(writer)
			default:
				return v.encodeCustom(writer)
			}
		}
	}
	return nil
}

func (v *values) value(index uint32) interface{} {
	v.RLock()
	defer v.RUnlock()
	return v.data[index]
}

func (v *values) encodeInts(writer io.Writer) error {

	for i := range v.data {
		item, err := toolbox.ToInt(v.data[i])
		if err == nil {
			err = binary.Write(writer, binary.LittleEndian, int64(item))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *values) encodeStrings(writer io.Writer) error {
	var err error
	for i := range v.data {
		item := toolbox.AsString(v.data[i])
		length := uint32(len(item))
		if err = binary.Write(writer, binary.LittleEndian, length); err == nil {
			err = binary.Write(writer, binary.LittleEndian, []byte(item))
		}
		if err != nil {
			return err
		}
	}
	return err
}

func (v *values) encodeBytes(writer io.Writer) error {
	var err error
	for i := range v.data {
		item := toolbox.AsString(v.data[i])
		if err = binary.Write(writer, binary.LittleEndian, uint32(len(item))); err == nil {
			err = binary.Write(writer, binary.LittleEndian, []byte(item))
		}
		if err != nil {
			return err
		}
	}
	return err
}

func (v *values) encodeBooleans(writer io.Writer) error {
	var err error
	for i := range v.data {
		item := uint8(0)
		if toolbox.AsBoolean(v.data[i]) {
			item = 1
		}
		err = binary.Write(writer, binary.LittleEndian, item)
		if err != nil {
			return err
		}
	}
	return err
}

func (v *values) encodeNumbers(writer io.Writer) error {
	for i := range v.data {
		if err := binary.Write(writer, binary.LittleEndian, v.data[i]); err != nil {
			return err
		}
	}
	return nil
}

func (v *values) encodeCustom(writer io.Writer) error {
	for i := range v.data {
		encoder, ok := v.data[i].(Encoder)
		if !ok {
			return fmt.Errorf("unable to cast Encoder from %T", v.data[i])
		}
		if err := encoder.Encode(writer); err != nil {
			return err
		}
	}
	return nil
}

func (v *values) decodeStrings(reader io.Reader) error {
	var err error
	for i := range v.data {
		var length uint32
		err = binary.Read(reader, binary.LittleEndian, &length)
		if err == nil {
			var item = make([]byte, length)
			if err = binary.Read(reader, binary.LittleEndian, item); err == nil {
				v.data[i] = string(item)
			}
		}
		if err != nil {
			return err
		}
	}
	return err

}

func (v *values) decodeBytes(reader io.Reader) error {
	var err error
	for i := range v.data {
		length := uint32(0)
		if err = binary.Read(reader, binary.LittleEndian, &length); err == nil {
			var item = make([]byte, length)
			if err = binary.Read(reader, binary.LittleEndian, item); err == nil {
				v.data[i] = item
			}
		}
		if err != nil {
			return err
		}
	}
	return err
}

func (v *values) decodeInts(reader io.Reader) error {
	for i := range v.data {
		item := int64(0)
		if err := binary.Read(reader, binary.LittleEndian, &item); err != nil {
			return err
		}
		v.data[i] = item
	}
	return nil
}

func (v *values) decodeBooleans(reader io.Reader) error {
	var err error
	for i := range v.data {
		item := uint8(0)
		err = binary.Read(reader, binary.LittleEndian, &item)
		if err != nil {
			return err
		}
		v.data[i] = toolbox.AsBoolean(item == 1)
	}
	return err
}

func (v *values) decodeNumbers(reader io.Reader) error {
	for i := range v.data {
		item := reflect.New(v.Type)
		if err := binary.Read(reader, binary.LittleEndian, item.Interface()); err != nil {
			return err
		}
		v.data[i] = item.Elem().Interface()
	}
	return nil
}

func (v *values) decodeCustom(reader io.Reader) error {
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
		v.data[i] = item
	}
	return nil
}

func newValues() *values {
	return &values{
		data:     make([]interface{}, 0),
		registry: make(map[interface{}]uint32),
		RWMutex:  &sync.RWMutex{},
	}
}
