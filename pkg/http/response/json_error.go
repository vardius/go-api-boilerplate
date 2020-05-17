/*
Package response provides helpers and utils for working with HTTP response
*/
package response

import (
	"context"
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/container"
	httpErrors "github.com/vardius/go-api-boilerplate/pkg/http/errors"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	mtd "github.com/vardius/go-api-boilerplate/pkg/metadata"
)

// JSONError returns data as json response
// uses JSON internally
func JSONError(ctx context.Context, w http.ResponseWriter, err error) error {
	httpError := httpErrors.NewHttpError(err)

	w.WriteHeader(httpError.Code)
	if m, ok := mtd.FromContext(ctx); ok {
		httpError.RequestID = m.TraceID
	}

	if requestContainer, ok := container.FromContext(ctx); ok {
		if v, ok := requestContainer.Get("logger"); ok {
			if logger, ok := v.(*log.Logger); ok {
				logger.Debug(ctx, "[HTTP] Error: %s", err)
				if httpError.Code == http.StatusInternalServerError {
					logger.Error(ctx, "[HTTP] Error: %s", err)
				} else {
					logger.Debug(ctx, "[HTTP] Error: %s", err)
				}
			}
		}
	}

	if err := JSON(ctx, w, httpError); err != nil {
		return err
	}

	return nil
}

// MustJSONError returns data as json response
// will panic if unable to marshal error into JSON object
// uses JSONError internally
func MustJSONError(ctx context.Context, w http.ResponseWriter, err error) {
	if err := JSONError(ctx, w, err); err != nil {
		panic(err)
	}
}
