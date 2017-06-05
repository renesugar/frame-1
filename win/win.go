package win

/*
 */

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"sync"
	"time"

	"github.com/as/frame"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/mouse"
	// "golang.org/x/mobile/event/paint"
)

const (
	HiWater  = 1024 * 1024
	LoWater  = 2 * 1024
	MinWater = 1024
	MsgSize  = 64 * 1024
)

type Win struct {
	*frame.Frame
	Sp        image.Point // window offset
	size      image.Point // window size
	pad       image.Point // window text offset
	b         screen.Buffer
	scr       screen.Screen
	events    screen.Window
	Org       int64
	Qh        int64
	Q0, Q1    int64
	Nr        int64
	R         []byte
	Maxr      int64
	Mc        Mc
	Lastclick time.Time
	Selectq   int64
	Scrollr   image.Rectangle
	Sweeping     bool

	dirtysb bool
	sb      screen.Buffer
}

func (w *Win) Clicksb(pt image.Point, dir int) {
	n := w.Org
	fmt.Printf("p.Y, w.Bar() = %d, %s\n", pt.Y, w.bar())
	switch dir {
	case -1:
		rat := float64(w.bar().Max.Y) / float64(pt.Y)
		delta := int64(float64(w.Nchars) * rat)
		n -= delta

	case 0:
		dy := float64(pt.Y - w.bar().Min.Y)
		rat := float64(dy) / float64(w.bar().Dy())
		delta := int64(float64(w.Nchars) * rat)
		n += delta
	case 1:
		rat := float64(w.bar().Max.Y) / float64(pt.Y)
		delta := int64(float64(w.Nchars) * rat)
		n += delta
	}
	fmt.Printf("net org is %d\n", n)
	w.SetOrigin(n, true)
	w.drawsb()
}

func (w *Win) bar() image.Rectangle {
	r := w.Scrollr.Sub(w.Sp)
	dy := float64(r.Dy())
	rat0 := float64(w.Org) / float64(w.Nr)          // % scrolled
	rat1 := float64(w.Org+w.Nchars) / float64(w.Nr) // % covered by screen
	r.Min.Y = int(dy * rat0)
	r.Max.Y = int(dy * rat1)
	return r
}

func (w *Win) drawsb() {
	r := w.Scrollr.Sub(w.Sp)
	dy := float64(r.Dy())
	draw.Draw(w.b.RGBA(), r, X, image.ZP, draw.Src)
	rat0 := float64(w.Org) / float64(w.Nr)          // % scrolled
	rat1 := float64(w.Org+w.Nchars) / float64(w.Nr) // % covered by screen
	r.Min.Y = int(dy * rat0)
	r.Max.Y = int(dy * rat1)
	draw.Draw(w.b.RGBA(), r, LtGray, image.ZP, draw.Src)
	//drawBorder(sb.Buffer().RGBA(), w.Buffer().Bounds(), LtGray, image.ZP, 1)
	//drawBorder(sb.Buffer().RGBA(), Scrollr, LtGray, image.ZP, 1)
}

var (
	Red    = image.NewUniform(color.RGBA{255, 0, 0, 255})
	Green  = image.NewUniform(color.RGBA{255, 255, 192, 25})
	Blue   = image.NewUniform(color.RGBA{0, 192, 192, 255})
	Cyan   = image.NewUniform(color.RGBA{234, 255, 255, 255})
	White  = image.NewUniform(color.RGBA{255, 255, 255, 255})
	Yellow = image.NewUniform(color.RGBA{255, 255, 224, 255})
	X      = image.NewUniform(color.RGBA{255 - 32, 255 - 32, 224 - 32, 255})

	LtGray = image.NewUniform(color.RGBA{66 * 2, 66 * 2, 66*2 + 35, 255})
	Gray   = image.NewUniform(color.RGBA{66, 66, 66, 255})
	Mauve  = image.NewUniform(color.RGBA{0x99, 0x99, 0xDD, 255})
)

func (w *Win) Buffer() screen.Buffer {
	return w.b
}
func (w *Win) Size() image.Point {
	return w.size
}

const minSbWidth = 5

func New(scr screen.Screen, ft frame.Font, events screen.Window,
	sp, size, pad image.Point, cols frame.Color) *Win {
	b, err := scr.NewBuffer(size)
	if err != nil {
		panic(err)
	}
	r := image.Rectangle{pad, size}.Inset(1)
	w := &Win{
		Frame:  frame.New(r, ft, b.RGBA(), cols),
		b:      b,
		pad:    pad,
		scr:    scr,
		Sp:     sp,
		size:   size,
		events: events,
	}
	w.scrollinit(pad)
	w.Frame.Scroll = w.FrameScroll
	w.Blank()
	w.drawsb()
	return w
}

