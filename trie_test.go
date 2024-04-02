package ptrie

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/toolbox"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
)

func TestTrie_Get(t *testing.T) {

	useCases := []struct {
		description string
		keywords    map[string]int
		key         string
	}{
		{
			description: "direct_match_get",
			keywords: map[string]int{
				"abc": 1,
				"zyx": 2,
				"mln": 3,
			},
			key: "abc",
		},

		{
			description: "no match",
			keywords: map[string]int{
				"abc": 1,
				"zyx": 2,
				"mln": 3,
			},
			key: "k1",
		},
		{
			description: "no close match",
			keywords: map[string]int{
				"k2":  1,
				"zyx": 2,
				"mln": 3,
				"k":   23,
			},
			key: "k1",
		},
		{
			description: "multi_match_get",
			keywords: map[string]int{
				"abc":  1,
				"ab":   10,
				"abcd": 12,
				"abcz": 13,
				"abrz": 14,
				"zyx":  2,
				"mln":  3,
				"a":    110,
			},
			key: "abc",
		},
	}

	for i := range useCases {
		useCase := useCases[i]
		trie := New[int]()
		for k, v := range useCase.keywords {
			err := trie.Put([]byte(k), v)
			assert.Nil(t, err, useCase.description)
		}
		value, ok := trie.Get([]byte(useCase.key))
		expectedValue, epxectKey := useCase.keywords[useCase.key]
		assert.Equal(t, epxectKey, ok, useCase.description)
		if epxectKey {
			assert.Equal(t, expectedValue, value, useCase.description)
			assert.True(t, trie.Has([]byte(useCase.key)), useCase.description)
		}
	}

}

func TestTrie_MatchPrefix(t *testing.T) {
	useCases := []struct {
		description     string
		keywords        map[string]int
		matchedKeywords map[string]int
		testMultiMatch  bool
		input           string
	}{
		{
			description: "multi match",
			keywords: map[string]int{
				"abcdef":   1,
				"abcdefgh": 2,
				"abc":      3,
				"bar":      4,
				"foo":      5,
				"a":        5,
			},
			testMultiMatch: true,
			matchedKeywords: map[string]int{
				"abc": 3,
				"a":   5,
			},
			input: "abc",
		},

		{
			description: "single match",
			keywords: map[string]int{
				"abcdef":   1,
				"abcdefgh": 2,
				"abc":      3,
				"bar":      4,
				"foo":      5,
				"a":        5,
			},
			testMultiMatch: false,
			matchedKeywords: map[string]int{
				"a": 5,
			},
			input: "abc",
		},
		{
			description: "no match",
			keywords: map[string]int{
				"abcdef":   1,
				"abcdefgh": 2,
				"abc":      3,
				"bar":      4,
				"foo":      5,
				"a":        5,
			},
			testMultiMatch: false,
			input:          "zero",
		},
	}

	for i := range useCases {
		useCase := useCases[i]
		trie := New[int]()
		for k, v := range useCase.keywords {
			_ = trie.Put([]byte(k), v)
		}
		actualMatch := map[string]int{}
		onMatch := func(key []byte, value int) bool {
			actualMatch[string(key)] = value
			return useCase.testMultiMatch
		}
		hasMatch := trie.MatchPrefix([]byte(useCase.input), onMatch)
		assert.Equal(t, len(useCase.matchedKeywords) > 0, hasMatch, useCase.description)
		if len(useCase.matchedKeywords) > 0 {
			assert.Equal(t, useCase.matchedKeywords, actualMatch, useCase.description)
		}
	}
}

func TestTrie_MatchAll(t *testing.T) {
	useCases := []struct {
		description     string
		keywords        map[string]int
		matchedKeywords map[string]int
		testMultiMatch  bool
		input           string
	}{
		{
			description: "multi match",
			keywords: map[string]int{
				"abcdef":   1,
				"abcdefgh": 2,
				"abc":      3,
				"bar":      4,
				"bc":       10,
				"fo":       11,
				"foo":      12,
				"a":        5,
			},
			testMultiMatch: true,
			matchedKeywords: map[string]int{
				"abc": 3,
				"a":   5,
				"bc":  10,
				"fo":  11,
				"foo": 12,
			},
			input: "abc is foo",
		},
	}

	for i := range useCases {
		useCase := useCases[i]
		trie := New[int]()
		for k, v := range useCase.keywords {
			_ = trie.Put([]byte(k), v)
		}
		actualMatch := map[string]int{}
		onMatch := func(key []byte, value int) bool {
			actualMatch[string(key)] = value
			return useCase.testMultiMatch
		}
		hasMatch := trie.MatchAll([]byte(useCase.input), onMatch)
		assert.Equal(t, len(useCase.matchedKeywords) > 0, hasMatch, useCase.description)
		if len(useCase.matchedKeywords) > 0 {
			assert.Equal(t, useCase.matchedKeywords, actualMatch, useCase.description)
		}
	}
}

