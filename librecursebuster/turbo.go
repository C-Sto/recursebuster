package librecursebuster

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//Based on James Kettle's turbo intruder talk, http pipelining!
type turboHTTP struct {
	tcpConns  []net.Conn
	tcpMut    *sync.RWMutex
	sendCount *uint64
	recvCount *uint64
	recvMap   map[uint64][]byte
}

func (t turboHTTP) New() *turboHTTP {
	c, e := net.Dial("tcp", "www.stealmylogin.com:80")
	if e != nil {
		panic(e)
	}
	ret := &turboHTTP{
		tcpConns:  []net.Conn{c},
		tcpMut:    &sync.RWMutex{},
		sendCount: new(uint64),
		recvCount: new(uint64),
		recvMap:   make(map[uint64][]byte),
	}
	go ret.lol()
	return ret
}

func (t *turboHTTP) send(b []byte) uint64 {
	t.tcpMut.Lock()
	defer t.tcpMut.Unlock()
	atomic.AddUint64(t.sendCount, 1)
	t.tcpConns[0].Write(b)
	return *t.sendCount
}

func (t *turboHTTP) Do(req *http.Request, wg *sync.WaitGroup) (*http.Response, error) {
	defer wg.Done()

	//get body as byte array
	buff := &bytes.Buffer{}
	e := req.Write(buff)
	if e != nil {
		//do error
		panic(e)
	}

	i := t.send(buff.Bytes())

	for {
		if val, ok := t.recvMap[i]; ok {
			//reader := bytes.NewReader(val)
			rd := bufio.NewReader(bytes.NewReader(val))
			hdrd, e := http.ReadResponse(rd, req)
			return hdrd, e
		}
		time.Sleep(time.Second * 1)
	}

}

func (t *turboHTTP) lol() {

	for {
		//look for a response header
		b := make([]byte, 8)
		t.tcpConns[0].Read(b)
		if bytes.Compare(b, []byte("HTTP/1.1")) == 0 {
			//fmt.Println("RESP HEADER!")
			ln := 0
			headLen := 0
			//we are in a response. Read it until we find a end header sequence
			for {
				bb := make([]byte, 1)
				t.tcpConns[0].Read(bb)
				if bytes.HasSuffix(b, []byte{13, 10, 13, 10}) { //&& bb[0] == 0x0a {
					//found head, look for content length
					headLen = len(b)
					cl := "Content-Length"
					headers := bytes.Split(b, []byte{13, 10})
					for _, header := range headers {
						//fmt.Println(string(header))
						if strings.HasPrefix(string(header), cl) {
							//fmt.Println("CONTENT LENGTH!!!!")
							//check content length
							var e error
							ln, e = strconv.Atoi(strings.Split(string(header), ":")[1][1:])
							if e != nil {
								panic(e)
							}
						}
					}
					//fmt.Println("HEAD END")
				}

				b = append(b, bb[0])
				if ln != 0 && len(b)-headLen == ln {
					atomic.AddUint64(t.recvCount, 1)
					t.recvMap[*t.recvCount] = b
					break
				}
			}
		}
	}
}
