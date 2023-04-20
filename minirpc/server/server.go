package main

import (
	"context"
	"flag"
	"runtime"
	"time"

	//_ "net/http/pprof"

	"github.com/dayueba/minirpc"
	"github.com/rpcxio/rpcx-benchmark/proto"
)

type Hello int

func (t *Hello) Say(ctx context.Context, args *proto.BenchmarkMessage, reply *proto.BenchmarkMessage) error {
	args.Field1 = "OK"
	args.Field2 = 100
	*reply = *args
	if *delay > 0 {
		time.Sleep(*delay)
	} else {
		runtime.Gosched()
	}
	return nil
}

var (
	host  = flag.String("s", "127.0.0.1:8972", "listened ip and port")
	delay = flag.Duration("delay", 0, "delay to mock business processing by sleep")
)

func main() {
	flag.Parse()

	//go func() {
	//	http.ListenAndServe(":6060", nil)
	//}()

	server := minirpc.NewServer(*host)

	err := server.RegisterName("Hello", new(Hello))
	if err != nil {
		panic(err)
	}
	if err = server.Start(); err != nil {
		panic(err)
	}
}
