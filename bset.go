package ptrie

const (
	bit64SetSize        = uint8(64)
	oneBitSet    uint64 = 1
)

//Bit64Set represent 64bit set
type Bit64Set uint64

//Put creates a new bit set for supplied value, value and not be grater than 64
func (s Bit64Set) Put(value uint8) Bit64Set {
	result := oneBitSet<<(value%bit64SetSize) | uint64(s)
	return Bit64Set(result)
}

//IsSet returns true if bit is set
func (s Bit64Set) IsSet(value uint8) bool {
	index := oneBitSet << (value % bit64SetSize)
	return uint64(s)&index > 0
}
