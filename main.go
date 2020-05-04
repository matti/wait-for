package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type servicesType []string

var timeout int
var services servicesType

func (s *servicesType) String() string {
	return fmt.Sprintf("%+v", *s)
}

func (s *servicesType) Set(value string) error {
	*s = strings.Split(value, ",")
	return nil
}

type waitersType map[string]chan bool

// waitForServices tests and waits on the availability of a TCP host and port
func waitForServices(services []string, timeOut time.Duration) (waitersType, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	waiters := make(waitersType)

	var wg sync.WaitGroup
	wg.Add(len(services))
	for _, s := range services {
		waiters[s] = make(chan bool)

		go func(waiter chan bool, s string) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, err := net.Dial("tcp", s)
					if err == nil {
						close(waiter)
						return
					}
					time.Sleep(1 * time.Second)
				}
			}
		}(waiters[s], s)
	}

	go func() {
		wg.Wait()
		cancel()
	}()

	select {
	case <-ctx.Done():
		return waiters, nil
	case <-time.After(timeOut):
		cancel()
		<-ctx.Done()
		return waiters, fmt.Errorf("services aren't ready in %s", timeOut)
	}
}

func init() {
	flag.IntVar(&timeout, "t", 20, "timeout")
}

func main() {
	flag.Parse()
	services = flag.Args()
	fmt.Println(services)
	if len(services) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if waiters, err := waitForServices(services, time.Duration(timeout)*time.Second); err != nil {
		fmt.Println(err)

		for s, waiter := range waiters {
			select {
			case <-waiter:
				fmt.Println(s, "did open")
			default:
				fmt.Println(s, "did not open")
			}
		}

		os.Exit(1)
	}
	fmt.Println("services are ready!")
	os.Exit(0)
}
