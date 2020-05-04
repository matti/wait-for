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

	log "github.com/sirupsen/logrus"
)

type servicesType []string

var timeout time.Duration
var services servicesType

func (s *servicesType) String() string {
	return fmt.Sprintf("%+v", *s)
}

func (s *servicesType) Set(value string) error {
	*s = strings.Split(value, ",")
	return nil
}

type waiterType chan struct{}
type waiterMap map[string]waiterType

type serviceResults map[string]bool

// waitForServices tests and waits on the availability of a TCP host and port
func waitForServices(ctx context.Context, cancel context.CancelFunc, waiters waiterMap) {
	var wg sync.WaitGroup
	wg.Add(len(waiters))
	for service, waiter := range waiters {
		go func(service string, waiter waiterType) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, err := net.Dial("tcp", service)
					if err == nil {
						close(waiter)
						return
					}
					time.Sleep(1 * time.Second)
				}
			}
		}(service, waiter)
	}

	go func() {
		wg.Wait()
		cancel()
	}()

	<-ctx.Done()
}

func init() {
	flag.DurationVar(&timeout, "t", time.Duration(3*time.Second), "timeout")
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.SetReportCaller(true)

	if levelString := os.Getenv("DEBUG"); levelString != "" {
		level, err := log.ParseLevel(levelString)
		if err != nil {
			panic(err)
		}
		log.SetLevel(level)
	}

	flag.Parse()
	services = flag.Args()
	log.Debug("services", services)
	if len(services) == 0 {
		flag.Usage()
		os.Exit(1)
	}
	waiters := make(waiterMap)
	for _, s := range services {
		waiters[s] = make(waiterType)
	}
	go waitForServices(ctx, cancel, waiters)
	<-ctx.Done()
	fmt.Println("services are ready!")

	switch ctx.Err() {
	case context.Canceled:
		os.Exit(0)
	case context.DeadlineExceeded:
		for service, waiter := range waiters {
			select {
			case <-waiter:
				log.Info(service, " did open")
			default:
				log.Info(service, " did not open")
			}
		}

		os.Exit(1)
	default:
		panic("not handled")
	}
}
