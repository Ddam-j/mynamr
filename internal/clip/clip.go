package clip

type Interface interface {
	ReadText() (string, error)
	WriteText(text string) error
}
