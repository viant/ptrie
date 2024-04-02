package ptrie

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"log"
	"path"
	"testing"
)

func loadData(URI string) (map[string]interface{}, error) {
	var result = make(map[string]interface{})
	resource := url.NewResource(URI)
	return result, resource.Decode(&result)
}

func TestNodes_add(t *testing.T) {

	parent := toolbox.CallerDirectory(3)
	useCases := []struct {
		description string
		keywords    []string
		expectURI   string
		merger      merger
	}{

		{
			description: "adding root",
			keywords:    []string{"abc", "abcd", "z", "abc", "a"},
			expectURI:   "test/add/root.json",
		},
		{
			description: "separate_nodes",
			keywords:    []string{"abc", "zyx", "mln"},
			expectURI:   "test/add/separate.json",
		},
		{
			description: "node_prefix",
			keywords:    []string{"abc", "zyx", "abcd"},
			expectURI:   "test/add/node_prefix.json",
		},
		{
			description: "edge_node",
			keywords:    []string{"abc", "ac", "zyx"},
			expectURI:   "test/add/edge.json",
		},
		{
			description: "merge_node",
			keywords:    []string{"abc", "ac", "zyx", "abc"},
			merger: func(prev uint32) uint32 {
				return prev + 100
			},
			expectURI: "test/add/merge.json",
		},
		{
			description: "multi_merge_node",
			keywords:    []string{"abcz", "abrz"},
			expectURI:   "test/add/multi_merge_node.json",
		},

		{
			description: "multi_node",
			keywords:    []string{"abcz", "abrz", "mln", "a", "abc", "ab", "abcd"},
			expectURI:   "test/add/multi_node.json",
		},
	}

	for _, useCase := range useCases {
		node := newValueNode[[]byte]([]byte("/"), 0)
		for i, keyword := range useCase.keywords {
			node.add(newValueNode[[]byte]([]byte(keyword), uint32(i+1)), useCase.merger)
		}
		expect, err := loadData(path.Join(parent, useCase.expectURI))
		if !assert.Nil(t, err, useCase.description) {
			log.Fatal(err)
		}

		if !assertly.AssertValues(t, expect, node, useCase.description) {
			_ = toolbox.DumpIndent(node, true)
		}

	}

}

func TestNodes_IndexOf(t *testing.T) {

	useCases := []struct {
		description string
		prefixes    string
		search      byte
		expectIndex int
	}{
		{
			description: "even mid test",
			prefixes:    "abdz",
			search:      []byte("b")[0],
			expectIndex: 1,
		},
		{
			description: "even last test",
			prefixes:    "abdz",
			search:      []byte("z")[0],
			expectIndex: 3,
		},
		{
			description: "even first test",
			prefixes:    "abdz",
			search:      []byte("a")[0],
			expectIndex: 0,
		},

		{
			description: "odd not found test",
			prefixes:    "abcdz",
			search:      []byte("i")[0],
			expectIndex: -1,
		},

		{
			description: "odd mid test",
			prefixes:    "abcdz",
			search:      []byte("c")[0],
			expectIndex: 2,
		},
		{
			description: "odd last test",
			prefixes:    "abcdz",
			search:      []byte("z")[0],
			expectIndex: 4,
		},
		{
			description: "odd first test",
			prefixes:    "abcdz",
			search:      []byte("a")[0],
			expectIndex: 0,
		},

		{
			description: "odd not found test",
			prefixes:    "abcdz",
			search:      []byte("i")[0],
			expectIndex: -1,
		},
	}

	for _, useCase := range useCases {
		nodes := Nodes[[]byte]{}
		for i := 0; i < len(useCase.prefixes); i++ {
			nodes = append(nodes, Node[[]byte]{Prefix: []byte(string(useCase.prefixes[i]))})
		}
		actualIndex := nodes.IndexOf(useCase.search)
		assert.Equal(t, useCase.expectIndex, actualIndex, useCase.description)

	}

}

func BenchmarkNodes_IndexOf(b *testing.B) {
	nodes := Nodes[[]byte]{}
	for i := 32; i < 98; i += 3 {
		//fmt.Printf("%v\n", i)
		nodes.add(&Node[[]byte]{Prefix: []byte{byte(i)}}, func(prev uint32) uint32 {
			return 0
		})
	}

	for i := 0; i < b.N; i++ {
		if nodes.IndexOf(byte(33)) != -1 {
			b.FailNow()
		}
		if nodes.IndexOf(byte(66)) != -1 {
			b.FailNow()
		}
		if nodes.IndexOf(byte(96)) != -1 {
			b.FailNow()
		}

		if nodes.IndexOf(byte(65)) != 11 {
			b.FailNow()
		}
		if nodes.IndexOf(byte(32)) != 0 {
			b.FailNow()
		}
	}

}
