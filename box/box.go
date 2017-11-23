package box

const SLOP = 25

type Box struct {
	Width    int
	Minwidth int
	Nrune    int
	Ptr      []byte
}

func (b *Box) Break() byte {
	n := b.Len()
	if n == 0 {
		return 0
	}
	return b.Ptr[0]
}

func (b *Box) Len() int {
	if b.Nrune < 0 {
		return 1
	}
	return b.Nrune
}

func (b *Box) Bytes() []byte {
	n := b.Len()
	if n <= 0 {
		return nil
	}
	return b.Ptr[:n]
}
