package cache

type ByteView struct {
	bytes []byte
}

// Len returns the view's length
func (bv *ByteView) Len() int {
	return len(bv.bytes)
}

// ByteSlice returns a copy of the data as a byte slice.
func (bv *ByteView) ByteSlice() []byte {
	return cloneBytes(bv.bytes)
}

// String returns the data as a string, making a copy if necessary.
func (bv *ByteView) String() string {
	return string(bv.bytes)
}

// Deep copy
func cloneBytes(bytes []byte) []byte {
	res := make([]byte, len(bytes))
	copy(res, bytes)
	return res
}