func TestTrie_MatchAllWithDecodedTrie(t *testing.T) {
	useCases := []struct {
		description     string
		keywords        map[string]int
		matchedKeywords map[string]int
		testMultiMatch  bool
		input           string
	}{
		{
			description: "multi match",
			keywords: map[string]int{
				"abcdef":   1,
				"abcdefgh": 2,
				"abc":      3,
				"bar":      4,
				"bc":       10,
				"fo":       11,
				"foo":      12,
				"a":        5,
			},
			testMultiMatch: true,
			matchedKeywords: map[string]int{
				"abc": 3,
				"a":   5,
				"bc":  10,
				"fo":  11,
				"foo": 12,
			},
			input: "abc is foo",
		},
	}

	for i := range useCases {
		useCase := useCases[i]
		trie := New[int]()
		for k, v := range useCase.keywords {
			_ = trie.Put([]byte(k), v)
		}

		readWriter := new(bytes.Buffer)
		err := trie.Encode(readWriter)
		if !assert.Nil(t, err, useCase.description) {
			continue
		}
		trie = New[int]()
		trie.UseType(reflect.TypeOf(1))

		err = trie.Decode(readWriter)
		if !assert.Nil(t, err, useCase.description) {
			continue
		}

		actualMatch := map[string]int{}
		onMatch := func(key []byte, value int) bool {
			actualMatch[string(key)] = value
			return useCase.testMultiMatch
		}
		hasMatch := trie.MatchAll([]byte(useCase.input), onMatch)
		assert.Equal(t, len(useCase.matchedKeywords) > 0, hasMatch, useCase.description)
		if len(useCase.matchedKeywords) > 0 {
			assertly.AssertValues(t, useCase.matchedKeywords, actualMatch, useCase.description)
		}
	}
}

func TestTrie_MatchAllWithDecodedSequentiallyTrie(t *testing.T) {
	useCases := []struct {
		description     string
		keywords        map[string]int
		matchedKeywords map[string]int
		testMultiMatch  bool
		input           string
	}{
		{
			description: "multi match",
			keywords: map[string]int{
				"abcdef":   1,
				"abcdefgh": 2,
				"abc":      3,
				"bar":      4,
				"bc":       10,
				"fo":       11,
				"foo":      12,
				"a":        5,
			},
			testMultiMatch: true,
			matchedKeywords: map[string]int{
				"abc": 3,
				"a":   5,
				"bc":  10,
				"fo":  11,
				"foo": 12,
			},
			input: "abc is foo",
		},
	}

	for i := range useCases {
		useCase := useCases[i]
		trie := New[int]()
		for k, v := range useCase.keywords {
			_ = trie.Put([]byte(k), v)
		}

		readWriter := new(bytes.Buffer)
		err := trie.Encode(readWriter)
		if !assert.Nil(t, err, useCase.description) {
			continue
		}
		trie = New[int]()
		trie.UseType(reflect.TypeOf(1))
		err = trie.DecodeSequentially(readWriter)
		if !assert.Nil(t, err, useCase.description) {
			continue
		}

		actualMatch := map[string]int{}
		onMatch := func(key []byte, value int) bool {
			actualMatch[string(key)] = value
			return useCase.testMultiMatch
		}
		hasMatch := trie.MatchAll([]byte(useCase.input), onMatch)
		assert.Equal(t, len(useCase.matchedKeywords) > 0, hasMatch, useCase.description)
		if len(useCase.matchedKeywords) > 0 {
			assertly.AssertValues(t, useCase.matchedKeywords, actualMatch, useCase.description)
		}
	}
}

