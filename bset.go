package ptrie

// Copy of https://go.dev/src/unicode/graphic.go#L8
// Bit masks for each code point under U+0100, for fast lookup.
const (
	pC     = 1 << iota // a control character.
	pP                 // a punctuation character.
	pN                 // a numeral.
	pS                 // a symbolic character.
	pZ                 // a spacing character.
	pLu                // an upper-case letter.
	pLl                // a lower-case letter.
	pp                 // a printable character according to Go's definition.
	pg     = pp | pZ   // a graphical character according to the Unicode definition.
	pLo    = pLl | pLu // a letter that is neither upper nor lower case.
	pLmask = pLo
)

//End copy

const (
	bit64SetSize        = uint8(64)
	oneBitSet    uint64 = 1
)

// Bit64Set represent 64bit set
type Bit64Set uint64

// Put creates a new bit set for supplied value, value and not be grater than 64
func (s Bit64Set) Put(value uint8) Bit64Set {
	value = value ^ 97
	result := oneBitSet<<(value%bit64SetSize) | uint64(s)
	return Bit64Set(result)
}

// IsSet returns true if bit is set
func (s Bit64Set) IsSet(value uint8) bool {
	value = value ^ 97
	index := oneBitSet << (value % bit64SetSize)
	return uint64(s)&index > 0
}

/*
GOROOT=/Users/vcarey/go/go1.21.7 #gosetup
GOPATH=/Users/vcarey/Projects/go #gosetup
/Users/vcarey/go/go1.21.7/bin/go test -v ./... -bench . -run ^$
goos: darwin
goarch: arm64
pkg: github.com/viant/ptrie
BenchmarkNodes_IndexOf
BenchmarkNodes_IndexOf-12             	48936063	        22.68 ns/op
Benchmark_LoremBruteForceLong
Benchmark_LoremBruteForceLong-12      	     652	   1834911 ns/op
Benchmark_LoremTrieLong
Benchmark_LoremTrieLong-12            	   28130	     40580 ns/op
Benchmark_LoremBruteForceShort
Benchmark_LoremBruteForceShort-12     	  272608	      4400 ns/op
Benchmark_LoremTrieShort
Benchmark_LoremTrieShort-12           	 1382800	       864.4 ns/op
Benchmark_HamletBruteForceLong
Benchmark_HamletBruteForceLong-12     	    3484	    344537 ns/op
Benchmark_HamletTrieLong
Benchmark_HamletTrieLong-12           	  142606	      8388 ns/op
Benchmark_HamletBruteForceShort
Benchmark_HamletBruteForceShort-12    	   14144	     84151 ns/op
Benchmark_HamletTrieShort
Benchmark_HamletTrieShort-12          	  292179	      4002 ns/op
PASS
ok  	github.com/viant/ptrie	13.621s

Process finished with the exit code 0

*/
