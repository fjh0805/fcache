package cache
//能够将普通的函数类型（需类型转换）作为参数，也可以将结构体作为参数，
//使用更为灵活，可读性也更好，这就是接口型函数的价值
type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}