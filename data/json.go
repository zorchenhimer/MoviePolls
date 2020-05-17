package data

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	//"runtime/debug"

	"github.com/zorchenhimer/MoviePolls/common"
)

//type jsonCycle
type jsonMovie struct {
	Id             int
	Name           string
	Links          []string
	Description    string
	CycleAddedId   int
	CycleWatchedId int
	Removed        bool
	Approved       bool
	Poster         string
}

func (j *jsonConnector) newJsonMovie(movie *common.Movie) jsonMovie {
	//fmt.Println("newJsonMovie()")
	currentCycle := j.currentCycle()
	cycleId := 0
	if currentCycle != nil {
		cycleId = currentCycle.Id
	}

	cycleWatched := 0
	if movie.CycleWatched != nil {
		cycleWatched = movie.CycleWatched.Id
	}

	id := j.nextMovieId()
	//fmt.Printf("newJsonMovie(): %d\n", id)

	//debug.PrintStack()

	return jsonMovie{
		Id:             id,
		Name:           movie.Name,
		Links:          movie.Links,
		Description:    movie.Description,
		CycleAddedId:   cycleId,
		CycleWatchedId: cycleWatched,
		Removed:        movie.Removed,
		Approved:       movie.Approved,
		Poster:         movie.Poster,
	}
}

type jsonVote struct {
	UserId  int
	MovieId int
	CycleId int
}

type jsonCycle struct {
	Id      int
	Start   time.Time
	End     *time.Time
	Watched []int
}

func (j *jsonConnector) newJsonCycle(cycle *common.Cycle) jsonCycle {
	watched := []int{}
	if cycle.Watched != nil {
		for _, movie := range cycle.Watched {
			watched = append(watched, movie.Id)
		}
	}

	return jsonCycle{
		Id:      cycle.Id,
		Start:   cycle.Start,
		End:     cycle.End,
		Watched: watched,
	}
}

type jsonConnector struct {
	filename string `json:"-"`
	lock     *sync.RWMutex

	//CurrentCycle int

	Cycles []*common.Cycle
	Movies []jsonMovie
	Users  []*common.User
	Votes  []jsonVote

	//Settings Configurator
	Settings map[string]configValue
}

func init() {
	register("json", func(connStr string) (DataConnector, error) {
		dc, err := newJsonConnector(connStr)
		return DataConnector(dc), err
	})
}

func newJsonConnector(filename string) (*jsonConnector, error) {
	if common.FileExists(filename) {
		return loadJson(filename)
	}

	j := &jsonConnector{
		filename: filename,
		lock:     &sync.RWMutex{},
		//CurrentCycle: 0,
		Settings: map[string]configValue{
			"Active": configValue{CVT_BOOL, true},
		},
	}

	return j, j.save()
}

func loadJson(filename string) (*jsonConnector, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	data := &jsonConnector{}
	err = json.Unmarshal(raw, data)
	if err != nil {
		return nil, fmt.Errorf("Unable to read JSON data: %v", err)
	}

	data.filename = filename
	data.lock = &sync.RWMutex{}

	return data, nil
}

func (j *jsonConnector) save() error {
	raw, err := json.MarshalIndent(j, "", " ")
	if err != nil {
		return fmt.Errorf("Unable to marshal JSON data: %v", err)
	}

	err = ioutil.WriteFile(j.filename, raw, 0777)
	if err != nil {
		return fmt.Errorf("Unable to write JSON data: %v", err)
	}

	return nil
}

/*
   On determining the current cycle.

   Should the current cycle have an end date?
   If so, this would be the automatic end date for the cycle.
   If not, only the current cycle would have an end date, which would define
   the current cycle as the cycle without an end date.
*/
func (j *jsonConnector) currentCycle() *common.Cycle {
	now := time.Now().Local().Round(time.Second)

	for _, c := range j.Cycles {
		if c.End == nil || c.End.After(now) {
			return c
		}
	}
	return nil
}

func (j *jsonConnector) GetCurrentCycle() (*common.Cycle, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	return j.currentCycle(), nil
}

func (j *jsonConnector) GetCycle(id int) (*common.Cycle, error) {
	for _, c := range j.Cycles {
		if c.Id == id {
			return c, nil
		}
	}

	return nil, fmt.Errorf("Cycle not found with ID %d", id)
}

