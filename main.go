package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/codahale/hdrhistogram"
)

type memcache struct {
	c net.Conn
}

func newMemcache(server string, timeout time.Duration) (*memcache, error) {
	c, err := net.Dial("tcp", server)
	if err != nil {
		return nil, err
	}

	m := memcache{}
	m.c = c

	return &m, nil
}

func (m *memcache) close() {
	m.c.Close()
}

func (m *memcache) get(key string, timeout time.Duration) error {

	cmd := "get " + key + "\r\n"

	_, err := m.c.Write([]uint8(cmd))
	if err != nil {
		return err
	}

	reader := bufio.NewReader(m.c)

	_, err = reader.ReadString('\n')
	if err != nil {
		return err
	}

	return nil
}

func main() {

	var (
		sleep   = time.Millisecond * 10
		timeout = time.Second * 5
		repeats uint64
		errors  uint64
		uptime  = time.Now()
	)

	connectHistogram := hdrhistogram.New(time.Millisecond.Nanoseconds(), (time.Millisecond * 100).Nanoseconds(), 2)
	getHistogram := hdrhistogram.New(time.Millisecond.Nanoseconds(), (time.Millisecond * 100).Nanoseconds(), 2)

	for {
		repeats++
		time.Sleep(sleep)

		func() {

			start := time.Now()

			mc, err := newMemcache("memcached2.mlan:11211", timeout)
			if err != nil {
				log.Printf("Error connecting to memcache: %s", err)
				errors++
				return
			}
			defer mc.close()

			connectHistogram.RecordValue(time.Since(start).Nanoseconds())

			start = time.Now()

			if err := mc.get("foo", timeout); err != nil {
				log.Printf("Error while getting from memcached: %s", err)
				errors++
			}

			getHistogram.RecordValue(time.Since(start).Nanoseconds())
		}()

		if repeats%1000 == 0 {
			fmt.Print("\033[H\033[2J")
			fmt.Printf("Uptime: %v\n", time.Since(uptime))
			fmt.Printf("Errors: %v\n", errors)
			fmt.Println("---- CONNECT ----")
			fmt.Printf("count: %v\n", repeats)
			distribution := connectHistogram.Distribution()
			for _, b := range distribution {
				if b.Count == 0 {
					continue
				}
				fmt.Printf("%v\t%v\t%v\n", time.Duration(b.From), time.Duration(b.To), b.Count)
			}

			fmt.Println("---- GET ----")
			fmt.Printf("count: %v\n", repeats)
			distribution = getHistogram.Distribution()
			for _, b := range distribution {
				if b.Count == 0 {
					continue
				}
				fmt.Printf("%v\t%v\t%v\n", time.Duration(b.From), time.Duration(b.To), b.Count)
			}
		}
	}
}
