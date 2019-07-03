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

func (b Bytes) Len() int           { return len(b) }
func (b Bytes) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b Bytes) Less(i, j int) bool { return b[i] < b[j] }
