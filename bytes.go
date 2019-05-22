package ptrie

//Bytes represents byte slice
type Bytes []byte

//LastSharedIndex computes the last prefix shared indexed
func (b Bytes) LastSharedIndex(bs []byte) int {
	upperBound := len(bs)
	if upperBound >= len(b) {
		upperBound = len(b)
	}
	result := -1
	for i := 0; i < upperBound; i++ {
		if bs[i] == b[i] {
			result = i
			continue
		}
		break
	}
	return result
}
