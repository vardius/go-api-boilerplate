package domain

import (
	"net"

	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

type EventMetadata struct {
	Identity  *identity.Identity `json:"identity,omitempty"`
	IPAddress net.IP             `json:"ip_address,omitempty"`
}

func (m *EventMetadata) IsEmpty() bool {
	return m.IPAddress == nil && m.Identity == nil
}
