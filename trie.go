package ptrie

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"sync"
	"time"
)

// Merger represents node value merger
type Merger[T any] func(previous, next T) (merged T)

// OnMatch represents matching input handler, return value instruct trie to continue search
type OnMatch[T any] func(key []byte, value T) bool

// Visitor represents value node visitor handler
type Visitor[T any] func(key []byte, value T) bool

// Trie represents prefix tree interface
type Trie[T any] interface {
	Put(key []byte, value T) error

	Merge(key []byte, value T, merger Merger[T]) error

	Get(key []byte) (T, bool)

	Has(key []byte) bool

	//Walk all tries value nodes.
	Walk(handler Visitor[T])

	//MatchPrefix matches input prefix, ie. input: dev.domain.com, would match with trie keys like: dev, dev.domain
	MatchPrefix(input []byte, handler OnMatch[T]) bool

	//MatchAll matches input with any occurrences of tries keys.
	MatchAll(input []byte, handler OnMatch[T]) bool

	UseType(vType reflect.Type)

	//Decode decodes concurrently trie nodes and values
	Decode(reader io.Reader) error

	//DecodeSequentially decode sequentially trie nodes and values
	DecodeSequentially(reader io.Reader) error

	Encode(writer io.Writer) error

	ValueCount() int

	Write(writer io.Writer) error

	Read(reader io.Reader) error

	Root() *Node[T]
}

type trie[T any] struct {
	data   []byte
	values *values[T]
	root   *Node[T]
	bset   Bit64Set
}

func (t *trie[T]) Put(key []byte, value T) error {
	return t.Merge(key, value, nil)
}

func (t *trie[T]) Merge(key []byte, value T, merger Merger[T]) error {
	return t.merge(t.root, key, value, merger)
}

func (t *trie[T]) merge(root *Node[T], key []byte, value T, merger Merger[T]) error {
	t.bset = t.bset.Put(key[0])
	index, err := t.values.put(value)
	if err != nil {
		return err
	}
	node := newValueNode[T](key, index)
	root.add(node, func(prev uint32) uint32 {
		if merger != nil {
			prevValue := t.values.value(prev)
			newValue := merger(prevValue, value)
			index, e := t.values.put(newValue)
			if e != nil {
				err = e
			}
			return index
		}
		return node.ValueIndex
	})
	return err
}

func (t *trie[T]) Get(key []byte) (T, bool) {
	var result T
	found := false
	handler := func(k []byte, value T) bool {
		if found = len(key) == len(k); found {
			result = value
			return false
		}
		return true
	}
	has := t.root.match(key, 0, func(key []byte, valueIndex uint32) bool {
		value := t.values.value(valueIndex)
		return handler(key, value)
	})
	if !has {
		return result, has
	}
	return result, found
}

func (t *trie[T]) Has(key []byte) bool {
	_, has := t.Get(key)
	return has
}

func (t *trie[T]) MatchPrefix(input []byte, handler OnMatch[T]) bool {
	return t.match(t.root, input, handler)
}

func (t *trie[T]) Walk(handler Visitor[T]) {
	t.root.walk([]byte{}, func(key []byte, valueIndex uint32) {
		value := t.values.value(valueIndex)
		handler(key, value)
	})
}

func (t *trie[T]) UseType(vType reflect.Type) {
	t.values.useType(vType)
}

func (t *trie[T]) decodeValues(reader io.Reader, err *error, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	if e := t.values.Decode(reader); e != nil {
		*err = e
	}
}

func (t *trie[T]) decodeTrie(root *Node[T], reader io.Reader, err *error, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	if e := root.Decode(reader); e != nil {
		*err = e
	}
}

func (t *trie[T]) Decode(reader io.Reader) error {
	return t.decodeConcurrently(reader)
}

