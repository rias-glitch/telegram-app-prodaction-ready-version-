package telegram

import (
	"encoding/json"
	"net/url"
)

type WebAppUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
}

func ParseUser(initData string) (*WebAppUser, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, err
	}

	var user WebAppUser
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil {
		return nil, err
	}

	return &user, nil
}
