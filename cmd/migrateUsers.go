package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/mitchellh/mapstructure"
	mpc "github.com/zorchenhimer/MoviePolls/common"
	mpd "github.com/zorchenhimer/MoviePolls/data"
)

const jsonFilename string = "db/data.json"

func main() {
	err := loadOldDB()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}

type OldUser struct {
	Id                  int
	Name                string
	Password            string
	OAuthToken          string
	Email               string
	NotifyCycleEnd      bool
	NotifyVoteSelection bool
	Privilege           int
	PassDate            time.Time
	RateLimitOverride   bool
	LastMovieAdd        time.Time
}

func loadOldDB() error {

	data, err := ioutil.ReadFile(jsonFilename)
	if err != nil {
		fmt.Println(err)
	}

	var fullData map[string]interface{}

	json.Unmarshal(data, &fullData)

	// fullData now contains the whole db
	jUsers := fullData["Users"].(map[string]interface{})
	oldUsers := []OldUser{}
	for _, user := range jUsers {
		ou := &OldUser{}
		mapstructure.Decode(user, &ou)
		// lets parse the fucking time by hand
		if user.(map[string]interface{})["PassDate"] == nil {
			fmt.Println("[ERROR] Could not parse field 'PassDate', was the database already converted?")
			return nil
		}
		ou.PassDate, err = time.Parse(time.RFC3339, user.(map[string]interface{})["PassDate"].(string))
		if err != nil {
			fmt.Println("[ERROR] Could not parse field 'PassDate', was the database already converted?")
			return nil
		}
		oldUsers = append(oldUsers, *ou)
	}

	delete(fullData, "Users")

	data, err = json.MarshalIndent(fullData, "", " ")

	// write a temporary file which contains the db without the "users"
	err = ioutil.WriteFile("db/temp.json", data, 0666)
	if err != nil {
		return err
	}

	l, err := mpc.NewLogger(mpc.LLDebug, "")

	if err != nil {
		return err
	}

	dc, err := mpd.GetDataConnector("json", "db/temp.json", l)

	if err != nil {
		return err
	}

	for _, oldUser := range oldUsers {

		// convert old movies to new ones
		newUser := mpc.User{
			Id:                  oldUser.Id,
			Name:                oldUser.Name,
			Email:               oldUser.Email,
			NotifyCycleEnd:      oldUser.NotifyCycleEnd,
			NotifyVoteSelection: oldUser.NotifyVoteSelection,
			Privilege:           mpc.PrivilegeLevel(oldUser.Privilege),
		}

		// Do NOT build new entries for users with empty passwords (i.e. Deleted users)
		if oldUser.Password != "" {
			authMethod := &mpc.AuthMethod{
				Type:     mpc.AUTH_LOCAL,
				Password: oldUser.Password,
				PassDate: oldUser.PassDate,
			}

			newUser.AuthMethods = []*mpc.AuthMethod{}

			id, err := dc.AddAuthMethod(authMethod)

			if err != nil {
				return err
			}

			authMethod.Id = id

			newUser.AuthMethods = append(newUser.AuthMethods, authMethod)
		}
		dc.UpdateUser(&newUser)
	}

	err = os.Rename("db/temp.json", jsonFilename)
	if err != nil {
		return err
	}

	return nil
}