func (w *Win) scrollinit(pad image.Point) {
	sp := w.Sp
	w.Scrollr = image.ZR
	if pad.X > minSbWidth+3 {
		w.Scrollr = image.Rect(sp.X, sp.Y, sp.X+pad.X-3, w.size.Y)
	}
}

func (w *Win) Resize(size image.Point) {
	return
	b, err := w.scr.NewBuffer(size)
	if err != nil {
		panic(err)
	}
	w.size = size
	w.b = b
	r := image.Rectangle{w.pad, w.size}.Inset(1)
	w.Frame = frame.New(r, w.Frame.Font, w.b.RGBA(), w.Frame.Color)
	w.Frame.Scroll = w.FrameScroll
	w.Fill()
	w.Select(w.Q0, w.Q1)
	w.Mark()
	w.Refresh()
}

func (w *Win) SetFont(ft frame.Font) {
	P0, P1 := w.Frame.Dot()
	r := image.Rectangle{w.pad, w.size}.Inset(1)
	w.Frame = frame.New(r, ft, w.b.RGBA(), w.Frame.Color)
	w.Frame.Scroll = w.FrameScroll
	w.Fill()
	w.Frame.Select(P0, P1)
	w.Mark()
	w.Refresh()
}

func (w *Win) NextEvent() (e interface{}) {
	switch e := w.events.NextEvent().(type) {
	case mouse.Event:
		e.X -= float32(w.Sp.X)
		e.Y -= float32(w.Sp.Y)
		return e
	case interface{}:
		return e
	}
	return nil
}
func (w *Win) Send(e interface{}) {
	w.events.Send(e)
}
func (w *Win) SendFirst(e interface{}) {
	w.events.SendFirst(e)
}
func (w *Win) Blank() {
	buf := w.b
	if buf != nil {
		r := buf.RGBA().Bounds()
		draw.Draw(buf.RGBA(), r, w.Color.Back, image.ZP, draw.Src)
		if w.Sp.Y > 0 {
			r.Min.Y--
			//r.Max.Y--
		}
		//		drawBorder(buf.RGBA(), r, w.Color.Hi.Back, image.ZP, 1)
	}
}

type Mc struct {
	Buttons int
	Msec    int
	XY      image.Point
}

func (w *Win) Dot() (q0, q1 int64) {
	q0 = clamp(w.Q0, 0, w.Nr)
	q1 = clamp(w.Q1, 0, w.Nr)
	return
}

func (w *Win) FrameScroll(dl int) {
	if dl == 0 {
		//time.Sleep(15*time.Millisecond)
		return
	}
	q0 := int64(0)
	P0, P1 := w.Frame.Dot()
	if dl < 0 {
		q0 = w.BackNL(w.Org, -dl)
		if w.Sweeping {
			if w.Selectq > w.Org+P0 {
				x := w.Selectq
				w.Select(w.Org+P0, x)
			} else {
				x := w.Org + P0
				w.Select(x, w.Org+P1)
			}
		}
	} else {
		if w.Org+w.Nchars == w.Nr {
			return
		}
		r := w.Frame.Bounds()
		q0 = w.Org + w.IndexOf(image.Pt(r.Min.X, r.Min.Y+dl*w.Font.Dy()))
		if w.Sweeping {
			if w.Selectq >= w.Org+P1 {
				w.Select(w.Org+P1, w.Selectq)
			} else {
				w.Select(w.Selectq, w.Org+P1)
			}
		}
	}
	if w.Sweeping {
		w.flush()
	} // Put
	w.SetOrigin(q0, true)

}

func (w *Win) Select(q0, q1 int64) {
	w.Q0, w.Q1 = q0, q1
	p0 := clamp(q0-w.Org, 0, w.Nchars)
	p1 := clamp(q1-w.Org, 0, w.Nchars)
	pp0, pp1 := w.Frame.Dot()
	if p0 == pp0 && p1 == pp1 {
		return
	}
	if pp1 <= p0 || p1 <= pp0 || p0 == p1 || pp1 == pp0 {
		w.Redraw(w.PointOf(pp0), pp0, pp1, false)
		w.Redraw(w.PointOf(p0), p0, p1, true)
	} else {
		step := func(i, j int64) {
			if i < j {
				w.Redraw(w.PointOf(i), i, j, true)
			} else if i > j {
				w.Redraw(w.PointOf(j), j, i, false)
			}
		}
		step(p0, pp0) // trim or extend origin
		step(pp1, p1) // trim or extend insertion
	}
	w.Frame.Select(p0, p1)
}

