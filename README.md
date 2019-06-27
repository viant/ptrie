# Trie (Prefix tree)

[![GoReportCard](https://goreportcard.com/badge/github.com/viant/ptrie)](https://goreportcard.com/report/github.com/viant/ptrie)
[![GoDoc](https://godoc.org/github.com/viant/ptrie?status.svg)](https://godoc.org/github.com/viant/ptrie)


This library is compatible with Go 1.11+

Please refer to [`CHANGELOG.md`](CHANGELOG.md) if you encounter breaking changes.

- [Motivation](#motivation)
- [Introduction](#introduction)
- [Usage](#usage)
- [Benchmark](#benchmark)

## Motivation

The goal of this project is to provide serverless prefix tree friendly implementation.
 where one function can easily building tree and publishing to some cloud storge.
Then the second load trie to perform various operations.

## Introduction


A  trie (prefix tree) is a space-optimized tree data structure  in which each node that is merged with its parent.
Unlike regular trees (where whole keys are from their beginning up to the point of inequality), the key at each node is compared chunk by chunk,


Prefix tree has the following application:
 - text document searching
 - rule based matching
 - constructing associative arrays for string keys

 
Character comparision complexity:

Brute Force: O(d n k)
Prefix Trie: O(d log(k))

Where
d: number of characters in document
n: number of keywords
k: average keyword length


### Usage

1. Building


```go

    trie := ptrie.New()
    
    for key, value := pairs {
         if err = trie.Put(key, value);err != nil {
         	log.Fatal(err)
         }
    }
    
    writer := new(bytes.Buffer)
	if err := trie.Encode(writer);err != nil {
		log.Fatal(err)
	}
	encoded := write.Bytes()
	//write encode data

```

2. Loading

```go

    //V type can be any type
    var v *V
    

    trie := ptrie.New()
    trie.UseType(reflect.TypeOf(v))
    if err := trie.Decode(reader);err != nil {
    	log.Fatal(err)
    }

```    

3. Traversing (range map)

```go

    trie.Walk(func(key []byte, value interface{}) bool {
		fmt.Printf("key: %s, value %v\n", key, value)
		return true
	})

```

3. Lookup

```go

    has := trie.Has(key)
    value, has := trie.Get(key)

```

3. MatchPrefix

```go

    var input []byte
    ...

    matched := trie.MatchPrefix(input,  func(key []byte, value interface{}) bool {
        fmt.Printf("matched: key: %s, value %v\n", key, value)
        return true 
    })

```

3. MatchAll

```go

    var input []byte
    ...

    matched := trie.MatchAll(input,  func(key []byte, value interface{}) bool {
        fmt.Printf("matched: key: %s, value %v\n", key, value)
        return true 
    })

```

#### Benchmark

The benchmark count all words that are part of the following extracts:

**[Lorem Ipsum](test/lorem.txt)**

1. Short: avg line size: 20, words: 13
2. Long: avg line size: 711, words: 551


```bash
Benchmark_LoremBruteForceShort-8    	  500000	      3646 ns/op
Benchmark_LoremTrieShort-8          	  500000	      2376 ns/op
Benchmark_LoremBruteForceLong-8     	    1000	   1612877 ns/op
Benchmark_LoremTrieLong-8           	   10000	    119990 ns/op
```

**[Hamlet](test/hamlet.txt)**

1. Short: avg line size: 20, words: 49
2. Long: avg line size: 41, words: 105

```bash
Benchmark_HamletBruteForceShort-8   	   30000	     44306 ns/op
Benchmark_HamletTrieShort-8         	  100000	     18530 ns/op
Benchmark_HamletBruteForceLong-8    	   10000	    226836 ns/op
Benchmark_HamletTrieLong-8          	   50000	     39329 ns/op
```