func (j *jsonConnector) AddCycle(end *time.Time) (int, error) {
	j.lock.Lock()
	defer j.lock.Unlock()

	if j.Cycles == nil {
		j.Cycles = []*common.Cycle{}
	}

	c := &common.Cycle{
		Id:    j.nextCycleId(),
		Start: time.Now(),
		End:   end,
	}

	j.Cycles = append(j.Cycles, c)

	return c.Id, j.save()
}

func (j *jsonConnector) AddOldCycle(c *common.Cycle) (int, error) {
	j.lock.Lock()
	defer j.lock.Unlock()

	if j.Cycles == nil {
		j.Cycles = []*common.Cycle{}
	}

	c.Id = j.nextCycleId()

	j.Cycles = append(j.Cycles, c)
	return c.Id, j.save()
}

func (j *jsonConnector) nextCycleId() int {
	highest := 0
	for _, c := range j.Cycles {
		if c.Id > highest {
			highest = c.Id
		}
	}
	return highest + 1
}

func (j *jsonConnector) nextMovieId() int {
	highest := 0
	for _, m := range j.Movies {
		if m.Id >= highest {
			highest = m.Id
		}
	}
	return highest + 1
}

func (j *jsonConnector) AddMovie(movie *common.Movie) (int, error) {
	j.lock.Lock()
	defer j.lock.Unlock()

	if j.Movies == nil {
		j.Movies = []jsonMovie{}
	}

	m := j.newJsonMovie(movie)
	//fmt.Printf("Adding movie %s\n", m.String())
	j.Movies = append(j.Movies, m)

	//fmt.Printf("AddMovie() ID: %d\n", m.Id)
	return m.Id, j.save()
}

func (j *jsonConnector) GetMovie(id int) (*common.Movie, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	movie := j.findMovie(id)
	if movie == nil {
		return nil, fmt.Errorf("Movie with ID %d not found.", id)
	}

	movie.Votes = j.findVotes(movie)
	return movie, nil
}

func (j *jsonConnector) GetActiveMovies() ([]*common.Movie, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	movies := []*common.Movie{}

	for _, m := range j.Movies {
		mov, _ := j.GetMovie(m.Id)
		if mov != nil && m.CycleWatchedId == 0 {
			movies = append(movies, mov)
		}
	}

	return movies, nil
}

type sortableCycle []*common.Cycle

func (s sortableCycle) Len() int { return len(s) }

// sort in reverse
func (s sortableCycle) Less(i, j int) bool { return s[i].Id > s[j].Id }
func (s sortableCycle) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (j *jsonConnector) GetPastCycles(start, end int) ([]*common.Cycle, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	past := sortableCycle{}
	for _, cycle := range j.Cycles {
		if cycle.End != nil {
			past = append(past, cycle)
		}
	}

	sort.Sort(past)
	filtered := []*common.Cycle{}
	idx := start
	for i := 0; i < end && i+idx < len(past); i++ {
		f := past[idx+i]
		f.Watched = []*common.Movie{}

		fmt.Printf("[GetPastCycles] finding watched movies for cycle %d\n", f.Id)
		for _, movie := range j.Movies {
			if movie.CycleWatchedId == f.Id {
				fmt.Printf("found movie with ID %d\n", movie.Id)
				f.Watched = append(f.Watched, j.movieFromJson(movie))
			}
		}

		filtered = append(filtered, f)
	}

	return filtered, nil
}

func (j *jsonConnector) movieFromJson(jMovie jsonMovie) *common.Movie {
	movie := &common.Movie{
		Id:          jMovie.Id,
		Name:        jMovie.Name,
		Description: jMovie.Description,
		Removed:     jMovie.Removed,
		Approved:    jMovie.Approved,
		//CycleAdded:   j.findCycle(jMovie.CycleAddedId),
		//CycleWatched: j.findCycle(jMovie.CycleWatchedId),
		Links:  jMovie.Links,
		Poster: jMovie.Poster,
	}

	return movie
}

func (j *jsonConnector) GetMoviesFromCycle(id int) ([]*common.Movie, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	watched := j.findCycle(id)
	if watched == nil {
		return nil, fmt.Errorf("Cycle with ID %d not found", id)
	}

	movies := []*common.Movie{}
	for _, movie := range j.Movies {
		if movie.CycleWatchedId == id {
			m := j.movieFromJson(movie)

			m.CycleWatched = watched
			m.CycleAdded = j.findCycle(movie.CycleAddedId)

			movies = append(movies, j.movieFromJson(movie))
		}
	}

	return movies, nil
}

