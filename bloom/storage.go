package bloom

// StorageType is storage types
type StorageType int

const (
	// Redis type
	Redis StorageType = iota
)

// storage is an interface bloom filter backend storage needs to implement.
type storage interface {
	Init(uint, uint) error
	Append([]byte) error
	Exists([]byte) (bool, error)
	ExistsAndAppend([]byte) (bool, error)
}
