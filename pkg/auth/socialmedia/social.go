package socialmedia

import (
	"encoding/json"
	"github.com/vardius/go-api-boilerplate/pkg/auth/identity"
	"io/ioutil"
	"net/http"
	"net/url"
)

type responsePayload struct {
	AuthToken string             `json:"authToken"`
	Identity  *identity.Identity `json:"identity"`
}

type commandPayload struct {
	AuthToken string          `json:"authToken"`
	Data      json.RawMessage `json:"data"`
}

func (p *commandPayload) toJSON() json.RawMessage {
	b, err := json.Marshal(p)
	if err != nil {
		return nil
	}

	return b
}

func getProfile(accessToken, apiUrl string) ([]byte, error) {
	resp, e := http.Get(apiUrl + "?access_token=" + url.QueryEscape(accessToken))
	if e != nil {
		return nil, e
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
