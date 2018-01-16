package picard

// ModelNotFoundError is returned when functions that expect to return an
// existing model cannot find one
const ModelNotFoundError Error = "Model Not Found"

// Error is a type of error that picard will return
type Error string

func (err Error) Error() string {
	return string(err)
}