// UserLogin returns a user if the given username and password match a user.
func (j *jsonConnector) UserLogin(name, hashedPw string) (*common.User, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	name = strings.ToLower(name)
	for _, user := range j.Users {
		if strings.ToLower(user.Name) == name {
			if hashedPw == user.Password {
				return user, nil
			}
			fmt.Printf("Bad password for user %s\n", name)
			return nil, fmt.Errorf("Invalid login credentials")
		}
	}
	fmt.Printf("User with name %s not found\n", name)
	return nil, fmt.Errorf("Invalid login credentials")
}

// Get the total number of users
func (j *jsonConnector) GetUserCount() int {
	j.lock.RLock()
	defer j.lock.RUnlock()

	return len(j.Users)
}

func (j *jsonConnector) GetUsers(start, count int) ([]*common.User, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	uids := []int{}
	for _, u := range j.Users {
		uids = append(uids, u.Id)
	}

	sort.Ints(uids)

	ulist := []*common.User{}
	for i := 0; i < len(uids) && len(ulist) <= count; i++ {
		id := uids[i]
		if id < start {
			continue
		}

		u := j.findUser(id)
		if u != nil {
			ulist = append(ulist, u)
		}
	}

	return ulist, nil
}

func (j *jsonConnector) GetUser(userId int) (*common.User, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	u := j.findUser(userId)
	if u == nil {
		return nil, fmt.Errorf("User not found with ID %d", userId)
	}
	return u, nil
}

func (j *jsonConnector) GetUserVotes(userId int) ([]*common.Movie, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	votes := []*common.Movie{}
	for _, v := range j.Votes {
		if v.UserId == userId {
			mov := j.findMovie(v.MovieId)
			if mov != nil {
				votes = append(votes, mov)
			}
		}
	}

	return votes, nil
}

func (j *jsonConnector) DecayVotes(age int) error {
	sortable := sortableCycle(j.Cycles)
	sort.Sort(sortable)

	idLimit := 0
	for i, cycle := range sortable {
		if i >= age {
			idLimit = cycle.Id
			break
		}
	}

	active, err := j.GetActiveMovies()
	if err != nil {
		return fmt.Errorf("Error getting active movies: %v", err)
	}

	for _, movie := range active {
		for _, vote := range movie.Votes {
			if vote.CycleAdded.Id < idLimit {
				err = j.DeleteVote(vote.User.Id, vote.Movie.Id)
				if err != nil {
					return fmt.Errorf("Error deleting vote by user ID %d for movie ID %d: %v", vote.User.Id, vote.Movie.Id, err)
				}
			}
		}
	}

	return nil
}

func (j *jsonConnector) nextUserId() int {
	highest := 0
	for _, u := range j.Users {
		if u.Id > highest {
			highest = u.Id
		}
	}
	return highest + 1
}

func (j *jsonConnector) AddUser(user *common.User) (int, error) {
	j.lock.Lock()
	defer j.lock.Unlock()

	name := strings.ToLower(user.Name)
	for _, u := range j.Users {
		if u.Id == user.Id {
			return 0, fmt.Errorf("User already exists with ID %d", user.Id)
		}
		if strings.ToLower(u.Name) == name {
			return 0, fmt.Errorf("User already exists with name %s", user.Name)
		}
	}

	user.Id = j.nextUserId()

	j.Users = append(j.Users, user)
	return user.Id, j.save()
}

func (j *jsonConnector) AddVote(userId, movieId int) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	user := j.findUser(userId)
	if user == nil {
		return fmt.Errorf("User not found with ID %d", userId)
	}

	movie := j.findMovie(movieId)
	if movie == nil {
		return fmt.Errorf("Movie not found with ID %d", movieId)
	}

	if movie.CycleWatched != nil {
		return fmt.Errorf("Movie has already been watched")
	}

	if movie.Removed {
		return fmt.Errorf("Movie has been removed by a mod or admin")
	}

	cc := j.currentCycle()
	if cc == nil {
		return fmt.Errorf("No cycle currently active")
	}

	j.Votes = append(j.Votes, jsonVote{userId, movieId, cc.Id})
	return j.save()
}

func (j *jsonConnector) requireApproval() bool {
	// ignore errors here.  "false" is default.
	val, _ := j.GetCfgBool("RequireApproval", false)
	return val
}

func (j *jsonConnector) DeleteVote(userId, movieId int) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	cc := j.currentCycle()
	if cc == nil {
		return fmt.Errorf("No cycle active")
	}

	found := false
	newVotes := []jsonVote{}
	for _, v := range j.Votes {
		if v.UserId == userId && v.MovieId == movieId && v.CycleId == cc.Id {
			found = true
		} else {
			newVotes = append(newVotes, v)
		}
	}

	if !found {
		return fmt.Errorf("Vote not found for current cycle")
	}
	j.Votes = newVotes
	return j.save()
}

