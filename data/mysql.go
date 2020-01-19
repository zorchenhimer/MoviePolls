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
	register("mysql", newMySqlConnector)
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

func (m *mysqlConnector) executeQueryRow(query string, args ...interface{}) (*sql.Conn, *sql.Stmt, *sql.Row, error) {
	ctx, _ := context.WithCancel(context.Background())

	conn, err := m.db.Conn(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	stmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		conn.Close()
		return nil, nil, nil, err
	}

	return conn, stmt, stmt.QueryRowContext(ctx, args...), nil
}

func (m *mysqlConnector) executeQueryRows(query string, args ...interface{}) (*sql.Conn, *sql.Stmt, *sql.Rows, error) {
	ctx, _ := context.WithCancel(context.Background())

	conn, err := m.db.Conn(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	stmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		conn.Close()
		return nil, nil, nil, err
	}

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		stmt.Close()
		conn.Close()
		return nil, nil, nil, err
	}
	return conn, stmt, rows, nil
}

func (m *mysqlConnector) executeQueryScalar(query string, args ...interface{}) (interface{}, error) {
	ctx, _ := context.WithCancel(context.Background())

	conn, err := m.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	stmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, args...)

	var value interface{}
	err = row.Scan(&value)
	if err != nil {
		return nil, err
	}

	return value, err
}

type mysqlConnector struct {
	connStr string
	db      *sql.DB
}

//func newMySqlConnector(connectionString string) (*mysqlConnector, error) {
func newMySqlConnector(connectionString string) (DataConnector, error) {
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

func (m *mysqlConnector) GetCurrentCycle() (*common.Cycle, error) {
	return nil, fmt.Errorf("GetCurrentCycle() not implemented for MySql")
}

func (m *mysqlConnector) GetMovie(id int) (*common.Movie, error) {
	return nil, fmt.Errorf("GetMovie() not implemented for MySql")
}

func (m *mysqlConnector) GetUser(id int) (*common.User, error) {
	return nil, fmt.Errorf("GetUser() not implemented for MySql")
}

func (m *mysqlConnector) GetActiveMovies() ([]*common.Movie, error) {
	return nil, fmt.Errorf("GetActiveMovies() not implemented for MySql")
}

func (m *mysqlConnector) GetUserVotes(userId int) ([]*common.Movie, error) {
	return nil, fmt.Errorf("GetUserVotes() not implemented for MySql")
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

func (m *mysqlConnector) UserVotedForMovie(userId, movieId int) (bool, error) {
	return false, fmt.Errorf("UserVotedForMovie() not implemented for MySql")
}

func (m *mysqlConnector) UserLogin(name, password string) (*common.User, error) {
	var notifyCycle int
	var notifySelection int
	var oauth *string

	user := &common.User{}
	conn, stmt, row, err := m.executeQueryRow("call user_Login(?, ?)", name, password)
	defer conn.Close()
	defer stmt.Close()

	if err != nil {
		return nil, err
	}

	err = row.Scan(
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

	if notifyCycle == 1 {
		user.NotifyCycleEnd = true
	}

	if notifySelection == 1 {
		user.NotifyVoteSelection = true
	}

	return user, nil
}

func (m *mysqlConnector) GetPastCycles(start, end int) ([]*common.Cycle, error) {
	return nil, fmt.Errorf("GetPastCycles() not implemented for MySql")
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

func (m *mysqlConnector) AddOldCycle(cycle *common.Cycle) (int, error) {
	return 0, fmt.Errorf("AddOldCycle() not implemented for MySql")
}

func (m *mysqlConnector) AddUser(user *common.User) (int, error) {
	return 0, fmt.Errorf("AddUser() not implemented for MySql")
}

func (m *mysqlConnector) AddVote(userId, movieId int) error {
	return fmt.Errorf("AddVote() not implemented for MySql")
}

func (m *mysqlConnector) UpdateUser(user *common.User) error {
	return fmt.Errorf("UpdateUser() not implemented for MySql")
}

func (m *mysqlConnector) UpdateMovie(movie *common.Movie) error {
	return fmt.Errorf("UpdateMovie() not implemented for MySql")
}

func (m *mysqlConnector) UpdateCycle(cycle *common.Cycle) error {
	return fmt.Errorf("UpdateCycle() not implemented for MySql")
}

func (m *mysqlConnector) CheckMovieExists(name string) (bool, error) {
	return false, fmt.Errorf("CheckMovieExists() not implemented for MySql")
}

func (m *mysqlConnector) GetUsers(start, count int) ([]*common.User, error) {
	ulist := make([]*common.User, count)
	conn, stmt, rows, err := m.executeQueryRows("call user_GetAll()")
	if err != nil {
		return nil, err
	}

	defer conn.Close()
	defer stmt.Close()
	defer rows.Close()

	for rows.Next() {
		u := &common.User{}

		var notifyCycle int
		var notifySelection int

		err := rows.Scan(
			&u.Id,
			&u.Name,
			&u.Email,
			&notifyCycle,
			&notifySelection,
			&u.Privilege,
			&u.PassDate,
		)

		if err != nil {
			return nil, err
		}

		if notifyCycle == 1 {
			u.NotifyCycleEnd = true
		}

		if notifySelection == 1 {
			u.NotifyVoteSelection = true
		}

		ulist = append(ulist, u)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return ulist, nil
}

func (m *mysqlConnector) GetCfgString(key string) (string, error) {
	val, err := m.executeQueryScalar("call config_GetString(?)", key)
	if err != nil {
		return "", err
	}
	return val.(string), nil
}

func (m *mysqlConnector) GetCfgBool(key string) (bool, error) {
	val, err := m.executeQueryScalar("call config_GetBool(?)", key)
	if err != nil {
		return false, err
	}

	intVal := val.(int)
	if intVal == 1 {
		return true, nil
	}
	return false, nil

	//return val.(string), nil
}

func (m *mysqlConnector) GetCfgInt(key string) (int, error) {
	val, err := m.executeQueryScalar("call config_GetInt(?)", key)
	if err != nil {
		return 0, err
	}

	return val.(int), nil
}

func (m *mysqlConnector) SetCfgString(key, value string) error {
	_, err := m.executeStatement("call config_SetString(?, ?)", key, value)
	if err != nil {
		return err
	}
	return nil
}

func (m *mysqlConnector) SetCfgInt(key string, value int) error {
	_, err := m.executeStatement("call config_SetInt(?, ?)", key, value)
	if err != nil {
		return err
	}
	return nil
}

func (m *mysqlConnector) SetCfgBool(key string, value bool) error {
	_, err := m.executeStatement("call config_SetBool(?, ?)", key, value)
	if err != nil {
		return err
	}
	return nil
}

func (m *mysqlConnector) DeleteCfgKey(key string) error {
	return fmt.Errorf("DeleteCfgKey() not implemented for MySql")
}
