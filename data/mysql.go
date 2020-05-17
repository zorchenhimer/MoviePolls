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
	register("mysql", func(connStr string) (DataConnector, error) {
		dc, err := newMySqlConnector(connStr)
		return DataConnector(dc), err
	})
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

type mysqlBool []uint8

func (mb mysqlBool) Value(def bool) bool {
	if len(mb) == 0 {
		return def
	}
	return mb[0] == 1
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

func (m *mysqlConnector) executeInsert(query string, args ...interface{}) (int, error) {
	ctx, _ := context.WithCancel(context.Background())

	conn, err := m.db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	stmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	row := stmt.QueryRowContext(ctx, args...)

	var lastId int
	err = row.Scan(&lastId)

	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("LastRowID wasn't returned")
	}

	return lastId, err
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

func (m *mysqlConnector) executeQueryScalar(query string, dest interface{}, args ...interface{}) error {
	ctx, _ := context.WithCancel(context.Background())

	conn, err := m.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	stmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, args...)

	//var value interface{}
	err = row.Scan(dest)
	if err != nil {
		return err
	}

	return err
}

type mysqlConnector struct {
	connStr string
	db      *sql.DB
}

func newMySqlConnector(connectionString string) (*mysqlConnector, error) {
	//func newMySqlConnector(connectionString string) (DataConnector, error) {
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
	conn, stmt, row, err := m.executeQueryRow("call cycle_GetCurrent()")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer stmt.Close()

	return scanCycle(row)
}

func (m *mysqlConnector) GetCycle(id int) (*common.Cycle, error) {
	return nil, fmt.Errorf("GetCycle(id) not implemented by MySQL yet")
}

func (m *mysqlConnector) GetMovie(id int) (*common.Movie, error) {
	conn, stmt, row, err := m.executeQueryRow("call movie_GetById(?)", id)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer stmt.Close()

	return scanMovie(row)
}

func (m *mysqlConnector) GetUser(id int) (*common.User, error) {
	conn, stmt, row, err := m.executeQueryRow("call user_GetById(?)", id)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer stmt.Close()

	user := &common.User{}
	var notifyEnd mysqlBool
	var notifySelected mysqlBool

	err = row.Scan(
		&user.Id,
		&user.Name,
		&user.Email,
		&notifyEnd,
		&notifySelected,
		&user.Privilege,
		&user.PassDate,
	)
	if err != nil {
		return nil, err
	}

	user.NotifyCycleEnd = notifyEnd.Value(false)
	user.NotifyVoteSelection = notifySelected.Value(false)

	return user, nil
}

func (m *mysqlConnector) GetActiveMovies() ([]*common.Movie, error) {
	conn, stmt, rows, err := m.executeQueryRows("call movie_GetActive")
	if err != nil {
		return nil, err
	}

	defer conn.Close()
	defer stmt.Close()
	defer rows.Close()

	movies := []*common.Movie{}

	for rows.Next() {
		mov, err := scanMovie(rows)
		if err != nil {
			return nil, err
		}

		movies = append(movies, mov)
	}
	return movies, nil
}

// database/sql has a Scanner interface, but it takes a single
// argument, not a list of arguments.  This means that the Scan()
// method of sql.Row and sql.Rows do not implement the *correct*
// scan method.
type rowScanner interface {
	Scan(...interface{}) error
}

func scanCycle(s rowScanner) (*common.Cycle, error) {
	cyc := &common.Cycle{}

	var start time.Time
	var end sql.NullTime

	err := s.Scan(
		&cyc.Id,
		&start,
		&end,
	)

	cyc.Start = start.Local().Round(time.Second)
	if end.Valid {
		t := end.Time.Local().Round(time.Second)
		cyc.End = &t
	}

	return cyc, err
}

func scanMovie(s rowScanner) (*common.Movie, error) {

	mov := &common.Movie{}
	cyc := &common.Cycle{}

	var links sql.NullString

	var removed mysqlBool
	var approved mysqlBool

	var start time.Time
	var end sql.NullTime

	err := s.Scan(
		&mov.Id,
		&mov.Name,
		&links,
		&mov.Description,
		&cyc.Id,
		&removed,
		&approved,
		//&mov.Watched,
		&mov.Poster,
		&start,
		&end,
	)

	if err != nil {
		return nil, err
	}

	cyc.Start = start.Local().Round(time.Second)
	if end.Valid {
		t := end.Time.Local().Round(time.Second)
		cyc.End = &t
	}

	mov.CycleAdded = cyc

	if links.Valid {
		mov.Links = strings.Split(links.String, "\n")
	}

	mov.Removed = removed.Value(false)
	mov.Approved = approved.Value(false)

	return mov, nil
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
	var value int
	err := m.executeQueryScalar("call user_VotedForMovie(?, ?)", &value, userId, movieId)
	if err != nil {
		return false, err
	}

	return value != 0, nil
}