func (j *jsonConnector) CheckMovieExists(title string) (bool, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	clean := common.CleanMovieName(title)
	for _, m := range j.Movies {
		if clean == common.CleanMovieName(m.Name) {
			return true, nil
		}
	}
	return false, nil
}

func (j *jsonConnector) CheckUserExists(name string) (bool, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	lc := strings.ToLower(name)
	for _, user := range j.Users {
		if lc == strings.ToLower(user.Name) {
			return true, nil
		}
	}
	return false, nil
}

/* Find */

func (j *jsonConnector) findMovie(id int) *common.Movie {
	if id == 0 {
		return nil
	}

	for _, m := range j.Movies {
		if m.Id == id {
			movie := j.movieFromJson(m)
			movie.CycleWatched = j.findCycle(m.CycleWatchedId)
			movie.CycleAdded = j.findCycle(m.CycleAddedId)
			//fmt.Printf("[findMovie] added:%s watched:%s\n", watched, added)
			return movie
		}
	}

	fmt.Printf("findMovie() not found with ID %d\n", id)
	return nil
}

func (j *jsonConnector) findCycle(id int) *common.Cycle {
	if id == 0 {
		return nil
	}

	for _, c := range j.Cycles {
		if c.Id == id {
			return &common.Cycle{
				Id:    c.Id,
				Start: c.Start,
				End:   c.End,
			}
		}
	}
	return nil
}

func (j *jsonConnector) findVotes(movie *common.Movie) []*common.Vote {
	votes := []*common.Vote{}
	for _, v := range j.Votes {
		if v.MovieId == movie.Id {
			votes = append(votes, &common.Vote{
				Movie:      movie,
				CycleAdded: j.findCycle(v.CycleId),
				User:       j.findUser(v.UserId),
			})
		}
	}

	return votes
}

func (j *jsonConnector) findUser(id int) *common.User {
	for _, u := range j.Users {
		if u.Id == id {
			return u
		}
	}
	return nil
}

/* Update */

func (j *jsonConnector) UpdateUser(user *common.User) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	newLst := []*common.User{}
	for _, u := range j.Users {
		if u.Id == user.Id {
			newLst = append(newLst, user)
		} else {
			newLst = append(newLst, u)
		}
	}
	return j.save()
}

func (j *jsonConnector) UpdateMovie(movie *common.Movie) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	newLst := []jsonMovie{}
	for _, m := range j.Movies {
		if m.Id == movie.Id {
			newM := j.newJsonMovie(movie)
			newM.Id = m.Id
			newLst = append(newLst, newM)
		} else {
			newLst = append(newLst, m)
		}
	}
	j.Movies = newLst

	return j.save()
}

func (j *jsonConnector) UpdateCycle(cycle *common.Cycle) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	// clear these out
	cycle.Watched = nil

	newLst := []*common.Cycle{}
	for _, c := range j.Cycles {
		if c.Id == cycle.Id {
			newLst = append(newLst, cycle)
		} else {
			newLst = append(newLst, c)
		}
	}
	return j.save()
}

func (j *jsonConnector) UserVotedForMovie(userId, movieId int) (bool, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	cc := j.currentCycle()
	if cc == nil {
		return false, fmt.Errorf("No cycle active")
	}

	for _, v := range j.Votes {
		if v.MovieId == movieId && v.UserId == userId && v.CycleId == cc.Id {
			return true, nil
		}
	}

	return false, nil
}

// Configuration stuff
type cfgValType int

const (
	CVT_STRING cfgValType = iota
	CVT_INT
	CVT_BOOL
)

type configValue struct {
	Type  cfgValType
	Value interface{}
}

func (v configValue) String() string {
	t := ""
	switch v.Type {
	case CVT_STRING:
		t = "string"
		break
	case CVT_INT:
		t = "int"
		break
	case CVT_BOOL:
		t = "bool"
		break
	}

	return fmt.Sprintf("configValue{Type:%s Value:%v}", t, v.Value)
}

