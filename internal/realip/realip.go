// Package realip holds the names under which the agent sends the address of
// its host, so that the agent and the server agree on them.
package realip

const (
	// Header is the HTTP header with the agent's address.
	Header = "X-Real-IP"
	// MetadataKey is the gRPC metadata key with the agent's address; gRPC
	// wants the keys in lower case.
	MetadataKey = "x-real-ip"
)
