package cache

type ByteView struct {
	Bytes []byte
}
//实现Value接口
func (v ByteView) Len() int {
	return len(v.Bytes)
}

func (v ByteView) String() string {
	return string(v.Bytes)
}

func (v ByteView) ByteSlice() []byte {
	return append([]byte{}, v.Bytes...)
}