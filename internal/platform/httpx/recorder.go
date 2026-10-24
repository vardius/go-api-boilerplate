package httpx

import "net/http"

// recorder watches a response for Recover, Log and WriteError. It
// implements Unwrap, so http.ResponseController reaches the writer below,
// and Flush, because Connect's streaming handlers assert http.Flusher.
type recorder struct {
	http.ResponseWriter
	status      int
	bytes       int64
	wroteHeader bool
	// err is the error WriteError was given, for Log's line.
	err error
}

func (w *recorder) WriteHeader(code int) {
	// 1xx responses other than 101 are interim; the final status follows.
	if !w.wroteHeader && (code >= http.StatusOK || code == http.StatusSwitchingProtocols) {
		w.status, w.wroteHeader = code, true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *recorder) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.status, w.wroteHeader = http.StatusOK, true
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += int64(n)
	return n, err
}

func (w *recorder) Flush() {
	if !w.wroteHeader {
		w.status, w.wroteHeader = http.StatusOK, true
	}
	// A writer that can't flush sends the data when the handler returns.
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *recorder) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// findRecorder returns the innermost recorder wrapping w, if any.
func findRecorder(w http.ResponseWriter) *recorder {
	for {
		switch v := w.(type) {
		case *recorder:
			return v
		case interface{ Unwrap() http.ResponseWriter }:
			w = v.Unwrap()
		default:
			return nil
		}
	}
}
