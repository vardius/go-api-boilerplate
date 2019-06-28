package response

import (
	"github.com/vardius/golog"
)

var logger golog.Logger

// WithLogger registers logger
func WithLogger(l golog.Logger) {
	logger = l
}
