package abstractions

//RequestHeaders represents a collection of request headers
type RequestHeaders struct {
	header
}

//NewRequestHeaders creates a new RequestHeaders
func NewRequestHeaders() *RequestHeaders {
	return &RequestHeaders{
		header{make(map[string]map[string]struct{})},
	}
}

//AddAll adds all headers from the other headers
func (r *RequestHeaders) AddAll(other *RequestHeaders) {
	if other == nil || other.headers == nil {
		return
	}
	for k, v := range other.headers {
		for k2 := range v {
			r.Add(k, k2)
		}
	}
}
