package api

import "fmt"

// HealthHandler returns a 200 OK response body.
func HealthHandler() string {
	return fmt.Sprintf("ok")
}

// GreetHandler returns a greeting for the given name.
func GreetHandler(name string) string {
	return fmt.Sprintf("hello, %s", name)
}
