package domain

import (
	"net"

	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

type EventMetadata struct {
	Identity  *identity.Identity `json:"identity,omitempty"`
	IPAddress net.IP             `json:"ip_address,omitempty"`
	UserAgent string             `json:"http_user_agent,omitempty"`
	Referer   string             `json:"http_referer,omitempty"`
}

func (m *EventMetadata) IsEmpty() bool {
	return m.IPAddress == nil && m.Identity == nil && m.UserAgent == "" && m.Referer == ""
}
