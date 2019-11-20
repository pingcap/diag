package db

import (
	"github.com/jinzhu/gorm"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

const (
	SQLITE = "sqlite3"
)

// For decoupling with sql.DB to be friendly with unit test
type DB interface {
	Where(query interface{}, args ...interface{}) DB
	CreateTable(models ...interface{}) DB
	Table(name string) DB
	Take(out interface{}, where ...interface{}) DB
	Create(value interface{}) DB
	Find(out interface{}, where ...interface{}) DB
	Save(value interface{}) DB
	Error() error
	RecordNotFound() bool
	HasTable(value interface{}) bool
	FirstOrCreate(out interface{}, where ...interface{}) DB
	Delete(value interface{}, where ...interface{}) DB
	Offset(offset interface{}) DB
	Limit(limit interface{}) DB
	Count(value interface{}) DB
	Order(value interface{}, reorder ...bool) DB
	Update(attrs ...interface{}) DB
	Updates(values interface{}, ignoreProtectedAttrs ...bool) DB
	Model(value interface{}) DB
	Close() error
	Debug() DB
	AutoMigrate(values ...interface{}) DB
}

// Open sqlite.db and return DB interface instead of a struct
func Open(fp string) (DB, error) {
	if ins, err := gorm.Open(SQLITE, fp); err == nil {
		// ins.LogMode(true)
		return wrap(ins), nil
	} else {
		return nil, err
	}
}

func OpenDebug(fp string) (DB, error) {
	db, err := Open(fp)
	if err == nil {
		return db.Debug(), nil
	} else {
		return nil, err
	}
}

func wrap(ins *gorm.DB) DB {
	return &wrapedDB{ins}
}

type wrapedDB struct {
	*gorm.DB
}

func (db *wrapedDB) Debug() DB {
	return wrap(db.DB.Debug())
}

func (db *wrapedDB) Model(value interface{}) DB {
	return wrap(db.DB.Model(value))
}

func (db *wrapedDB) Table(name string) DB {
	return wrap(db.DB.Table(name))
}

func (db *wrapedDB) Where(query interface{}, args ...interface{}) DB {
	return wrap(db.DB.Where(query, args...))
}

func (db *wrapedDB) Offset(offset interface{}) DB {
	return wrap(db.DB.Offset(offset))
}

func (db *wrapedDB) Limit(limit interface{}) DB {
	return wrap(db.DB.Limit(limit))
}

func (db *wrapedDB) Order(value interface{}, reorder ...bool) DB {
	return wrap(db.DB.Order(value, reorder...))
}

func (db *wrapedDB) Update(attrs ...interface{}) DB {
	return wrap(db.DB.Update(attrs...))
}

func (db *wrapedDB) Updates(values interface{}, ignoreProtectedAttrs ...bool) DB {
	return wrap(db.DB.Updates(values, ignoreProtectedAttrs...))
}

func (db *wrapedDB) Count(value interface{}) DB {
	return wrap(db.DB.Count(value))
}

func (db *wrapedDB) Take(out interface{}, where ...interface{}) DB {
	return wrap(db.DB.Take(out, where...))
}

func (db *wrapedDB) FirstOrCreate(out interface{}, where ...interface{}) DB {
	return wrap(db.DB.FirstOrCreate(out, where...))
}

func (db *wrapedDB) Find(out interface{}, where ...interface{}) DB {
	return wrap(db.DB.Find(out, where...))
}

func (db *wrapedDB) Save(value interface{}) DB {
	return wrap(db.DB.Save(value))
}

func (db *wrapedDB) Delete(value interface{}, where ...interface{}) DB {
	return wrap(db.DB.Delete(value, where...))
}

func (db *wrapedDB) CreateTable(models ...interface{}) DB {
	return wrap(db.DB.CreateTable(models...))
}

func (db *wrapedDB) Create(value interface{}) DB {
	return wrap(db.DB.Create(value))
}

func (db *wrapedDB) Error() error {
	return db.DB.Error
}

func (db *wrapedDB) AutoMigrate(values ...interface{}) DB {
	return wrap(db.DB.AutoMigrate(values))
}

// If the error is a not found error.
func IsNotFound(err error) bool {
	return gorm.IsRecordNotFoundError(err)
}
