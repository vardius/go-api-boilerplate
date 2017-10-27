package err

// HTTPError allows you yo return nice error responses
type HTTPError struct {
	Code    int
	Error   error  `json:"error"`
	Message string `json:"message"`
}
