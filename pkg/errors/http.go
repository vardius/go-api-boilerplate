// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package errors implements functions to manipulate errors.
package errors

import (
	"net/http"
	"strconv"
)

var errorCodeToHTTPStatusMap = map[string]int{
	// 4xx codes
	INVALID:      http.StatusBadRequest,
	UNAUTHORIZED: http.StatusUnauthorized,
	FORBIDDEN:    http.StatusForbidden,
	NOTFOUND:     http.StatusNotFound,
	TIMEOUT:      http.StatusRequestTimeout,

	// 5xx codes
	INTERNAL:          http.StatusInternalServerError,
	TEMPORARYDISABLED: http.StatusServiceUnavailable,
}

// HTTPStatusCode returns the http status code based on error code if available.
// It returns StatusOK if the error is nil.
func HTTPStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	} else if e, ok := err.(*appError); ok && e.Code != "" {
		statusCode, isMapped := errorCodeToHTTPStatusMap[e.Code]
		if isMapped {
			return statusCode
		}

		codeAsInt, err := strconv.Atoi(e.Code)
		if err != nil {
			return http.StatusInternalServerError
		}

		if http.StatusText(codeAsInt) != "" {
			return codeAsInt
		}
	} else if ok && e.Err != nil {
		return HTTPStatusCode(e.Err)
	}

	return http.StatusInternalServerError
}