func (m *mysqlConnector) UserLogin(name, password string) (*common.User, error) {
	var notifyCycle int
	var notifySelection int
	var oauth_str string = ""
	var oauth_pointer *string = &oauth_str

	user := &common.User{}
	conn, stmt, row, err := m.executeQueryRow("call user_Login(?, ?)", name, password)
	if err != nil {
		return nil, err
	}

	defer conn.Close()
	defer stmt.Close()

	err = row.Scan(
		&user.Id,
		&user.Name,
		&user.Password, // FIXME: remove this
		oauth_pointer,
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

	if oauth_pointer != nil {
		user.OAuthToken = oauth_str
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
	return m.executeInsert("call movie_Add(?, ?, ?, ?, ?, ?, ?, ?)",
		movie.Name,
		strings.Join(movie.Links, "\n"),
		movie.Description,
		movie.CycleAdded.Id,
		movie.Removed,
		movie.Approved,
		//movie.Watched,
		movie.Poster,
	)
}

func (m *mysqlConnector) AddCycle(end *time.Time) (int, error) {
	//fmt.Printf("local time: %s\n", time.Now().Local())
	start := time.Now().Local().Round(time.Second)
	if end != nil {
		*end = end.Local().Round(time.Second)
	}

	return m.executeInsert("call cycle_Add(?, ?)", start, end)
}

func (m *mysqlConnector) AddOldCycle(cycle *common.Cycle) (int, error) {
	return 0, fmt.Errorf("AddOldCycle() not implemented for MySql")
}

func (m *mysqlConnector) AddUser(user *common.User) (int, error) {
	var token *string
	if user.OAuthToken != "" {
		token = &user.OAuthToken
	}

	var email *string
	if user.Email != "" {
		email = &user.Email
	}

	//conn, stmt, row, err := m.executeQueryRow(
	return m.executeInsert(
		"call user_Add(?, ?, ?, ?, ?, ?, ?, ?)",
		user.Name,
		user.Password,
		//user.OAuthToken
		token,
		email,
		user.NotifyCycleEnd,
		user.NotifyVoteSelection,
		user.Privilege,
		user.PassDate)
}

func (m *mysqlConnector) AddVote(userId, movieId int) error {
	_, err := m.executeStatement("call vote_Add(?, ?)", userId, movieId)
	return err
}

func (m *mysqlConnector) DeleteVote(userId, movieId int) error {
	_, err := m.executeStatement("call vote_Delete(?, ?)", userId, movieId)
	return err
}

func (m *mysqlConnector) UpdateUser(user *common.User) error {
	return fmt.Errorf("UpdateUser() not implemented for MySql")
}

func (m *mysqlConnector) UpdateMovie(movie *common.Movie) error {
	return fmt.Errorf("UpdateMovie() not implemented for MySql")
	//_, err := m.executeStatement("call movie_Update()")
}

func (m *mysqlConnector) UpdateCycle(cycle *common.Cycle) error {
	return fmt.Errorf("UpdateCycle() not implemented for MySql")
}

func (m *mysqlConnector) CheckMovieExists(name string) (bool, error) {
	var boolVal mysqlBool

	err := m.executeQueryScalar("select fn_CheckMovieExists(?)", &boolVal, name)
	if err != nil {
		return false, err
	}

	//boolVal := mysqlBool(val.([]uint8))
	return boolVal.Value(false), nil
}

// TODO: implement start and count
func (m *mysqlConnector) GetUsers(start, count int) ([]*common.User, error) {
	ulist := []*common.User{}
	conn, stmt, rows, err := m.executeQueryRows("call user_GetAll()")
	if err != nil {
		return nil, err
	}

	defer conn.Close()
	defer stmt.Close()
	defer rows.Close()

	for rows.Next() {
		u := &common.User{}

		var notifyCycle mysqlBool
		var notifySelection mysqlBool

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

		u.NotifyCycleEnd = notifyCycle.Value(false)
		u.NotifyVoteSelection = notifySelection.Value(false)

		ulist = append(ulist, u)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return ulist, nil
}

func (m *mysqlConnector) GetCfgString(key, val string) (string, error) {
	var value sql.NullString
	err := m.executeQueryScalar("call config_GetString(?)", &value, key)
	if err != nil {
		return "", err
	}

	if value.Valid {
		return value.String, nil
	}
	//return "", fmt.Errorf("String Cfg value does not exist for key %q", key)
	return val, nil
}

func (m *mysqlConnector) GetCfgBool(key string, val bool) (bool, error) {
	var value mysqlBool
	err := m.executeQueryScalar("call config_GetBool(?)", &value, key)
	if err != nil {
		return false, err
	}

	return value.Value(val), nil
}

func (m *mysqlConnector) GetCfgInt(key string, val int) (int, error) {
	var value sql.NullInt64
	err := m.executeQueryScalar("call config_GetInt(?)", &value, key)
	if err != nil {
		return 0, err
	}

	if value.Valid {
		return int(value.Int64), nil
	}
	return val, nil // fmt.Errorf("Invalid Int64 value")
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
	return err
}

func (m *mysqlConnector) DeleteCfgKey(key string) error {
	_, err := m.executeStatement("call config_Delete(?)", key)
	return err
}

func (m *mysqlConnector) DeleteUser(userId int) error {
	_, err := m.executeStatement("call user_Delete(?)", userId)
	return err
}

func (m *mysqlConnector) DeleteMovie(movieId int) error {
	_, err := m.executeStatement("call movie_Delete(?)", movieId)
	return err
}

func (m *mysqlConnector) DeleteCycle(cycleId int) error {
	_, err := m.executeStatement("call cycle_Delete(?)", cycleId)
	return err
}

func (m *mysqlConnector) GetMoviesFromCycle(id int) ([]*common.Movie, error) {
	return nil, fmt.Errorf("GetMoviesFromCycle() not implemented for MySQL")
}

func (m *mysqlConnector) SearchMovieTitles(query string) ([]*common.Movie, error) {
	return nil, fmt.Errorf("SearchMovieTitles() not implemented for MySQL")
}

func (m *mysqlConnector) DecayVotes(age int) error {
	return fmt.Errorf("DecayVotes() not implemented for MySQL")
}


func (m *mysqlConnector) Test_GetUserVotes(userId int) ([]*common.Vote, error) {
	return nil, fmt.Errorf("Test_GetUserVotes() not implemented for MySQL")
}
