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

func (a Bytes) Len() int           { return len(a) }
func (a Bytes) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Bytes) Less(i, j int) bool { return a[i] < a[j] }
