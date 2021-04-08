package snowflake



import "fmt"

// SnowException .
type SnowException struct {
	message string
	error   error
}

// Exception .
func (e SnowException) Exception(message ...interface{}) {
	fmt.Println(message...)
}

// Error .
func (e SnowException) Error(err error) string {
	e.message = err.Error()
	e.error = err
	return e.message
}