func TestTrie_Walk(t *testing.T) {
	useCases := []struct {
		description string
		keywords    []string
	}{
		{
			description: "basic_encode",
			keywords:    []string{"abc", "zyx", "mln"},
		},
		{
			description: "prefix_encode",
			keywords:    []string{"abc", "zyx", "abcd"},
		},
		{
			description: "edge_encode",
			keywords:    []string{"abc", "ac", "zyx"},
		},
		{
			description: "merge_node",
			keywords:    []string{"abc", "ac", "zyx", "abc", "abcdefx"},
		},
	}

	for _, useCase := range useCases {
		trie := New[uint32]()
		var expect = make(map[string]uint32)
		var actual = make(map[string]uint32)
		for i, keyword := range useCase.keywords {
			expect[keyword] = uint32(i + 1)
			err := trie.Put([]byte(keyword), uint32(i+1))
			assert.Nil(t, err)
		}
		trie.Walk(func(key []byte, value uint32) bool {
			actual[string(key)] = value
			return true
		})
		assert.Equal(t, expect, actual, useCase.description)
	}
}

func TestTrie_Decode(t *testing.T) {
	useCases := []struct {
		description string
		keywords    []string
	}{
		{
			description: "basic_encode",
			keywords:    []string{"abc", "zyx", "mln"},
		},
		{
			description: "prefix_encode",
			keywords:    []string{"abc", "zyx", "abcd"},
		},
		{
			description: "edge_encode",
			keywords:    []string{"abc", "ac", "zyx"},
		},
	}

	for _, useCase := range useCases {
		trie := New[uint32]()
		for i, keyword := range useCase.keywords {
			_ = trie.Put([]byte(keyword), uint32(i+1))
		}
		writer := new(bytes.Buffer)
		err := trie.Encode(writer)
		assert.Nil(t, err, useCase.description)

		cloned := New[uint32]()
		cloned.UseType(reflect.TypeOf(uint32(0)))

		err = cloned.Decode(bytes.NewReader(writer.Bytes()))
		if !assert.Nil(t, err, useCase.description) {
			continue
		}
		actual := trieToMap(cloned)
		expect := trieToMap(trie)
		assert.EqualValues(t, expect, actual, useCase.description)
	}
}
func TestTrie_Read(t *testing.T) {
	useCases := []struct {
		description string
		keywords    []string
	}{
		{
			description: "basic_write",
			keywords:    []string{"abc", "zyx", "mln", "a"},
		},
		{
			description: "prefix_write",
			keywords:    []string{"abc", "zyx", "abcd"},
		},
		{
			description: "edge_write",
			keywords:    []string{"abc", "ac", "zyx"},
		},
	}

	for _, useCase := range useCases {
		trie := New[uint32]()
		for i, keyword := range useCase.keywords {
			_ = trie.Put([]byte(keyword), uint32(i+1))
		}
		writer := new(bytes.Buffer)
		err := trie.Write(writer)
		assert.Nil(t, err, useCase.description)

		cloned := New[uint32]()
		cloned.UseType(reflect.TypeOf(uint32(0)))

		err = cloned.Read(bytes.NewReader(writer.Bytes()))
		if !assert.Nil(t, err, useCase.description) {
			continue
		}
		actual := trieToMap(cloned)
		expect := trieToMap(trie)
		assert.EqualValues(t, expect, actual, useCase.description)
	}
}

func trieToMap[T any](trie Trie[T]) map[string]interface{} {
	var result = make(map[string]interface{})
	trie.Walk(func(key []byte, value T) bool {
		result[string(key)] = value
		return true
	})
	return result
}

type perfCase struct {
	inputs []string
	trie   Trie[bool]
	expect map[string]int
}

func buildUseCase(name string, maxLineLength int) (*perfCase, error) {
	var err error
	result := &perfCase{
		trie:   New[bool](),
		inputs: make([]string, 0),
		expect: make(map[string]int),
	}
	result.trie.UseType(reflect.TypeOf(true))
	parent := toolbox.CallerDirectory(3)
	file, err := os.Open(path.Join(parent, fmt.Sprintf("test/%v.txt", name)))
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {

		line := strings.ToLower(scanner.Text())
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		if maxLineLength > 0 && len(line) > maxLineLength {
			line = line[:maxLineLength]
		}
		words := strings.Split(line, " ")

		for i := 1; i < len(words)-1; i++ {
			word := words[i]
			if len(word) < 4 {
				continue
			}
			word = " " + word + " "
			result.expect[word]++
			_ = result.trie.Put([]byte(word), true)
		}
		result.inputs = append(result.inputs, line)
	}
	return result, nil
}

