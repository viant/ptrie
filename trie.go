package ptrie

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"reflect"
	"sync"
)

//Merger
type Merger func(previous, next interface{}) (merged interface{})

//onMatch represents matching input handler, return value instruct trie to continue search
type OnMatch func(key []byte, value interface{}) bool

//Visitor represents value node visitor handler
type Visitor func(key []byte, value interface{}) bool

type Trie interface {
	Put(key []byte, value interface{}) error

	Merge(key []byte, value interface{}, merger Merger) error

	Get(key []byte) (interface{}, bool)

	Has(key []byte) bool

	//Walk all tries value nodes.
	Walk(handler Visitor)

	//MatchPrefix matches input prefix, ie. input: dev.domain.com, would match with trie keys like: dev, dev.domain
	MatchPrefix(input []byte, handler OnMatch) bool

	//MatchSuffix matches input suffix, ie. input: dev.domain.com, would match with trie keys like: domain.com, .com
	MatchSuffix(input []byte, handler OnMatch) bool

	//MatchAll matches input  with any occurencrs of tries keys.
	MatchAll(input []byte, handler OnMatch) bool

	UseType(vType reflect.Type)

	Decode(reader io.Reader) error

	Encode(writer io.Writer) error

	//BuildReverse flag to control to build reverse trie for suffix matches, if do not need that, set this flag to false
	//It is enabled by default
	BuildReverse(enable bool)

	ValueCount() int
}

type trie struct {
	buildReverse bool
	values       *values
	root         *Node
	reversedRoot *Node
}

func (t *trie) BuildReverse(enable bool) {
	t.buildReverse = enable
}

func (t *trie) Put(key []byte, value interface{}) error {
	return t.Merge(key, value, nil)
}

func reverseKey(key []byte) []byte {
	reverseKey := make([]byte, len(key))
	for i := 0; i < len(reverseKey)/2+1; i++ {
		j := len(reverseKey) - i - 1
		reverseKey[i], reverseKey[j] = key[j], key[i]
	}
	return reverseKey
}

func (t *trie) Merge(key []byte, value interface{}, merger Merger) error {
	err := t.merge(t.root, key, value, merger)
	if err == nil {
		if t.buildReverse {
			err = t.merge(t.reversedRoot, reverseKey(key), value, merger)
		}
	}
	return err
}

func (t *trie) merge(root *Node, key []byte, value interface{}, merger Merger) error {
	index, err := t.values.put(value)
	if err != nil {
		return err
	}
	node := newValueNode(key, index)
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

func (t *trie) Get(key []byte) (interface{}, bool) {
	var result interface{}
	found := false
	handler := func(k []byte, value interface{}) bool {
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
		return nil, has
	}
	return result, found
}

func (t *trie) Has(key []byte) bool {
	_, has := t.Get(key)
	return has
}

func (t *trie) MatchPrefix(input []byte, handler OnMatch) bool {
	return t.match(t.root, input, handler)
}

func (t *trie) MatchSuffix(input []byte, handler OnMatch) bool {
	return t.match(t.reversedRoot, reverseKey(input), func(key []byte, value interface{}) bool {
		if handler == nil {
			return true
		}
		return handler(reverseKey(key), value)
	})
}

func (t *trie) Walk(handler Visitor) {
	t.root.walk([]byte{}, func(key []byte, valueIndex uint32) {
		value := t.values.value(valueIndex)
		handler(key, value)
	})
}

func (t *trie) UseType(vType reflect.Type) {
	t.values.useType(vType)
}

func (t *trie) decodeValues(reader io.Reader, err *error, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	if e := t.values.Decode(reader); e != nil {
		*err = e
	}
}

func (t *trie) decodeTrie(root *Node, reader io.Reader, err *error, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	if e := root.Decode(reader); e != nil {
		*err = e
	}
}

func (t *trie) Decode(reader io.Reader) error {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(3)
	valuesLength := uint64(0)
	err := binary.Read(reader, binary.BigEndian, &valuesLength)
	if err != nil {
		return err
	}
	data := make([]byte, valuesLength)
	if err = binary.Read(reader, binary.BigEndian, data); err != nil {
		return err
	}
	go t.decodeValues(bytes.NewReader(data), &err, waitGroup)
	reverseTrieLength := uint64(0)
	if err = binary.Read(reader, binary.BigEndian, &reverseTrieLength); err != nil {
		return err
	}
	reverseTrieData := make([]byte, reverseTrieLength)
	if err = binary.Read(reader, binary.BigEndian, reverseTrieData); err != nil {
		return err
	}
	go t.decodeTrie(t.reversedRoot, bytes.NewReader(reverseTrieData), &err, waitGroup)
	trieData, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	go t.decodeTrie(t.root, bytes.NewReader(trieData), &err, waitGroup)
	waitGroup.Wait()
	return err
}

func (t *trie) encodeTrie(root *Node, writer io.Writer, err *error, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	if e := root.Encode(writer); e != nil {
		*err = e
	}
}

func (t *trie) encodeValues(writer io.Writer, err *error, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	if e := t.values.Encode(writer); e != nil {
		*err = e
	}
}

func (t *trie) ValueCount() int {
	return len(t.values.data)
}

func (t *trie) Encode(writer io.Writer) error {
	trieBuffer := new(bytes.Buffer)
	reverseTrieBuffer := new(bytes.Buffer)

	valueBuffer := new(bytes.Buffer)
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(3)
	var err error
	go t.encodeTrie(t.reversedRoot, reverseTrieBuffer, &err, waitGroup)
	go t.encodeTrie(t.root, trieBuffer, &err, waitGroup)
	go t.encodeValues(valueBuffer, &err, waitGroup)
	waitGroup.Wait()
	if err != nil {
		return err
	}
	if err = binary.Write(writer, binary.BigEndian, uint64(valueBuffer.Len())); err == nil {
		if err = binary.Write(writer, binary.BigEndian, valueBuffer.Bytes()); err == nil {
			if err = binary.Write(writer, binary.BigEndian, uint64(reverseTrieBuffer.Len())); err == nil {
				if err = binary.Write(writer, binary.BigEndian, reverseTrieBuffer.Bytes()); err == nil {
					err = binary.Write(writer, binary.BigEndian, trieBuffer.Bytes())
				}
			}
		}
	}
	return err
}

func (t *trie) MatchAll(input []byte, handler OnMatch) bool {
	toContinue := true
	matched := false
	for i := 0; i < len(input); i++ {

		if hasMatched := t.match(t.root, input[i:], func(key []byte, value interface{}) bool {
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

func (t *trie) match(root *Node, input []byte, handler OnMatch) bool {
	return root.match(input, 0, func(key []byte, valueIndex uint32) bool {
		value := t.values.value(valueIndex)
		return handler(key, value)
	})
}

//New create new prefix trie
func New() Trie {
	return &trie{
		buildReverse: true,
		values:       newValues(),
		root:         newValueNode([]byte{}, 0),
		reversedRoot: newValueNode([]byte{}, 0),
	}
}
