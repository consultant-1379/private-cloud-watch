package expect

import (
	"io"
	"unicode/utf8"

	"github.com/erixzone/xaas/platform/ssh-utils/pkg/regexp"
)

const (
	nAlloc = 1024
)

type Expecter struct {
	src     io.Reader
	debug   io.Writer
	reDict  map[string]*regexp.Regexp
	buf     []byte
	unread  bool
	lastc   rune
	lastw   int
	rptr    int
	wptr    int
	mlo     int
	mhi     int
	err     error
}

func NewExpecter(src io.Reader) *Expecter {
	r := new(Expecter)
	r.src = src
	return r
}

func (r *Expecter) Debug(w io.Writer) {
	r.debug = w
}

func (r *Expecter) Reset() {
	r.unread = false
	r.lastc = 0
	r.lastw = 0
	r.rptr = 0
	r.wptr = 0
	r.mlo = 0
	r.mhi = 0
	r.err = nil
}

func (r *Expecter) Payload() []byte {
	return r.buf[0:r.mlo]
}

func (r *Expecter) CopyPayload() []byte {
	p := make([]byte, r.mlo)
	copy(p, r.buf[0:r.mlo])
	return p
}

func (r *Expecter) Match() []byte {
	return r.buf[r.mlo:r.mhi]
}

func (r *Expecter) CopyMatch() []byte {
	p := make([]byte, r.mhi-r.mlo)
	copy(p, r.buf[r.mlo:r.mhi])
	return p
}

func Compile(reStr string) (*regexp.Regexp, error) {
	re, err := regexp.Compile(reStr)
	if re != nil {
		re.First()
	}
	return re, err
}

func (r *Expecter) Expect(reStr string) error {
	var re *regexp.Regexp
	var err error
	if r.reDict == nil {
		 r.reDict = make(map[string]*regexp.Regexp)
	} else {
		re = r.reDict[reStr]
	}
	if re == nil {
		re, err = Compile(reStr)
		if err != nil {
			return err
		}
		r.reDict[reStr] = re
	}
	return r.ExpectRe(re)
}

func (r *Expecter) ExpectRe(re *regexp.Regexp) error {
	if n := r.wptr - r.mhi; n > 0 {
		copy(r.buf, r.buf[r.mhi:r.wptr])
		r.wptr = n
	} else {
		r.wptr = 0
	}
	r.rptr = 0
	r.mlo = 0
	r.mhi = 0
	loc := re.FindReaderIndex(r)
	if loc != nil {
		r.mlo = loc[0]
		r.mhi = loc[1]
	}
	return r.err
}

func (r *Expecter) ReadRune() (rune, int, error) {
	if r.unread {
		r.unread = false
		return r.lastc, r.lastw, nil
	}
	for r.rptr >= r.wptr {
		if err := r.fill(); err != nil {
			r.lastc, r.lastw = -1, 0
			return -1, 0, err
		}
	}
	if c := r.buf[r.rptr]; c < utf8.RuneSelf {
		r.rptr++
		r.lastc, r.lastw = rune(c), 1
		return rune(c), 1, nil
	}
	for !utf8.FullRune(r.buf[r.rptr:r.wptr]) {
		if err := r.fill(); err != nil {
			r.lastc, r.lastw = -1, 0
			return -1, 0, err
		}
	}
	c, w := utf8.DecodeRune(r.buf[r.rptr:r.wptr])
	r.rptr += w
	r.lastc, r.lastw = c, w
	return c, w, nil
}

func (r *Expecter) UnreadRune() error {
	r.unread = true
	return nil
}

func (r *Expecter) fill() error {
	if len(r.buf)-r.wptr < nAlloc {
		r.buf = append(r.buf, make([]byte, nAlloc)...)
	}
	var n int
	n, r.err = r.src.Read(r.buf[r.wptr:])
	if n > 0 {
		if r.debug != nil {
			r.debug.Write(r.buf[r.wptr:r.wptr+n])
		}
		r.wptr += n
	}
	return r.err
}
