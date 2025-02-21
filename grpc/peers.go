package grpc

//
type Picker interface {
	Pick(key string) (Fetcher, bool)
}

//
type Fetcher interface {
	Fetch(groupName string, key string) ([]byte, error)
}