func (t *trie[T]) decodeConcurrently(reader io.Reader) error {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(2)
	trieLength := uint64(0)
	bset := uint64(0)
	err := binary.Read(reader, binary.LittleEndian, &bset)
	if err == nil {
		t.bset = Bit64Set(bset)
		if err = binary.Read(reader, binary.LittleEndian, &trieLength); err != nil {
			return err
		}
	}
	data := make([]byte, trieLength)
	if err = binary.Read(reader, binary.LittleEndian, data); err != nil {
		return err
	}
	go t.decodeTrie(t.root, bytes.NewReader(data), &err, waitGroup)

	dataReader, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	go t.decodeValues(bytes.NewReader(dataReader), &err, waitGroup)
	waitGroup.Wait()
	return err
}

func (t *trie[T]) DecodeSequentially(reader io.Reader) error {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(2)
	trieLength := uint64(0)
	bset := uint64(0)
	err := binary.Read(reader, binary.LittleEndian, &bset)
	if err == nil {
		t.bset = Bit64Set(bset)
		if err = binary.Read(reader, binary.LittleEndian, &trieLength); err != nil {
			return err
		}
	}
	//s := time.Now()
	t.decodeTrie(t.root, reader, &err, waitGroup)

	t.decodeValues(reader, &err, waitGroup)
	waitGroup.Wait()

	return err
}

func (t *trie[T]) encodeTrie(root *Node[T], writer io.Writer) error {
	return root.Encode(writer)
}

func (t *trie[T]) encodeValues(writer io.Writer) error {
	return t.values.Encode(writer)
}

func (t *trie[T]) ValueCount() int {
	return len(t.values.data)
}

func (t *trie[T]) Encode(writer io.Writer) error {
	trieSize := t.root.size()
	err := binary.Write(writer, binary.LittleEndian, uint64(t.bset))
	if err == nil {
		if err = binary.Write(writer, binary.LittleEndian, uint64(trieSize)); err == nil {
			if err = t.encodeTrie(t.root, writer); err == nil {
				err = t.encodeValues(writer)
			}
		}
	}
	return err
}

func (t *trie[T]) Write(writer io.Writer) error {
	err := binary.Write(writer, binary.LittleEndian, uint64(t.bset))
	if err == nil {
		nodesData := t.root.Data()
		trieLength := len(nodesData)
		if err == nil {
			if err = binary.Write(writer, binary.LittleEndian, uint64(trieLength)); err == nil {
				if _, err = io.Copy(writer, bytes.NewReader(nodesData)); err != nil {
					return err
				}
			}
			return t.encodeValues(writer)
		}
	}
	return err
}

func (t *trie[T]) Root() *Node[T] {
	return t.root
}

func (t *trie[T]) Read(reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	reader = bytes.NewReader(data)
	bset := uint64(0)
	err = binary.Read(reader, binary.LittleEndian, &bset)
	if err == nil {
		t.bset = Bit64Set(bset)
		trieLength := uint64(0)
		if err = binary.Read(reader, binary.LittleEndian, &trieLength); err == nil {
			data = data[16:]
			t.data = data
			t.root = &Node[T]{}
			s := time.Now()
			t.root.LoadNode(data[:trieLength])
			fmt.Printf("load new index elapsed %s\n", time.Since(s))
			reader = bytes.NewReader(data[trieLength:])
			return t.values.Decode(reader)
		}
	}
	return nil
}
func (t *trie[T]) MatchAll(input []byte, handler OnMatch[T]) bool {
	toContinue := true
	matched := false
	for i := 0; i < len(input); i++ {

		if t.bset > 0 && !t.bset.IsSet(input[i]) {
			continue
		}
		if hasMatched := t.match(t.root, input[i:], func(key []byte, value T) bool {
			toContinue = handler(key, value)
			return toContinue
		}); hasMatched {
			matched = true
		}
		if !toContinue {
			break
		}
	}
	return matched
}

func (t *trie[T]) match(root *Node[T], input []byte, handler OnMatch[T]) bool {
	return root.match(input, 0, func(key []byte, valueIndex uint32) bool {
		value := t.values.value(valueIndex)
		return handler(key, value)
	})
}

// New create new prefix trie
func New[T any]() Trie[T] {
	node := newValueNode[T]([]byte{}, 0)
	return &trie[T]{
		values: newValues[T](),
		root:   node,
	}
}
