package ptrie

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/stretchr/testify/assert"
	"hash/fnv"
	"io"
	"log"
	"reflect"
	"testing"
)

func TestValues_Decode(t *testing.T) {

	var useCases = []struct {
		description string
		values      []interface{}
		hasError    bool
	}{
		{
			description: "string coding",
			values:      []interface{}{"abc", "xyz", "klm", "xyz", "eee"},
		},
		{
			description: "int coding",
			values:      []interface{}{int(0), int(10), int(30), int(300), int(4)},
		},
		{
			description: "int8 coding",
			values:      []interface{}{int8(3), int8(10), int8(30), int8(121), int8(4)},
		},
		{
			description: "bool coding",
			values:      []interface{}{true, false},
		},
		{
			description: "[]byte coding",
			values:      []interface{}{[]byte("abc"), []byte("xyz")},
		},
		{
			description: "custom type coding",
			values:      []interface{}{&foo{ID: 10, Name: "abc"}, &foo{ID: 20, Name: "xyz"}},
		},
		{
			description: "custom type error coding",
			values:      []interface{}{&bar{ID: 10, Name: "abc"}, &bar{ID: 20, Name: "xyz"}},
			hasError:    true,
		},
	}

	for _, useCase := range useCases {
		values := newValues()
		for _, item := range useCase.values {
			_, err := values.put(item)
			assert.Nil(t, err, useCase.description)
		}
		writer := new(bytes.Buffer)
		err := values.Encode(writer)
		if useCase.hasError {
			assert.NotNil(t, err, useCase.description)
			cloned := newValues()
			cloned.useType(reflect.TypeOf(useCase.values[0]))
			err = cloned.Decode(writer)
			assert.NotNil(t, err)
			continue
		}
		if !assert.Nil(t, err, useCase.description) {
			log.Print(err)
			continue
		}
		cloned := newValues()
		cloned.useType(reflect.TypeOf(useCase.values[0]))
		err = cloned.Decode(writer)
		assert.Nil(t, err, useCase.description)
		assert.EqualValues(t, len(values.data), len(cloned.data))
		for i := range values.data {
			assert.EqualValues(t, values.data[i], cloned.data[i], fmt.Sprintf("[%d]: %v", i, useCase.description))
		}
	}

}

type foo struct {
	ID   int
	Name string
}

type bar foo

func (c *bar) Key() interface{} {
	h := fnv.New32a()
	_, _ = h.Write([]byte(c.Name))
	return c.ID + 100000*int(h.Sum32())
}

func (c *foo) Key() interface{} {
	h := fnv.New32a()
	_, _ = h.Write([]byte(c.Name))
	return c.ID + 100000*int(h.Sum32())
}

func (c *foo) Decode(reader io.Reader) error {
	id := int64(0)
	if err := binary.Read(reader, binary.BigEndian, &id); err != nil {
		return err
	}
	c.ID = int(id)
	length := uint16(0)
	err := binary.Read(reader, binary.BigEndian, &length)
	if err == nil {
		name := make([]byte, length)
		if err = binary.Read(reader, binary.BigEndian, name); err == nil {
			c.Name = string(name)
		}
	}
	return err
}

func (c *foo) Encode(writer io.Writer) error {
	err := binary.Write(writer, binary.BigEndian, int64(c.ID))
	if err != nil {
		return err
	}
	length := uint16(len(c.Name))
	if err = binary.Write(writer, binary.BigEndian, length); err == nil {
		err = binary.Write(writer, binary.BigEndian, []byte(c.Name))
	}
	return err
}
