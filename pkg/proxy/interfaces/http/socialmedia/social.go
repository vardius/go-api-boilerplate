/*
Package socialmedia provides auth handlers for social media
*/
package socialmedia

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/vardius/go-api-boilerplate/pkg/common/application/security/identity"
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

func getProfile(accessToken, apiURL string) ([]byte, error) {
	resp, e := http.Get(apiURL + "?access_token=" + url.QueryEscape(accessToken))
	if e != nil {
		return nil, e
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
