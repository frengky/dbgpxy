package dbgpxy

// IDERepository represent a way to find registered IDE by key
type IDERepository interface {
	FindByKey(key string) (IDE, error)
}

type defaultIDERepository struct {
	storage IDEStorage
}

// NewIDERepository create a default implementation of IDERepository
func NewIDERepository(storage IDEStorage) IDERepository {
	return &defaultIDERepository{
		storage: storage,
	}
}

func (r *defaultIDERepository) FindByKey(key string) (IDE, error) {
	return r.storage.Get(key)
}
