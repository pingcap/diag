package db

type Row interface {
	Scan(dest ...interface{}) error
}
