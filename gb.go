package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var testMsgLen1 = flag.Int("ll", 26, "test message length")

func serverRun() {
	flag.Parse()

	if lsn, err := net.Listen("tcp", "0.0.0.0:22335"); err == nil {
		for {
			conn, err := lsn.Accept()

			if err != nil {
				break
			}

			sendChan := make(chan []byte, 1000)

			go func() {
				defer func() {
					conn.Close()
					close(sendChan)
				}()

				msg := make([]byte, *testMsgLen1)

			L:
				for {
					if _, err := conn.Read(msg); err == nil {
						select {
						case sendChan <- msg:
						default:
							break L
						}
					} else {
						break L
					}
				}
			}()

			go func() {
				defer conn.Close()

				for {
					msg, ok := <-sendChan

					if !ok {
						break
					}

					if _, err := conn.Write(msg); err != nil {
						break
					}
				}
			}()
		}
	}
}

var (
	targetAddr  = flag.String("a", "127.0.0.1:22335", "target echo server address")
	testMsgLen  = flag.Int("l", 26, "test message length")
	testConnNum = flag.Int("c", 50, "test connection number")
	testSeconds = flag.Int("t", 10, "test duration in seconds")
)

func clientRun() {
	flag.Parse()

	var (
		outNum uint64
		inNum  uint64
		stop   uint64
	)

	msg := make([]byte, *testMsgLen)

	go func() {
		time.Sleep(time.Second * time.Duration(*testSeconds))
		atomic.StoreUint64(&stop, 1)
	}()

	wg := new(sync.WaitGroup)

	for i := 0; i < *testConnNum; i++ {
		wg.Add(1)

		go func() {
			if conn, err := net.DialTimeout("tcp", *targetAddr, time.Minute*99999); err == nil {
				l := len(msg)
				recv := make([]byte, l)

				for {
					for rest := l; rest > 0; {
						i, err := conn.Write(msg)
						rest -= i
						if err != nil {
							log.Println(err)
							break
						}
					}

					atomic.AddUint64(&outNum, 1)

					if atomic.LoadUint64(&stop) == 1 {
						break
					}

					for rest := l; rest > 0; {
						i, err := conn.Read(recv)
						rest -= i
						if err != nil {
							log.Println(err)
							break
						}
					}

					atomic.AddUint64(&inNum, 1)

					if atomic.LoadUint64(&stop) == 1 {
						break
					}
				}
			} else {
				log.Println(err)
			}

			wg.Done()
		}()
	}

	wg.Wait()

	fmt.Println("Benchmarking:", *targetAddr)
	fmt.Println(*testConnNum, "clients, running", *testMsgLen, "bytes,", *testSeconds, "sec.")
	fmt.Println()
	fmt.Println("Speed:", outNum/uint64(*testSeconds), "request/sec,", inNum/uint64(*testSeconds), "response/sec")
	fmt.Println("Requests:", outNum)
	fmt.Println("Responses:", inNum)
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("server run...")
		serverRun()
	} else {
		fmt.Println("client run...")
		clientRun()
	}

}
