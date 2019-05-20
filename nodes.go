package ptrie

import (
	"bytes"
	"sort"
)

//Nodes represents node slice
type Nodes []*Node

func (n *Nodes) add(node *Node, merger merger) {
	index := n.IndexOf(node.Prefix[0])
	if index == -1 {
		*n = append(*n, node)
		sort.Sort(*n)
		return
	}
	sharedNode := (*n)[index]
	if bytes.HasPrefix(node.Prefix, sharedNode.Prefix) { //new: abcd, shared: abc
		sharedLen := len(sharedNode.Prefix)
		if len(sharedNode.Prefix) == len(node.Prefix) { //override
			if merger != nil && node.isValueType() && sharedNode.isValueType() {
				node.ValueIndex = merger(sharedNode.ValueIndex)
			}
			if sharedNode.isEdgeType() {
				if !node.isEdgeType() {
					node.makeEdge()
				}
				*node.Nodes = append(*(node.Nodes), (*sharedNode.Nodes)...)
			}
			(*n)[index] = node
			return
		}
		node.Prefix = node.Prefix[sharedLen:]
		sharedNode.add(node, merger)
		return
	} else {

		// find common Prefix and merge into new edge node//new: abz, shared: abc
		sharedPrefixIndex := Bytes(sharedNode.Prefix).LastSharedIndex(node.Prefix)
		sharedNode.Prefix = sharedNode.Prefix[sharedPrefixIndex+1:]
		nodePrefix := node.Prefix[sharedPrefixIndex+1:]
		if len(nodePrefix) == 0 {
			node.add(sharedNode, nil)
			(*n)[index] = node
			return
		}

		edge := &Node{Type: NodeTypeEdge, Prefix: node.Prefix[:sharedPrefixIndex+1], Nodes: &Nodes{}}
		edge.add(sharedNode, nil)
		node.Prefix = node.Prefix[sharedPrefixIndex+1:]
		edge.add(node, nil)
		(*n)[index] = edge
	}
}

//IndexOf returns index of expectMatched byte or -1
func (n Nodes) IndexOf(b byte) int {
	lowerBoundIndex := 0
	upperBoundIndex := len(n) - 1
	for lowerBoundIndex <= upperBoundIndex {
		mediumIndex := (lowerBoundIndex + upperBoundIndex) / 2
		if n[mediumIndex].Prefix[0] < b {
			lowerBoundIndex = mediumIndex + 1
		} else {
			upperBoundIndex = mediumIndex - 1
		}
	}
	if lowerBoundIndex == len(n) || n[lowerBoundIndex].Prefix[0] != b {
		return -1
	}
	return lowerBoundIndex
}

func (a Nodes) Len() int           { return len(a) }
func (a Nodes) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Nodes) Less(i, j int) bool { return a[i].Prefix[0] < a[j].Prefix[0] }
