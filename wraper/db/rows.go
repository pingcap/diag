package db

type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close() error
}
