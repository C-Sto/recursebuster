package librecursebuster

import (
	"fmt"
	"io"
	"sync"
	"time"
)

//ConsoleWriter is pretty much a straight rip from the go project's log writer, but modified as needed https://golang.org/src/log/log.go
type ConsoleWriter struct {
	mu     *sync.Mutex
	prefix string
	flag   int
	out    io.Writer
	buf    []byte
}

//This is super stupid, I should use a lib for this

func (c *ConsoleWriter) formatHeader(buf *[]byte, t time.Time, file string, line int) {
	*buf = append(*buf, c.prefix...)
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

// New creates a new Console writer.
// The prefix appears at the beginning of each generated log line.
func (ConsoleWriter) New(w io.Writer, prefix string) *ConsoleWriter {
	m := &sync.Mutex{}
	return &ConsoleWriter{out: w, prefix: prefix, flag: 0, mu: m}
}

// Output writes the output for an event. The string s contains
// the text to print after the prefix specified by the flags of the
// Logger. A newline is appended if the last character of s is not
// already a newline. Calldepth is used to recover the PC and is
// provided for generality, although at the moment on all pre-defined
// paths it will be 2.
func (c *ConsoleWriter) Output(calldepth int, s string) error {
	now := time.Now() // get this early.
	var file string
	var line int
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buf = c.buf[:0]
	c.formatHeader(&c.buf, now, file, line)
	c.buf = append(c.buf, s...)
	_, err := c.out.Write(c.buf)
	return err
}

// Println calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Println.
func (c *ConsoleWriter) Println(v ...interface{}) {
	err := c.Output(2, fmt.Sprintln(v...))
	if err != nil {
		fmt.Println(err)
	}
}

// Printf calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
func (c *ConsoleWriter) Printf(format string, v ...interface{}) {
	err := c.Output(2, fmt.Sprintf(format, v...))
	if err != nil {
		fmt.Println(err)
	}
}
