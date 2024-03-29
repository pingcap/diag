package iterator

import (
	"github.com/pingcap/diag/collector/log/item"
)

type Iterator interface {
	// parse next log item
	Next() (item.Item, error)

	// close resource
	Close() error
}

type IteratorWithPeek interface {
	Iterator

	// the next item Next() will return
	Peek() item.Item
}
