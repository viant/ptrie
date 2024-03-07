package ptrie

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"
)

const (
	//NodeTypeValue value type
	NodeTypeValue = uint8(2)
	//NodeTypeEdge edge type
	NodeTypeEdge = uint8(4)
	controlByte  = uint8(0x3f)
)

type region struct {
	offset uint32
	size   uint32 //fos node max 256, for bytes 1024
}

func (r *region) fragment(data []byte, ofSize uint32) []byte {
	return data[r.offset : r.offset+r.size*ofSize]
}

// Node represents a node
type Node struct {
	bset         Bit64Set
	Type         uint8
	ValueIndex   uint32
	prefixRegion region
	nodesRegion  region
	Prefix       []byte //24
	Nodes               //24
}

func (n *Node) Equals(d *Node) bool {

	if !bytes.Equal(n.Prefix, d.Prefix) {
		fmt.Printf("prefix diff: %s - %s", n.Prefix, d.Prefix)
		return false
	}
	if n.bset != d.bset {
		fmt.Printf("bset diff: %v - %v", n.bset, d.bset)
		return false
	}
	if n.ValueIndex != d.ValueIndex {
		fmt.Printf("ValueIndex diff: %v - %v", n.ValueIndex, d.ValueIndex)
		return false
	}

	if len(n.Nodes) != len(d.Nodes) {
		fmt.Printf("nodes diff: %v- %v", len(n.Nodes), len(d.Nodes))
		return false
	}
	for i, nn := range n.Nodes {
		if !nn.Equals(&d.Nodes[i]) {
			return false
		}
	}
	return true

}
func (n *Node) Size() int {
	ret := int(unsafe.Sizeof(*n))
	ret += len(n.Prefix) //of byte size
	for _, n := range n.Nodes {
		ret += n.Size()
	}
	return ret
}

func (n *Node) Data() []byte {
	data := make([]byte, n.Size())
	n.write(data, 0, true)
	return data
}

func (n *Node) write(data []byte, offset int, includeSelf bool) int {

	initial := offset

	if includeSelf {
		//reserve size of Bar here, but wite at the end when regions are computed
		offset += int(unsafe.Sizeof(*n))
	}
	if len(n.Prefix) > 0 {
		n.prefixRegion.size = uint32(len(n.Prefix))
		n.prefixRegion.offset = uint32(offset)
		offset += copy(data[offset:], unsafe.Slice((*byte)(unsafe.Pointer(&n.Prefix[0])), n.prefixRegion.size))
	}

	var prefixes = make([][]byte, len(n.Nodes))
	var nodes = make([]Nodes, len(n.Nodes))
	for i := range n.Nodes {
		node := &n.Nodes[i]
		prefixes[i] = node.Prefix
		nodes[i] = node.Nodes
		offset = node.write(data, offset, false)
		node.Nodes = nil
		node.Prefix = nil
	}

	n.nodesRegion.size = uint32(len(n.Nodes))
	if n.nodesRegion.size > 0 {
		n.nodesRegion.offset = uint32(offset)
		offset += copy(data[offset:], unsafe.Slice((*byte)(unsafe.Pointer(&n.Nodes[0])), int(unsafe.Sizeof(*n))*int(n.nodesRegion.size)))
	}

	for i := range n.Nodes { //restore
		node := &n.Nodes[i]
		node.Prefix = prefixes[i]
		node.Nodes = nodes[i]
	}

	if includeSelf {
		prefix := n.Prefix
		nodes := n.Nodes
		n.Prefix = nil
		n.Nodes = nil
		copy(data[initial:], unsafe.Slice((*byte)(unsafe.Pointer(n)), int(unsafe.Sizeof(*n))))
		n.Prefix = prefix
		n.Nodes = nodes
	}
	return offset
}

func (n *Node) Read(data *[]byte) {

	if n.prefixRegion.size > 0 {

		//fragment := n.prefixRegion.fragment(data, 1)
		r := n.prefixRegion
		prefix := unsafe.Slice((*byte)(unsafe.Pointer(&(*data)[r.offset : r.offset+r.size*1][0])), int(n.prefixRegion.size))
		n.Prefix = prefix
		//make([]byte, len(prefix))
		//copy(n.Prefix, prefix)
	}

	if n.nodesRegion.size > 0 {

		//fragment := n.nodesRegion.fragment(data, uint32(unsafe.Sizeof(*n)))
		r := n.nodesRegion
		nodes := unsafe.Slice((*Node)(unsafe.Pointer(&(*data)[r.offset : r.offset+r.size*uint32(unsafe.Sizeof(*n))][0])), int(n.nodesRegion.size))
		for i := range nodes {
			node := &nodes[i]
			node.Read(data)
		}
		n.Nodes = nodes
		//make([]Node, len(nodes))
		//copy(n.Nodes, nodes)
	}
}

func (n *Node) LoadNode(data []byte) {
	dest := unsafe.Slice((*byte)(unsafe.Pointer(n)), unsafe.Sizeof(*n))
	copy(dest, data)
	n.Read(&data)
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
		n.Nodes = make([]Node, 0)
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

// Encode encode node
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

// Decode decode node
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
			n.Nodes = make([]Node, nodeLength)
			for i := range n.Nodes {
				if err = n.Nodes[i].Decode(reader); err != nil {
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
