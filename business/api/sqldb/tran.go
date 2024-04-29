package sqldb

import (
	"fmt"

	"github.com/ardanlabs/service/business/api/transaction"
	"github.com/jmoiron/sqlx"
)

// dbBeginner implements the domain transaction interface,
type dbBeginner struct {
	sqlxDB *sqlx.DB
}

// NewBeginner constructs a value that implements the database
// beginner interface.
func NewBeginner(sqlxDB *sqlx.DB) transaction.Beginner {
	return &dbBeginner{
		sqlxDB: sqlxDB,
	}
}

// Begin start a transaction and returns a value that implements
// the domain transactor interface.
func (db *dbBeginner) Begin() (transaction.Transaction, error) {
	return db.sqlxDB.Beginx()
}

// GetExtContext is a helper function that extracts the sqlx value
// from the domain transactor interface for transactional use.
func GetExtContext(tx transaction.Transaction) (sqlx.ExtContext, error) {
	ec, ok := tx.(sqlx.ExtContext)
	if !ok {
		return nil, fmt.Errorf("Transactor(%T) not of a type *sql.Tx", tx)
	}

	return ec, nil
}