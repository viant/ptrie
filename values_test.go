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
	"unsafe"
)

type useCase[T any] struct {
	description string
	values      []T
	hasError    bool
}

func test[T any](t *testing.T, uc useCase[T]) {
	values := newValues[T]()
	for _, item := range uc.values {
		_, err := values.put(item)
		assert.Nil(t, err, uc.description)
	}
	writer := new(bytes.Buffer)
	err := values.Encode(writer)
	if uc.hasError {
		assert.NotNil(t, err, uc.description)
		cloned := newValues[T]()
		cloned.useType(reflect.TypeOf(uc.values[0]))
		err = cloned.Decode(writer)
		assert.NotNil(t, err)
		return
	}
	if !assert.Nil(t, err, uc.description) {
		log.Print(err)
		return
	}
	cloned := newValues[T]()
	cloned.useType(reflect.TypeOf(uc.values[0]))
	err = cloned.Decode(writer)
	assert.Nil(t, err, uc.description)
	assert.EqualValues(t, len(values.data), len(cloned.data))
	for i := range values.data {
		assert.EqualValues(t, values.data[i], cloned.data[i], fmt.Sprintf("[%d]: %v", i, uc.description))
	}
}

func TestIntValues_Decode(t *testing.T) {
	u := useCase[int]{
		description: "int coding",
		values:      []int{0, 10, 30, 300, 4},
	}
	test(t, u)
}
func TestStringValues_Decode(t *testing.T) {
	u := useCase[string]{
		description: "string coding",
		values:      []string{"abc", "xyz", "klm", "xyz", "eee"},
	}
	test(t, u)
}
func TestInt8Values_Decode(t *testing.T) {
	u := useCase[int8]{
		description: "int8 coding",
		values:      []int8{int8(3), int8(10), int8(30), int8(121), int8(4)},
	}
	test(t, u)
}
func TestInt64Values_Decode(t *testing.T) {
	u := useCase[int64]{
		description: "int64 coding",
		values:      []int64{int64(3), int64(10), int64(88888888830), int64(121), int64(4)},
	}
	test(t, u)
}
func TestBoolValues_Decode(t *testing.T) {
	u := useCase[bool]{
		description: "bool coding",
		values:      []bool{true, false},
	}
	test(t, u)
}
func TestByteValues_Decode(t *testing.T) {
	u := useCase[[]byte]{
		description: "[]byte coding",
		values:      [][]byte{[]byte("abc"), []byte("xyz")},
	}
	test(t, u)
}
func TestCustomValues_Decode(t *testing.T) {
	u := useCase[*foo]{
		description: "custom type coding",
		values:      []*foo{{ID: 10, Name: "abc"}, {ID: 20, Name: "xyz"}},
	}
	test(t, u)
}
func TestCustomValuesErr_Decode(t *testing.T) {
	u := useCase[*bar]{
		description: "custom type error coding",
		values:      []*bar{{ID: 10, Name: "abc"}, {ID: 20, Name: "xyz"}},
		hasError:    true,
	}
	test(t, u)
}

func TestStuct_Decode(t *testing.T) {
	u := useCase[*rules]{
		description: "scalar struct type coding",
		values: []*rules{
			{
				P: []rule{{Id1: 1, Id2: 2}},
				B: []rule{{Id1: 3, Id2: 4}},
				K: []rule{{Id1: 5, Id2: 6}},
			},
		},
	}
	test(t, u)
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

type rules struct {
	P []rule
	B []rule
	K []rule
}

type rule struct {
	Id1 int32
	Id2 int32
}

func (rs *rules) Key() interface{} {
	result := 0
	if len(rs.P) > 0 {
		for i, r := range rs.P {
			result += int(r.Id1+r.Id2) * (1 + i + (i * 10))
		}
	}
	if len(rs.B) > 0 {
		for i, r := range rs.B {
			result += 100000 * int(r.Id2+r.Id1) * (1 + i + (i * 10))
		}
	}
	if len(rs.K) > 0 {
		for i, r := range rs.K {
			result += 2000000 * int(r.Id2+r.Id1) * (1 + i + (i * 10))
		}
	}
	return result
}

func (r *rules) decode(reader io.Reader, rs *[]rule) error {
	size := uint32(0)
	err := binary.Read(reader, binary.LittleEndian, &size)
	if size == 0 || err != nil {
		return err
	}
	*rs = make([]rule, size)
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(&(*rs)[0])), len(*rs)*int(unsafe.Sizeof(&(*rs)[0])))
	return binary.Read(reader, binary.LittleEndian, bytes)
}

// Decode decodes data into match rules
func (r *rules) Decode(reader io.Reader) error {
	err := r.decode(reader, &r.P)
	if err == nil {
		if err = r.decode(reader, &r.B); err == nil {
			err = r.decode(reader, &r.K)
		}
	}
	return err
}

func (r *rules) encode(rs []rule, writer io.Writer) error {
	err := binary.Write(writer, binary.LittleEndian, uint32(len(rs)))
	if len(rs) == 0 || err != nil {
		return err
	}
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(&rs[0])), len(rs)*int(unsafe.Sizeof(&rs[0])))
	return binary.Write(writer, binary.LittleEndian, &bytes)
}

// Encode encode message rules into writer
func (r *rules) Encode(writer io.Writer) error {
	err := r.encode(r.P, writer)
	if err == nil {
		if err = r.encode(r.B, writer); err == nil {
			err = r.encode(r.K, writer)
		}
	}
	return err
}
