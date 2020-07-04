package email

import (
	"fmt"
	"html/template"
)

var (
	Login *template.Template
)

func init() {
	var err error

	loginTemplate := template.New("login.html")
	Login, err = loginTemplate.Parse(loginHTML)
	if err != nil {
		panic(fmt.Sprintf("Could not load email login template: %s", err.Error()))
	}
}
