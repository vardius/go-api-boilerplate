package gorouter

import "strings"

func splitPath(path string) (parts []string) {
	for {
		if i := strings.IndexByte(path, '{'); i >= 0 {
			if part := trimPath(path[:i]); part != "" {
				parts = append(parts, part)
			}
			if j := strings.IndexByte(path, '}') + 1; j > 0 {
				if part := trimPath(path[i:j]); part != "" {
					parts = append(parts, part)
				}
				i = j
			} else {
				continue
			}
			path = path[i:]
		} else {
			break
		}
	}

	if len(path) != 0 && path != "/" {
		if part := trimPath(path); part != "" {
			parts = append(parts, part)
		}
	}

	return
}

func trimPath(path string) string {
	pathLen := len(path)
	if pathLen > 0 && path[0] == '/' {
		path = path[1:]
		pathLen--
	}

	if pathLen > 0 && path[pathLen-1] == '/' {
		path = path[:pathLen-1]
		pathLen--
	}

	return path
}
