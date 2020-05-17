package data

import (
	"fmt"
	"os"
	"testing"

	"github.com/zorchenhimer/MoviePolls/common"
)

/*
	Helper functions used in tests
*/

var (
	err error
)

var testConnectors = map[string]func() (TestableDataConnector, error) {
	"mysql": func() (TestableDataConnector, error) {
		dc, err := newMySqlConnector("root:buttslol@tcp(127.0.0.1:3306)/moviepolls?parseTime=true&loc=Local")
		return TestableDataConnector(dc), err
	},
	"json": func() (TestableDataConnector, error) {
		dc, err := newJsonConnector("test.json")
		return TestableDataConnector(dc), err
	},
}

func TestMain(m *testing.M) {
	failval := 0
	for name, connector := range testConnectors {
		fmt.Println("Running "+name+" tests")
		if name == "json" {
			os.Remove("test.json")
		}

		conn, err = connector()
		if err != nil {
			fmt.Println(err)
			failval = 1
			continue
		}

		retval := m.Run()
		if retval != 0 {
			failval = retval
		}
	}

	os.Exit(failval)
}

func compareSlices(t *testing.T, SliceA, SliceB []string) error {
	t.Helper()

	// Verify all of A is in B
	for _, A := range SliceA {
		found := false
		for _, B := range SliceB {
			if A == B {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("Missing value in second slice: %q", A)
		}
	}

	// Verify all of B is in A
	for _, B := range SliceB {
		found := false
		for _, A := range SliceA {
			if B == A {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("Extra value in second slice: %q", B)
		}
	}

	return nil
}

func compareUsers(a, b *common.User, t *testing.T) {
	t.Helper()

	if a.Id != b.Id {
		t.Fatalf("Id mismatch: %d vs %d", a.Id, b.Id)
	}

	if a.Name != b.Name {
		t.Fatalf("Username mismatch: %q vs %q", a.Name, b.Name)
	}

	// Token stuff isn't implemented yet.
	//if a.OAuthToken != b.OAuthToken {
	//	t.Fatalf("Token mismatch: %q vs %q", a.OAuthToken, b.OAuthToken)
	//}

	if a.Email != b.Email {
		t.Fatalf("Email mismatch: %q vs %q", a.Email, b.Email)
	}

	if a.NotifyCycleEnd != b.NotifyCycleEnd {
		t.Fatalf("NotifyCycleEnd mismatch: %t vs %t", a.NotifyCycleEnd, b.NotifyCycleEnd)
	}

	if a.NotifyVoteSelection != b.NotifyVoteSelection {
		t.Fatalf("NotifyVoteSelection mismatch: %t vs %t", a.NotifyVoteSelection, b.NotifyVoteSelection)
	}

	if a.PassDate != b.PassDate {
		t.Fatalf("PassDate mismatch: %s vs %s", a.PassDate, b.PassDate)
	}

	//if user.RateLimitOverride != loggedIn.RateLimitOverride {
	//	t.Fatalf("RateLimitOverride mismatch: %t vs %t", user.RateLimitOverride, loggedIn.RateLimitOverride)
	//}

	//if user.LastMovieAdd != loggedIn.LastMovieAdd {
	//	t.Fatalf("LastMovieAdd mismatch: %s vs %s", user.LastMovieAdd, loggedIn.LastMovieAdd)
	//}
}

func compareMovies(a, b *common.Movie, t *testing.T) {
	t.Helper()

	if a.Name != b.Name {
		t.Fatalf("Name mismatch: %q vs %q", a.Name, b.Name)
	}

	if len(a.Links) != len(b.Links) {
		t.Fatalf("Links list length mismatch: %d vs %d", len(a.Links), len(b.Links))
	}

	err = compareSlices(t, a.Links, b.Links)
	if err != nil {
		t.Fatal(err)
	}

	if a.Description != b.Description {
		t.Fatalf("Description mismatch: %q vs %q", a.Description, b.Description)
	}

	if a.Removed != b.Removed {
		t.Fatalf("Removed mismatch: %t vs %t", a.Removed, b.Removed)
	}

	if a.Approved != b.Approved {
		t.Fatalf("Approved mismatch: %t vs %t", a.Approved, b.Approved)
	}

	//if a.Watched != b.Watched {
	//	t.Fatalf("Watched mismatch: %s vs %s", a.Watched, b.Watched)
	//}

	if a.Poster != b.Poster {
		t.Fatalf("Poster mismatch: %q vs %q", a.Poster, b.Poster)
	}

	compareCycles(a.CycleAdded, b.CycleAdded, t)
}

func compareCycles(a, b *common.Cycle, t *testing.T) {
	t.Helper()

	if a.Id != b.Id {
		t.Fatalf("Cycle Id mismatch: %d vs %d", a.Id, b.Id)
	}

	if a.Start != b.Start {
		t.Fatalf("Cycle start mismatch: %s vs %s", a.Start, b.Start)
	}

	if a.End != b.End {
		t.Fatalf("Cycle end mismatch: %s vs %s", a.End, b.End)
	}
}
