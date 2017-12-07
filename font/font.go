package font

import (
	"image"
	"image/draw"
	"unicode"

	"github.com/golang/freetype/truetype"
	gofont "golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"
)

type Font struct {
	gofont.Face
	hexDx   int
	data    []byte
	size    int
	ascent  int
	descent int
	stride  int
	letting int
	dy      int

	cache    Cache
	hexCache Cache
	decCache Cache
	imgCache map[signature]*image.RGBA
}

type rgba struct {
	r, g, b, a uint32
}
type signature struct {
	b  byte
	dy int
	rgba
}

func NewGoRegular(size int) *Font {
	return NewTTF(goregular.TTF, size)
}

func NewGoMedium(size int) *Font {
	return NewTTF(gomedium.TTF, size)
}

func NewGoMonoBold(size int) *Font {
	return NewTTF(gomonobold.TTF, size)
}

func NewGoMono(size int) *Font {
	return NewTTF(gomono.TTF, size)
}

// NewBasic always returns a 7x13 basic font
func NewBasic(size int) *Font {
	f := basicfont.Face7x13
	size = 13
	ft := &Font{
		Face:     f,
		size:     size,
		ascent:   2,
		descent:  1,
		letting:  0,
		stride:   0,
		imgCache: make(map[signature]*image.RGBA),
	}
	ft.dy = ft.ascent + ft.descent + ft.size
	hexFt := fromTTF(gomono.TTF, ft.Dy()/4+3)
	ft.hexDx = ft.genChar('_').Bounds().Dx()
	helper := mkhelper(hexFt)
	for i := 0; i != 256; i++ {
		ft.cache[i] = ft.genChar(byte(i))
		if ft.cache[i] == nil {
			ft.cache[i] = hexFt.genHexCharTest(ft.Dy(), byte(i), helper)
		}
	}
	return ft
}

const hexbytes = "0123456789abcdef"

func mkhelper(hexFt *Font) []*Glyph {
	helper := make([]*Glyph, 0, len(hexbytes))
	for _, s := range hexbytes {
		helper = append(helper, hexFt.genChar(byte(s)))
	}
	return helper
}

func NewTTF(data []byte, size int) *Font {
	//return makefont(data, size)
	ft := fromTTF(data, size)
	hexFt := fromTTF(gomono.TTF, ft.Dy()/3+3)
	//hexFt := fromTTF(gomono.TTF, ft.Dy()/4+3)
	ft.hexDx = ft.genChar('_').Bounds().Dx()

	helper := mkhelper(hexFt)
	for i := 0; i != 256; i++ {
		ft.cache[i] = ft.genChar(byte(i))
		if ft.cache[i] == nil {
			ft.cache[i] = hexFt.genHexChar(ft.Dy(), byte(i), helper)
		}
	}
	return ft
}
func fromTTF(data []byte, size int) *Font {
	f, err := truetype.Parse(data)
	if err != nil {
		panic(err)
	}
	ft := &Font{
		Face: truetype.NewFace(f,
			&truetype.Options{
				Size: float64(size),
			}),
		size:     size,
		ascent:   2,
		descent:  +(size / 3),
		stride:   0,
		data:     data,
		imgCache: make(map[signature]*image.RGBA),
	}
	ft.dy = ft.ascent + ft.descent + ft.size
	return ft
}

// Clone returns a copy of the font
func Clone(f *Font, dy int) *Font {
	// TODO(as): letting, etc
	f2 := clone(f, dy)
	f2.SetLetting(f.Letting())
	f2.SetStride(f.Stride())
	return f2
}
func clone(f *Font, dy int) *Font {
	if f.data == nil {
		return NewBasic(dy)
	}
	return NewTTF(f.data, dy)
}

func (f *Font) NewSize(dy int) *Font {
	if dy == f.Dy() {
		return f
	}
	return Clone(f, dy)
}

func (f *Font) Ascent() int  { return f.ascent }
func (f *Font) Descent() int { return f.descent }
func (f *Font) Stride() int  { return f.stride }
func (f *Font) Letting() int { return f.letting }

func (f *Font) SetAscent(px int) {
	f.ascent = px
	f.dy = f.ascent + f.descent + f.size
}
func (f *Font) SetDescent(px int) {
	f.descent = px
	f.dy = f.ascent + f.descent + f.size
}
func (f *Font) SetStride(px int) {
	f.stride = px
}
func (f *Font) SetLetting(px int) {
	f.letting = px
}

func (f *Font) genChar(b byte) *Glyph {
	if !f.Printable(b) {
		return nil
	}
	dr, mask, maskp, adv, _ := f.Face.Glyph(fixed.P(0, f.size), rune(b))
	r := image.Rect(0, 0, Fix(adv), f.Dy())
	m := image.NewAlpha(r)
	r = r.Add(image.Pt(dr.Min.X, dr.Min.Y))
	draw.Draw(m, r, mask, maskp, draw.Src)
	return &Glyph{mask: m, Rectangle: m.Bounds()}
}
func (f *Font) genHexCharTest(dy int, b byte, helper []*Glyph) *Glyph {
	g0 := helper[b/16]
	g1 := helper[b%16]
	r := image.Rect(2, f.descent+f.ascent, g0.Bounds().Dx()+g1.Bounds().Dx()+6, dy)
	m := image.NewAlpha(r)
	draw.Draw(m, r, g0.Mask(), image.ZP, draw.Over)
	r.Min.X += g0.Mask().Bounds().Dx()
	draw.Draw(m, r.Add(image.Pt(-f.descent/4, f.descent*2)), g1.Mask(), image.ZP, draw.Over)
	return &Glyph{mask: m, Rectangle: m.Bounds()}
}

func (f *Font) genHexChar(dy int, b byte, helper []*Glyph) *Glyph {
	g0 := helper[b/16]
	g1 := helper[b%16]
	r := image.Rect(2, f.descent+f.ascent-3, g0.Bounds().Dx()+g1.Bounds().Dx()+7, dy)
	m := image.NewAlpha(r)
	draw.Draw(m, r, g0.Mask(), image.ZP, draw.Over)
	r.Min.X += g0.Mask().Bounds().Dx()
	draw.Draw(m, r.Add(image.Pt(-f.descent/4, f.descent+f.descent/2)), g1.Mask(), image.ZP, draw.Over)
	return &Glyph{mask: m, Rectangle: m.Bounds()}
}

func (f *Font) Char(b byte) (mask *image.Alpha) {
	return f.cache[b].mask
}

func (f *Font) Dx(s string) int {
	return f.MeasureBytes([]byte(s))
}
func (f *Font) Dy() int {
	return f.dy + f.letting
}
func (f *Font) Size() int {
	return f.size
}
func Fix(i fixed.Int26_6) int {
	return i.Round()
}
func (f *Font) MeasureBytes(p []byte) (w int) {
	for i := range p {
		w += f.Measure(rune(byte(p[i])))
	}
	return w
}

func (f *Font) MeasureByte(b byte) (n int) {
	return f.cache[b].Dx() + f.stride
}

func (f *Font) MeasureRune(r rune) (q int) {
	advance, _ := f.Face.GlyphAdvance(r)
	return Fix(advance)
}

func (f *Font) Measure(r rune) (q int) {
	return f.cache[byte(r)].Dx() + f.stride
}

func (f *Font) MeasureHex() int {
	return f.hexDx
}

func (f *Font) TTF() []byte {
	return f.data
}

func (f *Font) Printable(b byte) bool {
	if b == 0 || b > 127 {
		return false
	}
	if unicode.IsGraphic(rune(b)) {
		return true
	}
	return false
}
