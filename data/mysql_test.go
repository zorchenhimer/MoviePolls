package data

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/zorchenhimer/MoviePolls/common"
)

var (
	mc TestableDataConnector

	testUser  *common.User
	testMovie *common.Movie
	testCycle *common.Cycle

	userId  int
	cycleId int
	movieId int

	err error

	userFail  bool
	movieFail bool
	cycleFail bool
)

func TestMain(m *testing.M) {
	var err error
	mc, err = newMySqlConnector("root:buttslol@tcp(127.0.0.1:3306)/moviepolls?parseTime=true&loc=Local")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestMySql_AddCycle(t *testing.T) {
	future := time.Now().Local().AddDate(0, 0, 7)
	cycleId, err = mc.AddCycle(&future)
	if err != nil {
		t.Fatal(err)
	}

	if cycleId < 1 {
		t.Fatal("Invalid cycle Id returned")
	}

	t.Logf("created cycle Id: %d", cycleId)
}

func TestMySql_GetCurrentCycle(t *testing.T) {
	if cycleId < 1 {
		t.Skip("Skipping due to previous failure")
	}

	cycle, err := mc.GetCurrentCycle()
	if err != nil {
		t.Fatal(err)
	}

	if cycle == nil {
		t.Fatal("GetCurrentCycle() returned nil cycle and nil error")
	}

	if cycle.Id != cycleId {
		t.Fatalf("GetCurrentCycle() returned the wrong cycle.  Id %d vs %d", cycle.Id, cycleId)
	}
	testCycle = cycle
}

func TestMySql_AddMovie(t *testing.T) {
	if testCycle == nil {
		t.Skip("Skipping due to previous failure")
	}

	testDate := time.Now().Unix()
	movieName := fmt.Sprintf("Test Movie %d", testDate)

	// Add Movie
	m := &common.Movie{
		Name: movieName,
		Links: []string{
			fmt.Sprintf("http://example.com/1/%d", testDate),
			fmt.Sprintf("https://example.com/2/%d", testDate),
		},
		Description: fmt.Sprintf("%s description", movieName),
		CycleAdded:  &common.Cycle{Id: cycleId},
		Removed:     true,
		Approved:    false,
		Votes:       []*common.Vote{},
		Poster:      "unknown.jpg",
	}

	movieId, err = mc.AddMovie(m)
	if err != nil {
		movieFail = true
		t.Fatal(err)
	}

	if movieId < 1 {
		movieFail = true
		t.Fatal("Invalid movie Id returned")
	}

	testMovie = m
}

func TestMySql_GetMovie(t *testing.T) {
	if testMovie == nil {
		t.Skip("Skipping due to previous failure")
	}

	movie, err := mc.GetMovie(movieId)
	if err != nil {
		t.Fatal(err)
	}

	if movie == nil {
		t.Fatal("GetMovie() returned nil movie and nil error")
	}

	compareMovies(testMovie, movie, t)
}

func TestMySql_GetActiveMovies(t *testing.T) {
	if testMovie == nil {
		t.Skip("Skipping due to previous failure")
	}

	active, err := mc.GetActiveMovies()
	if err != nil {
		t.Fatal(err)
	}

	var movie *common.Movie
	for _, mov := range active {
		if mov.Id == movieId {
			movie = mov
			break
		}
	}

	compareMovies(testMovie, movie, t)
}

func TestMySql_CheckMovieExists_True(t *testing.T) {
	if testMovie == nil {
		t.Skip("Skipping due to previous failure")
	}

	if ok, err := mc.CheckMovieExists(testMovie.Name); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatal("CheckMovieExists() failed")
	}
}

func TestMySql_CheckMovieExists_False(t *testing.T) {
	val, err := mc.CheckMovieExists("doesn't exist")
	if err != nil {
		t.Fatal(err)
	}

	if val {
		t.Fatal("non existent movie apparently exists")
	}
}

func TestMySql_AddUser(t *testing.T) {
	passDate := time.Now().UTC().Truncate(time.Second)
	name := fmt.Sprintf("test_user_parts_%d", passDate.Unix())
	testUser = &common.User{
		Id:                  -1, // this should be ignored when adding.
		Name:                name,
		Password:            `"hashed" password`,
		OAuthToken:          fmt.Sprintf("%s token", name),
		Email:               fmt.Sprintf("%s@example.com", name),
		NotifyCycleEnd:      true,
		NotifyVoteSelection: true,
		Privilege:           common.PRIV_MOD,
		PassDate:            passDate,
		RateLimitOverride:   true,
		LastMovieAdd:        passDate.Add(time.Hour),
	}

	uid, err := mc.AddUser(testUser)
	if err != nil {
		t.Fatal(err)
	}

	testUser.Id = uid
	if testUser.Id == -1 {
		t.Fatal("User Id not updated")
	}
}

func TestMySql_GetUser(t *testing.T) {
	if testUser == nil || testUser.Id == -1 {
		t.Fatal("Skipping due to previous failure")
	}

	u, err := mc.GetUser(testUser.Id)
	if err != nil {
		t.Fatal(err)
	}

	if u == nil {
		t.Fatal("GetUser() returned a nil user and no error")
	}

	compareUsers(testUser, u, t)
}

func TestMySql_CheckUserExists(t *testing.T) {
	if testUser == nil || testUser.Id == -1 {
		t.Skip("Skipping due to previous failure")
	}

	exist, err := mc.CheckUserExists(testUser.Name)
	if err != nil {
		t.Fatal(err)
	}

	if !exist {
		t.Fatal("User doesn't exist when they should")
	}
}

