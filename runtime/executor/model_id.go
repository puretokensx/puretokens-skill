package main

import "regexp"

// IDs become individual profile filenames, never paths or shell expressions.
var modelIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+( [A-Za-z0-9_.-]+)*(\([A-Za-z0-9_.-]+\))?$`)

func validModelID(id string) bool {
	return len(id) <= 160 && id != "." && id != ".." && modelIDPattern.MatchString(id)
}
