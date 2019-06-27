package ptrie

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	NodeTypeValue = uint8(2)
	NodeTypeEdge  = uint8(4)
	controlByte   = uint8(0x3f)
)

//Node represents a node
type Node struct {
	Type       uint8
	Prefix     []byte
	ValueIndex uint32
	Nodes
	bset Bit64Set
}

type merger func(prev uint32) uint32

func (n *Node) isValueType() bool {
	return n.Type&NodeTypeValue == NodeTypeValue
}

func (n *Node) isEdgeType() bool {
	return n.Type&NodeTypeEdge == NodeTypeEdge
}

func (n *Node) makeEdge() {
	n.Type = n.Type | NodeTypeEdge
}

func (n *Node) add(node *Node, merger merger) {
	if len(n.Nodes) == 0 {
		n.Nodes = make([]*Node, 0)
		n.makeEdge()
	}
	n.bset = n.bset.Put(node.Prefix[0])
	n.Nodes.add(node, merger)
}

func (n *Node) walk(parent []byte, handler func(key []byte, valueIndex uint32)) {
	prefix := append(parent, n.Prefix...)
	if n.isValueType() {
		handler(prefix, n.ValueIndex)
	}
	if !n.isEdgeType() {
		return
	}
	for _, node := range n.Nodes {
		node.walk(prefix, handler)
	}
}

func (n *Node) matchNodes(input []byte, offset int, handler func(key []byte, valueIndex uint32) bool) bool {
	hasMatch := false
	if n.isEdgeType() {
		if !n.bset.IsSet(input[offset]) {
			return false
		}
		index := n.Nodes.IndexOf(input[offset])
		if index == -1 {
			return false
		}
		if (n.Nodes)[index].match(input, offset, handler) {
			hasMatch = true
		}
	}
	return hasMatch
}

func (n *Node) match(input []byte, offset int, handler func(key []byte, valueIndex uint32) bool) bool {
	if offset >= len(input) {
		return false
	}
	if len(n.Prefix) == 0 {
		return n.matchNodes(input, offset, handler)
	}
	hasMatch := n.isValueType()
	if !bytes.HasPrefix(input[offset:], n.Prefix) {
		return false
	}
	offset += len(n.Prefix)
	if hasMatch {
		toContinue := handler(input[:offset], n.ValueIndex)
		if !toContinue {
			return hasMatch
		}
	}
	if offset >= len(input) {
		return hasMatch
	}
	if n.isEdgeType() {
		if n.matchNodes(input, offset, handler) {
			return true
		}
	}
	return hasMatch
}

//Encode encode node
func (n *Node) Encode(writer io.Writer) error {
	var err error
	if err = binary.Write(writer, binary.LittleEndian, controlByte); err == nil {
		if err = binary.Write(writer, binary.LittleEndian, n.Type); err == nil {
			if err = binary.Write(writer, binary.LittleEndian, uint32(len(n.Prefix))); err == nil {
				if err = binary.Write(writer, binary.LittleEndian, n.Prefix); err == nil {
					if n.isValueType() {
						err = binary.Write(writer, binary.LittleEndian, n.ValueIndex)
					}
					if err == nil {
						err = n.encodeNodes(writer)
					}
				}
			}
		}
	}
	return err
}

func (n *Node) size() int {
	result := 2 + 4 + len(n.Prefix)
	if n.isValueType() {
		result += 4
	}
	if n.isEdgeType() {
		result += 12
		for _, node := range n.Nodes {
			result += node.size()
		}
	}
	return result
}

func (n *Node) encodeNodes(writer io.Writer) error {
	var err error
	if !n.isEdgeType() {
		return err
	}
	if err = binary.Write(writer, binary.LittleEndian, uint64(n.bset)); err == nil {
		if err = binary.Write(writer, binary.LittleEndian, uint32(len(n.Nodes))); err == nil {
			for i := range n.Nodes {
				if err = (n.Nodes)[i].Encode(writer); err != nil {
					return err
				}
			}
		}
	}

	return err
}

//Decode decode node
func (n *Node) Decode(reader io.Reader) error {
	var err error
	var control uint8
	if err = binary.Read(reader, binary.LittleEndian, &control); err == nil {
		if control != controlByte {
			return fmt.Errorf("corrupted stream, expected control byte:%x, byt had: %x", controlByte, control)
		}
		if err = binary.Read(reader, binary.LittleEndian, &n.Type); err == nil {
			prefixLength := uint32(0)

			if err = binary.Read(reader, binary.LittleEndian, &prefixLength); err == nil {
				n.Prefix = make([]byte, prefixLength)
				if err = binary.Read(reader, binary.LittleEndian, &n.Prefix); err == nil {
					if n.isValueType() {
						err = binary.Read(reader, binary.LittleEndian, &n.ValueIndex)
					}
					if err == nil {
						err = n.decodeNodes(reader)
					}
				}
			}
		}
	}
	return err
}

func (n *Node) decodeNodes(reader io.Reader) error {
	var err error
	if !n.isEdgeType() {
		return err
	}
	nodeLength := uint32(0)
	bset := uint64(0)

	if err = binary.Read(reader, binary.LittleEndian, &bset); err == nil {
		n.bset = Bit64Set(bset)
		if err = binary.Read(reader, binary.LittleEndian, &nodeLength); err == nil {
			n.Nodes = make([]*Node, nodeLength)
			for i := range n.Nodes {
				node := &Node{}
				n.Nodes[i] = node
				if err = node.Decode(reader); err != nil {
					return err
				}
			}
		}
	}
	return err
}

func newValueNode(prefix []byte, valueIndex uint32) *Node {
	node := &Node{
		Prefix:     prefix,
		ValueIndex: valueIndex,
	}
	if len(prefix) > 0 {
		node.Type = NodeTypeValue
	}
	return node
}
