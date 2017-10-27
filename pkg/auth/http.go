package auth

import (
	"app/pkg/identity"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
)

type authResponse struct {
	AuthToken string             `json:"authToken"`
	Identity  *identity.Identity `json:"identity"`
}

type authCommandPayload struct {
	AuthToken string          `json:"authToken"`
	Data      json.RawMessage `json:"data"`
}

func (p *authCommandPayload) toJSON() json.RawMessage {
	b, err := json.Marshal(p)
	if err != nil {
		return nil
	}

	return b
}

func authCallback(accessToken, apiUrl string) ([]byte, error) {
	resp, e := http.Get(apiUrl + "?access_token=" + url.QueryEscape(accessToken))
	if e != nil {
		return nil, e
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
