package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	"github.com/rpcxio/rpcx-benchmark/proto"
	rlog "github.com/smallnest/rpcx/log"
	"github.com/smallnest/rpcx/server"
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
	host       = flag.String("s", "127.0.0.1:8972", "listened ip and port")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	delay      = flag.Duration("delay", 0, "delay to mock business processing")
	debugAddr  = flag.String("d", "127.0.0.1:9981", "server ip and port")
	usePool    = flag.Bool("p", false, "use pool or not")
)

func main() {
	flag.Parse()

	server.UsePool = true

	rlog.SetDummyLogger()

	go func() {
		log.Println(http.ListenAndServe(*debugAddr, nil))
	}()

	var rpcxserver *server.Server

	if *usePool {
		rpcxserver = server.NewServer(server.WithPool(100, 1000000))
	} else {
		rpcxserver = server.NewServer()
	}
	// server.AsyncWrite = true

	rpcxserver.RegisterName("Hello", new(Hello), "")
	rpcxserver.Serve("tcp", *host)
}