var loremLong *perfCase
var loremShort *perfCase

var hamletLong *perfCase
var hamletShort *perfCase

func init() {
	hamletShort, _ = buildUseCase("hamlet", 20)
	hamletLong, _ = buildUseCase("hamlet", 0)
	loremLong, _ = buildUseCase("lorem", 0)
	loremShort, _ = buildUseCase("lorem", 20)

}

func Test_BenchmarkBruteForce(t *testing.T) {
	testPerformanceBruteForceCase(t, loremLong)
	testPerformanceBruteForceCase(t, loremShort)
	testPerformanceBruteForceCase(t, hamletLong)
	testPerformanceBruteForceCase(t, hamletShort)

}

func Test_BenchmarkTrie(t *testing.T) {
	testPerformanceTrieCase(t, hamletLong)
	testPerformanceBruteForceCase(t, hamletShort)

}

func testPerformanceBruteForceCase(t *testing.T, useCase *perfCase) {
	var actual = make(map[string]int)
	lineTotalLen := 0
	for i := range useCase.inputs {
		line := useCase.inputs[i]
		lineTotalLen += len(line)
		for word := range useCase.expect {
			innerLine := line
			for j := 0; j < len(line); j++ {
				index := strings.Index(innerLine[j:], word)
				if index == -1 {
					break
				}
				innerLine = innerLine[index+1:]
				actual[word]++
			}
		}
	}
	fmt.Printf("avg line size: %v, words: %v\n", lineTotalLen/len(useCase.inputs), len(useCase.expect))
	assert.EqualValues(t, useCase.expect, actual)
}

func testPerformanceTrieCase(t *testing.T, useCase *perfCase) {
	var actual = make(map[string]int)
	for i := range useCase.inputs {
		line := useCase.inputs[i]
		hamletLong.trie.MatchAll([]byte(line), func(key []byte, value bool) bool {
			actual[string(key)]++
			return true
		})
	}
	assert.EqualValues(t, useCase.expect, actual)
}

func Benchmark_LoremBruteForceLong(b *testing.B) {
	useCase := loremLong
	benchmarkBruteForce(b, useCase)
}

func Benchmark_LoremTrieLong(b *testing.B) {
	useCase := loremLong
	benchmarkTrie(b, useCase)
}

func Benchmark_LoremBruteForceShort(b *testing.B) {
	useCase := loremShort
	benchmarkBruteForce(b, useCase)
}

func Benchmark_LoremTrieShort(b *testing.B) {
	useCase := loremShort
	benchmarkTrie(b, useCase)
}

func Benchmark_HamletBruteForceLong(b *testing.B) {
	useCase := hamletLong
	benchmarkBruteForce(b, useCase)
}

func Benchmark_HamletTrieLong(b *testing.B) {
	useCase := hamletLong
	benchmarkTrie(b, useCase)
}

func Benchmark_HamletBruteForceShort(b *testing.B) {
	useCase := hamletShort
	benchmarkBruteForce(b, useCase)
}

func Benchmark_HamletTrieShort(b *testing.B) {
	useCase := hamletShort
	benchmarkTrie(b, useCase)
}

func benchmarkTrie(b *testing.B, useCase *perfCase) {
	for k := 0; k < b.N; k++ {
		for i := range useCase.inputs {
			line := useCase.inputs[i]
			hamletLong.trie.MatchAll([]byte(line), func(key []byte, value bool) bool {
				return true
			})
		}
	}
}

func benchmarkBruteForce(b *testing.B, useCase *perfCase) {
	for k := 0; k < b.N; k++ {
		for i := range useCase.inputs {
			line := useCase.inputs[i]
			for word := range useCase.expect {
				innerLine := line
				for j := 0; j < len(line); j++ {
					index := strings.Index(innerLine[j:], word)
					if index == -1 {
						break
					}
					innerLine = innerLine[index+1:]
				}
			}
		}
	}
}
