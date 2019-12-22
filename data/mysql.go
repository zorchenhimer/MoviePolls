package data

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zorchenhimer/MoviePolls/common"
)

func init() {
	//register("mysql", newMySqlConnector)
}

type resultError struct {
	//rows int64
	err error
}

func newResultError(err error) resultError {
	return resultError{err: err}
}

func (re resultError) LastInsertId() (int64, error) {
	return 0, re.err
}

func (re resultError) RowsAffected() (int64, error) {
	return 0, re.err
}

func (m *mysqlConnector) executeStatement(query string, args ...interface{}) (sql.Result, error) {
	ctx, _ := context.WithCancel(context.Background())

	conn, err := m.db.Conn(ctx)
	if err != nil {
		return newResultError(err), err
	}
	defer conn.Close()

	stmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		return newResultError(err), err
	}
	defer stmt.Close()

	return stmt.ExecContext(ctx, args...)
}

type mysqlConnector struct {
	connStr string
	db      *sql.DB
}

//func newMySqlConnector(connectionString string) (common.DataConnector, error) {
func newMySqlConnector(connectionString string) (*mysqlConnector, error) {
	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, err
	}

	ctr := &mysqlConnector{
		connStr: connectionString,
		db:      db,
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return ctr, nil
}

func (m *mysqlConnector) CheckUserExists(name string) (bool, error) {
	ctx, _ := context.WithCancel(context.Background())

	conn, err := m.db.Conn(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	stmt, err := conn.PrepareContext(ctx, "select fn_CheckUserExists(?)")
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	var exists int
	err = stmt.QueryRowContext(ctx, name).Scan(&exists)
	if err != nil {
		return false, err
	}

	if exists == 1 {
		return true, nil
	}
	return false, nil
}

func (m *mysqlConnector) UserLogin(name, password string) (*common.User, error) {
	ctx, _ := context.WithCancel(context.Background())

	conn, err := m.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	stmt, err := conn.PrepareContext(ctx, "call user_Login(?, ?)")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var notifyCycle int
	var notifySelection int
	var oauth *string

	user := &common.User{}
	err = stmt.QueryRowContext(ctx, name, password).Scan(
		&user.Id,
		&user.Name,
		&user.Password,
		&oauth,
		&user.Email,
		&notifyCycle,
		&notifySelection,
		&user.Privilege,
		&user.PassDate,
	)
	switch {
	case err == sql.ErrNoRows:
		return nil, fmt.Errorf("Invalid login credentials")
	case err != nil:
		return nil, err
	}

	return user, nil
}

func (m *mysqlConnector) AddMovie(movie *common.Movie) (int, error) {
	res, err := m.executeStatement("call movie_Add(?, ?, ?, ?, ?, ?, ?, ?)",
		movie.Name,
		strings.Join(movie.Links, "\n"),
		movie.Description,
		movie.CycleAdded.Id,
		movie.Removed,
		movie.Approved,
		movie.Watched,
		movie.Poster,
	)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (m *mysqlConnector) AddCycle(end *time.Time) (int, error) {
	res, err := m.executeStatement("call cycle_Add(?, ?)", time.Now(), end)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

//func (m *mysqlConnector) GetCurrentCycle() (*common.Cycle, error) {
//}

