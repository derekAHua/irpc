package server

// contextKey is a value for use with context.WithValue.
// It's used as a pointer So it fits in an interface{} without allocation.
type contextKey struct {
	name string
}

func (k *contextKey) String() string { return "irpc context value " + k.name }

var (
	// RemoteConnContextKey is a context key. It can be used in
	// services with context.WithValue to access the connection arrived at.
	// The associated value will be of type net.Conn.
	RemoteConnContextKey = &contextKey{"remote-conn"}

	// StartRequestContextKey records the start time
	StartRequestContextKey = &contextKey{"start-parse-request"}

	// StartSendRequestContextKey records the start time
	StartSendRequestContextKey = &contextKey{"start-send-request"}

	//// TagContextKey is used to record extra info in handling services. Its value is a map[string]interface{}
	//TagContextKey = &contextKey{"service-tag"}

	// HttpConnContextKey is used to store http connection.
	HttpConnContextKey = &contextKey{"http-conn"}
)
