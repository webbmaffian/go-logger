package logerror

type Error string

func (err Error) Error() string {
	return string(err)
}
