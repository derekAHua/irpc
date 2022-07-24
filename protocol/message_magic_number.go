package protocol

const (
	magicNumber byte = 0x08
)

func MagicNumber() byte {
	return magicNumber
}
