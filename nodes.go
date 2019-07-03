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
				for j := range sharedNode.Nodes {
					node.add(sharedNode.Nodes[j], nil)
				}
			}
			(*n)[index] = node
			return
		}
		node.Prefix = node.Prefix[sharedLen:]
		sharedNode.add(node, merger)
		return
	}

	// find common Prefix and merge into new edge node//new: abz, shared: abc
	sharedPrefixIndex := Bytes(sharedNode.Prefix).LastSharedIndex(node.Prefix)
	sharedNode.Prefix = sharedNode.Prefix[sharedPrefixIndex+1:]
	nodePrefix := node.Prefix[sharedPrefixIndex+1:]
	if len(nodePrefix) == 0 {
		node.add(sharedNode, nil)
		(*n)[index] = node
		return
	}

	edge := &Node{Type: NodeTypeEdge, Prefix: node.Prefix[:sharedPrefixIndex+1], Nodes: Nodes{}}
	edge.add(sharedNode, nil)
	node.Prefix = node.Prefix[sharedPrefixIndex+1:]
	edge.add(node, nil)
	(*n)[index] = edge

}

//IndexOf returns index of expectMatched byte or -1
func (n Nodes) IndexOf(b byte) int {
	lowerBoundIndex := 0
	upperBoundIndex := len(n) - 1
loop:
	if lowerBoundIndex <= upperBoundIndex {
		mediumIndex := (lowerBoundIndex + upperBoundIndex) / 2
		candidate := n[mediumIndex].Prefix[0]
		if candidate < b {
			lowerBoundIndex = mediumIndex + 1
		} else if candidate > b {
			upperBoundIndex = mediumIndex - 1
		} else {
			return mediumIndex
		}
		goto loop
	}
	return -1
}

func (n Nodes) Len() int           { return len(n) }
func (n Nodes) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n Nodes) Less(i, j int) bool { return n[i].Prefix[0] < n[j].Prefix[0] }