func (w *Win) BackNL(p int64, n int) int64 {
	if n == 0 && p > 0 && w.R[p-1] != '\n' {
		n = 1
	}
	for i := n; i > 0 && p > 0; {
		i--
		p--
		if p == 0 {
			break
		}
		for j := 128; j-1 > 0 && p > 0; p-- {
			j--
			if p-1 < 0 || p-1 > w.Nr || w.R[p-1] == '\n' {
				break
			}
		}
	}
	return p
}

/*	SetS
func (w *Win) Drawscroll(){
	r := w.Scrollr
	b := w.Scrolltmp
	r1 := r
	r1.Min.X = 0
	r1.Max.X = r.Dx()
	r2 := scrpos(r1, w.Org, w.Org+w.Nchars, w.Nr)
}
*/

func (w *Win) SetOrigin(org int64, exact bool) {
	//fmt.Printf("SetOrigin: %d %v\n", org, exact)
	org = clamp(org, 0, w.Nr)
	if org > 0 && !exact {
		for i := 0; i < 256 && org < w.Nr; i++ {
			if w.R[org] == '\n' {
				org++
				break
			}
			org++
		}
	}
	a := org - w.Org // distance to new origin
	fix := false
	if a >= 0 && a < w.Nchars {
		// a bytes to the right; intersects the frame
		w.Frame.Delete(0, a)
		fix = true
	} else if a < 0 && -a < w.Nchars {
		// -a bytes to the left; intersects the frame
		i := org - a
		j := org
		if i > j {
			i, j = j, i
		}
		i = max(0, i)
		j = min(w.Nr, j)
		w.Frame.Insert(w.R[i:j], 0)
	} else {
		w.Frame.Delete(0, w.Nchars)
	}
	w.Org = org
	w.Fill()
	w.drawsb()
	w.Select(w.Q0, w.Q1)
	if P0, P1 := w.Frame.Dot(); fix && P1 > P0 {
		w.Redraw(w.PointOf(P1-1), P1-1, P1, true)
	}
}

func (w *Win) filldebug() {
	// Put
	fmt.Printf("lines/maxlines = %d/%d\n", w.Line(), w.MaxLine())
}

func (w *Win) Fill() {
	//w.filldebug()
	if w.Frame.Full() {
		return
	}
	var rp [MsgSize]byte
	for !w.Frame.Full() {
		qep := w.Org + w.Nchars
		n := min(w.Nr-qep, 2000)
		if n == 0 {
			break
		}
		m := copy(rp[:], w.R[qep:qep+n])
		nl := w.MaxLine() - w.Line()
		m = 0
		i := int64(0)
		for i < n {
			if rp[i] == '\n' {
				m++
				if m >= nl {
					i++
					break
				}
			}
			i++
		}
		w.Frame.Insert(rp[:i], w.Nchars)
	}
}

func (w *Win) Delete(q0, q1 int64) {
	n := q1 - q0
	if n == 0 {
		return
	}
	copy(w.R[q0:], w.R[q1:][:w.Nr-q1])
	w.Nr -= n
	if q0 < w.Q0 {
		w.Q0 -= min(n, w.Q0-q0)
	}
	if q0 < w.Q1 {
		w.Q1 -= min(n, w.Q1-q0)
	}
	if q1 < w.Qh {
		w.Qh = q0
	} else if q0 < w.Qh {
		w.Org -= n
	}

	if q1 <= w.Org {
		w.Org -= n
	} else if q0 < w.Org+w.Nchars {
		p1 := q1 - w.Org
		p0 := int64(0)
		if p1 > w.Nchars {
			p1 = w.Nchars
		}
		if q0 < w.Org {
			w.Org = q0
		} else {
			p0 = q0 - w.Org
		}
		w.Frame.Delete(p0, p1)
		w.Fill()
	}
}

func (w *Win) InsertString(s string, q0 int64) int64 {
	return w.Insert([]byte(s), q0)
}

