package librecursebuster

import (
	"fmt"
	"io"
	"sync"
	"time"
)

type ConsoleWriter struct {
	mu     sync.Mutex
	prefix string
	flag   int
	out    io.Writer
	buf    []byte
}

//This is super stupid, I should use a lib for this

func (l *ConsoleWriter) formatHeader(buf *[]byte, t time.Time, file string, line int) {
	*buf = append(*buf, l.prefix...)
	year, month, day := t.Date()
	itoa(buf, year, 4)
	*buf = append(*buf, '/')
	itoa(buf, int(month), 2)
	*buf = append(*buf, '/')
	itoa(buf, day, 2)
	*buf = append(*buf, ' ')

	hour, min, sec := t.Clock()
	itoa(buf, hour, 2)
	*buf = append(*buf, ':')
	itoa(buf, min, 2)
	*buf = append(*buf, ':')
	itoa(buf, sec, 2)
	*buf = append(*buf, ' ')

}

func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

func (ConsoleWriter) New(w io.Writer, prefix string) *ConsoleWriter {
	return &ConsoleWriter{out: w, prefix: prefix, flag: 0}
}

func (l *ConsoleWriter) Output(calldepth int, s string) error {
	now := time.Now() // get this early.
	var file string
	var line int
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf = l.buf[:0]
	l.formatHeader(&l.buf, now, file, line)
	l.buf = append(l.buf, s...)
	_, err := l.out.Write(l.buf)
	return err
}

func (c *ConsoleWriter) Println(v ...interface{}) { c.Output(2, fmt.Sprintln(v...)) }

func (l *ConsoleWriter) Printf(format string, v ...interface{}) {
	l.Output(2, fmt.Sprintf(format, v...))
}
