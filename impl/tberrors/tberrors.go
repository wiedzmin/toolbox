package tberrors

type ErrInvalidUrl struct {
	Content string
}

func (e ErrInvalidUrl) Error() string {
	return "invalid url found"
}
