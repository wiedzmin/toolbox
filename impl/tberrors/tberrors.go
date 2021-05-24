package tberrors

type ErrInvalidUrl struct {
	Content string
}

func (e ErrInvalidUrl) Error() string {
	return "invalid url found"
}

type ErrNotExist struct{}

func (e ErrNotExist) Error() string {
	return "file/dir does not exist"
}
