/*
Package auth provides auth handlers for social media
*/
package auth

import (
	"io/ioutil"
	"net/http"
	"net/url"
)

type authTokenResponse struct {
	AuthToken string `json:"authToken"`
}

func getProfile(accessToken, apiURL string) ([]byte, error) {
	resp, e := http.Get(apiURL + "?access_token=" + url.QueryEscape(accessToken))
	if e != nil {
		return nil, e
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