func (w *Win) Insert(s []byte, q0 int64) int64 {
	n := int64(len(s))
	if n == 0 {
		return q0
	}
	if w.Nr+n > HiWater && q0 >= w.Org && q0 >= w.Qh {
		m := min(HiWater-LoWater, min(w.Org, w.Qh))
		w.Org -= m
		w.Qh -= m
		if w.Q0 > m {
			w.Q0 -= m
		} else {
			w.Q0 = 0
		}
		if w.Q1 > m {
			w.Q1 -= m
		} else {
			w.Q1 = 0
		}
		w.Nr -= m
		copy(w.R, w.R[m:][:w.Nr])
		q0 -= m
	}
	if w.Nr+n > w.Maxr {
		println("insert if D")
		m := max(min(2*(w.Nr+n), HiWater), w.Nr+n) + MinWater
		if m > HiWater {
			m = max(HiWater+MinWater, w.Nr+n)
		}
		if m > w.Maxr {
			extra := int64(m) - int64(len(w.R))
			w.R = append(w.R, make([]byte, extra)...)
			w.Maxr = m
		}
	}
	copy(w.R[q0+n:], w.R[q0:][:w.Nr-q0])
	copy(w.R[q0:], s[:n])
	w.Nr += n
	if q0 <= w.Q1 {
		w.Q1 += n
	}
	if q0 <= w.Q0 {
		w.Q0 += n
	}
	if q0 < w.Qh {
		w.Qh += n
	}
	if q0 < w.Org {
		w.Org += n
	} else if q0 <= w.Org+w.Nchars {
		n--
		if n < 0 {
			n++
		}
		w.Frame.Insert(s, q0-w.Org)
	}
	return q0
}

func (w *Win) upload() {
	w.events.Upload(w.Sp.Add(image.Pt(5, 5)), w.b, w.Bounds())
}
func (w *Win) flush() {
	sp := image.Pt(1, 1).Add(w.Sp)
	scrollsp := w.Sp
	s0 := w.Scrollr.Sub(w.Sp)
	r := w.Bounds()
	Ny := r.Dy() / 4
	r0 := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+Ny)
	r1 := image.Rect(r.Min.X, r.Min.Y+Ny, r.Max.X, r.Min.Y+Ny*2)
	r2 := image.Rect(r.Min.X, r.Min.Y+Ny*2, r.Max.X, r.Min.Y+Ny*3)
	r3 := image.Rect(r.Min.X, r.Min.Y+Ny*3, r.Max.X, r.Min.Y+Ny*4)
	var wg sync.WaitGroup
	sp.X = w.pad.X - w.Scrollr.Dx() + 1
	wg.Add(5)
	go func() { w.events.Upload(scrollsp, w.b, s0); wg.Done() }()
	sp = sp.Add(image.Pt(s0.Dx(), 11+5+1+1))
	go func() { w.events.Upload(sp, w.b, r0); wg.Done() }()
	go func() { w.events.Upload(sp.Add(image.Pt(0, Ny)), w.b, r1); wg.Done() }()
	go func() { w.events.Upload(sp.Add(image.Pt(0, Ny*2)), w.b, r2); wg.Done() }()
	go func() { w.events.Upload(sp.Add(image.Pt(0, Ny*3)), w.b, r3); wg.Done() }()
	w.Flushcache()
	wg.Wait()
}

// Put
func (w *Win) Upload() {
	///	w.upload()
	//	return
	var wg sync.WaitGroup
	wg.Add(len(w.Cache))
	sp := w.Sp
	for _, r := range w.Frame.Cache {
		go func(r image.Rectangle) {
			w.events.Upload(sp.Add(r.Min), w.b, r)
			wg.Done()
		}(r)
	}
	wg.Add(1)
	scrollsp := image.Pt(0, w.Sp.Y)
	go func() { w.events.Upload(scrollsp, w.b, w.Scrollr.Sub(w.Sp)); wg.Done() }()
	wg.Wait()
	w.Flushcache()
}

func (w *Win) ReadAt(off int64, p []byte) (n int, err error) {
	if off > w.Nr {
		return
	}
	return copy(p, w.R[off:w.Nr]), err

}

func (w *Win) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (w *Win) Bytes() []byte {
	return w.R[:w.Nr]
}

func (w *Win) Rdsel() []byte {
	i := w.Q0
	j := w.Q1
	return w.R[i:j]
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func clamp(v, l, h int64) int64 {
	if v < l {
		return l
	}
	if v > h {
		return h
	}
	return v
}

// Put