func TestMySql_UserLogin(t *testing.T) {
	if testUser == nil || testUser.Id == -1 {
		t.Skip("Skipping due to previous failure")
	}

	u, err := mc.UserLogin(testUser.Name, testUser.Password)
	if err != nil {
		t.Fatal(err)
	}

	if u == nil {
		t.Fatal("GetUser() returned a nil user and no error")
	}

	compareUsers(testUser, u, t)
}

func TestMySql_GetUsers(t *testing.T) {
	if testUser == nil || testUser.Id == -1 {
		t.Skip("Skipping due to previous failure")
	}

	lst, err := mc.GetUsers(0, 100)
	if err != nil {
		t.Fatal(err)
	}

	if len(lst) == 0 {
		t.Fatal("GetUsers() returned no users and no error")
	}

	var u *common.User
	for _, user := range lst {
		if testUser.Id == user.Id {
			u = user
			break
		}
	}

	compareUsers(testUser, u, t)
}

func testMySql_GetUserVotes(t *testing.T) {
	t.Fatal("Not implemented")
}

func testMySql_GetPastCycles(t *testing.T) {
	t.Fatal("Not implemented")
}

func TestMySql_AddOldCycle(t *testing.T) {
	t.Fatal("Not implemented")
}

func TestMySql_AddVote(t *testing.T) {
	t.Fatal("Not implemented")
}

func TestMySql_UpdateUser(t *testing.T) {
	t.Fatal("Not implemented")
}

func TestMySql_UpdateMovie(t *testing.T) {
	t.Fatal("Not implemented")
}

func TestMySql_UpdateCycle(t *testing.T) {
	t.Fatal("Not implemented")
}

func TestMySql_UserVotedForMovie(t *testing.T) {
	t.Fatal("Not implemented")
}

func TestSql_CfgInt(t *testing.T) {
	testDate := time.Now().Unix()
	data := int(testDate)
	key := fmt.Sprintf("test int %d", testDate)

	err := mc.SetCfgInt(key, data)
	if err != nil {
		t.Fatal(err)
	}

	val, err := mc.GetCfgInt(key)
	if err != nil {
		t.Fatal(err)
	}

	if data != val {
		t.Fatalf("cfg int mismatch: %d vs %d", data, val)
	}

	// Make sure int/string variants return an error
	_, err = mc.GetCfgString(key)
	if err == nil {
		t.Fatal("GetCfgBool() did not return an error for int key")
	}

	_, err = mc.GetCfgBool(key)
	if err == nil {
		t.Fatal("GetCfgBool() did not return an error for int key")
	}

	err = mc.DeleteCfgKey(key)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSql_CfgBool(t *testing.T) {
	testDate := time.Now().Unix()
	data := true
	key := fmt.Sprintf("test bool %d", testDate)

	err := mc.SetCfgBool(key, data)
	if err != nil {
		t.Fatal(err)
	}

	val, err := mc.GetCfgBool(key)
	if err != nil {
		t.Fatal(err)
	}

	if data != val {
		t.Fatalf("cfg bool mismatch: %t vs %t", data, val)
	}

	// Make sure int/string variants return an error
	_, err = mc.GetCfgString(key)
	if err == nil {
		t.Fatal("GetCfgBool() did not return an error for bool key")
	}

	_, err = mc.GetCfgInt(key)
	if err == nil {
		t.Fatal("GetCfgInt() did not return an error for bool key")
	}

	err = mc.DeleteCfgKey(key)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMySql_CfgString(t *testing.T) {
	testDate := time.Now().Unix()
	data := "test data"
	key := fmt.Sprintf("test string %d", testDate)

	err := mc.SetCfgString(key, data)
	if err != nil {
		t.Fatal(err)
	}

	val, err := mc.GetCfgString(key)
	if err != nil {
		t.Fatal(err)
	}

	if data != val {
		t.Fatalf("cfg string mismatch: %q vs %q", data, val)
	}

	// Make sure int/bool variants return an error
	_, err = mc.GetCfgBool(key)
	if err == nil {
		t.Fatal("GetCfgBool() did not return an error for string key")
	}

	_, err = mc.GetCfgInt(key)
	if err == nil {
		t.Fatal("GetCfgInt() did not return an error for string key")
	}

	err = mc.DeleteCfgKey(key)
	if err != nil {
		t.Fatal(err)
	}
}

/*
	Cleanup
*/

func TestMySql_DeleteVote(t *testing.T) {
	if testUser == nil || testUser.Id < 1 ||
		testMovie == nil || testMovie.Id < 1 ||
		testCycle == nil || testCycle.Id < 1 {
		t.Skip("Skipping due to previous failure")
	}

	err := mc.DeleteVote(testUser.Id, testMovie.Id)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMySql_DeleteMovie(t *testing.T) {
	if movieFail || testMovie == nil || testMovie.Id < 1 {
		t.Skip("Skipping due to previous failure")
	}

	err := mc.DeleteMovie(testMovie.Id)
	if err != nil {
		t.Fatal(err)
	}
	testMovie = nil
}

func TestMySql_DeleteCycle(t *testing.T) {
	if movieFail || cycleFail || testMovie != nil || testCycle == nil || testCycle.Id < 1 {
		t.Skip("Skipping due to previous failure")
	}

	err := mc.DeleteCycle(testCycle.Id)
	if err != nil {
		t.Fatal(err)
	}
	testCycle = nil
}

func TestMySql_DeleteUser(t *testing.T) {
	if userFail || testUser == nil || testUser.Id < 1 {
		t.Skip("Skipping due to previous failure")
	}

	if err := mc.DeleteUser(testUser.Id); err != nil {
		t.Fatal(err)
	}
}