func (j *jsonConnector) GetCfgString(key, value string) (string, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	val, ok := j.Settings[key]
	if !ok {
		return value, nil
		//return "", fmt.Errorf("Setting with key %q does not exist", key)
	}

	switch val.Type {
	case CVT_STRING:
		return val.Value.(string), nil
	case CVT_INT:
		return fmt.Sprintf("%d", val.Value.(int)), nil
	case CVT_BOOL:
		return fmt.Sprintf("%t", val.Value.(bool)), nil
	default:
		return "", fmt.Errorf("Unknown type %d", val.Type)
	}
}

func (j *jsonConnector) GetCfgInt(key string, value int) (int, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	val, ok := j.Settings[key]
	if !ok {
		return value, nil
		//return 0, fmt.Errorf("Setting with key %q does not exist", key)
	}

	switch val.Type {
	case CVT_STRING:
		ival, err := strconv.ParseInt(val.Value.(string), 10, 32)
		if err != nil {
			return 0, fmt.Errorf("Int parse error: %s", err)
		}

		return int(ival), nil
	case CVT_INT:
		if val, ok := val.Value.(int); ok {
			return val, nil
		}
		if val, ok := val.Value.(float64); ok {
			return int(val), nil
		}
		return 0, fmt.Errorf("Unknown number type for %s", key)
	case CVT_BOOL:
		if val.Value.(bool) == true {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("Unknown type %d", val.Type)
	}
}

func (j *jsonConnector) GetCfgBool(key string, value bool) (bool, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	val, ok := j.Settings[key]
	if !ok {
		return value, nil
		//return false, fmt.Errorf("Setting with key %q does not exist", key)
	}

	switch val.Type {
	case CVT_STRING:
		bval, err := strconv.ParseBool(val.Value.(string))
		if err != nil {
			return false, fmt.Errorf("Bool parse error: %s", err)
		}
		return bval, nil
	case CVT_INT:
		if v, ok := val.Value.(int); ok && v == 0 {
			return false, nil
		}
		return true, nil
	case CVT_BOOL:
		v, ok := val.Value.(bool)
		return (ok && v), nil
	default:
		return false, fmt.Errorf("Unknown type %d", val.Type)
	}
}

func (j *jsonConnector) SetCfgString(key, value string) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	j.Settings[key] = configValue{CVT_STRING, value}

	return j.save()
}

func (j *jsonConnector) SetCfgInt(key string, value int) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	j.Settings[key] = configValue{CVT_INT, value}

	return j.save()
}

func (j *jsonConnector) SetCfgBool(key string, value bool) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	j.Settings[key] = configValue{CVT_BOOL, value}

	return j.save()
}

func (j *jsonConnector) DeleteCfgKey(key string) error {
	j.lock.Lock()
	defer j.lock.Unlock()

	delete(j.Settings, key)

	return j.save()
}

func (j *jsonConnector) DeleteUser(userId int) error {
	found := false
	j.lock.Lock()
	defer j.lock.Unlock()

	newUsers := []*common.User{}

	for _, user := range j.Users {
		if user.Id == userId {
			found = true
		} else {
			newUsers = append(newUsers, user)
		}
	}

	if !found {
		return fmt.Errorf("User with ID %d does not exist", userId)
	}

	j.Users = newUsers
	return j.save()
}

func (j *jsonConnector) SearchMovieTitles(query string) ([]*common.Movie, error) {
	j.lock.RLock()
	defer j.lock.RUnlock()

	found := []*common.Movie{}
	query = strings.ToLower(query)
	words := strings.Split(query, " ")

	for _, movie := range j.Movies {
		ok := true
		for _, word := range words {
			if !strings.Contains(movie.Name, word) {
				ok = false
				break
			}
		}

		if ok {
			m := j.findMovie(movie.Id)
			m.Votes = j.findVotes(m)
			found = append(found, m)
		}
	}

	return found, nil
}

func (j *jsonConnector) DeleteCycle(cycleId int) error {
	return fmt.Errorf("DeleteCycle() not implemented for JSON")
}

func (j *jsonConnector) DeleteMovie(movieId int) error {
	return fmt.Errorf("DeleteMovie() not implemented for JSON")
}

func (j *jsonConnector) Test_GetUserVotes(userId int) ([]*common.Vote, error) {
	votes := []*common.Vote{}
	for _, vote := range j.Votes {
		if vote.UserId != userId {
			continue
		}
		u := j.findUser(vote.UserId)
		m := j.findMovie(vote.MovieId)
		c := j.findCycle(vote.CycleId)

		//fmt.Printf("Test_GetUserVotes() movie: %s\n", m)
		votes = append(votes, &common.Vote{CycleAdded: c, Movie: m, User: u})
	}
	return votes, nil
}
