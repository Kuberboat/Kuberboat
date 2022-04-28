package error

type KubeErrorType uint

const (
	// ErrUnknown is error caused by unknown reason.
	KubeErrUnknown KubeErrorType = iota
	// KubeErrGrpc is error occuring during grpc communication.
	KubeErrGrpc
)

// KubeError is the error type for all the internal errors in Kuberboat.
type KubeError struct {
	// Type is the type of the error.
	Type KubeErrorType
	// Message is the error massage.
	Message string
}

func (e KubeError) Error() string {
	return e.Message
}
