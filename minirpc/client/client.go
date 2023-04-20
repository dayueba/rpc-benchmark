package main

import (
	"flag"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dayueba/minirpc"
	"github.com/rpcxio/rpcx-benchmark/stat"
	"github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"

	"github.com/dayueba/minirpc/protocol"
	"github.com/rpcxio/rpcx-benchmark/proto"
)

var (
	concurrency = flag.Int("c", 1, "concurrency")
	total       = flag.Int("n", 10000, "total requests for all clients")
	host        = flag.String("s", "127.0.0.1:8972", "server ip and port")
	pool        = flag.Int("pool", 10, "shared rpcx clients")
	rate        = flag.Int("r", 0, "throughputs")
)

func main() {
	flag.Parse()

	var rl ratelimit.Limiter
	if *rate > 0 {
		rl = ratelimit.New(*rate)
	}

	// 并发goroutine数.模拟客户端
	n := *concurrency
	// 每个客户端需要发送的请求数
	m := *total / n
	logrus.Infof("concurrency: %d\nrequests per client: %d\n\n", n, m)

	// 准备好参数
	args := proto.PrepareArgs()

	// 参数的大小
	b := make([]byte, 1024)
	i, _ := args.MarshalTo(b)
	logrus.Infof("message size: %d bytes\n\n", i)

	// 等待所有测试完成
	var wg sync.WaitGroup
	wg.Add(n * m)

	// 创建客户端连接池
	var clientIndex uint64
	poolClients := make([]minirpc.Client, 0, *pool)

	for i := 0; i < *pool; i++ {
		client, err := minirpc.NewClient(*host, minirpc.WithSerializeType(protocol.ProtoBuffer))
		defer client.Close()
		if err != nil {
			panic(err)
		}

		// warmup
		var reply proto.BenchmarkMessage
		for j := 0; j < 5; j++ {
			client.Call("Hello", "Say", args, &reply)
		}

		poolClients = append(poolClients, client)
	}

	// 栅栏，控制客户端同时开始测试
	var startWg sync.WaitGroup
	startWg.Add(n + 1) // +1 是因为有一个goroutine用来记录开始时间

	// 总请求数
	var trans uint64
	// 返回正常的总请求数
	var transOK uint64

	// 每个goroutine的耗时记录
	d := make([][]int64, n, n)

	// 创建客户端 goroutine 并进行测试
	startTime := time.Now().UnixNano()
	go func() {
		startWg.Done()
		startWg.Wait()
		startTime = time.Now().UnixNano()
	}()
	for i := 0; i < n; i++ {
		dt := make([]int64, 0, m)
		d = append(d, dt)

		go func(i int) {
			var reply proto.BenchmarkMessage

			startWg.Done()
			startWg.Wait()

			for j := 0; j < m; j++ {
				// 限流，这里不把限流的时间计算到等待耗时中
				if rl != nil {
					rl.Take()
				}

				t := time.Now().UnixNano()
				ci := atomic.AddUint64(&clientIndex, 1)
				ci = ci % uint64(*pool)
				client := poolClients[int(ci)]

				// err := client.Call("Hello.Say", args, &reply, arpc.TimeForever)
				err := client.Call("Hello", "Say", args, &reply)
				t = time.Now().UnixNano() - t // 等待时间+服务时间，等待时间是客户端调度的等待时间以及服务端读取请求、调度的时间，服务时间是请求被服务处理的实际时间

				d[i] = append(d[i], t)

				if err == nil && reply.Field1 == "OK" {
					atomic.AddUint64(&transOK, 1)
				}

				atomic.AddUint64(&trans, 1)
				wg.Done()
			}
		}(i)

	}

	// 等待测试完成
	wg.Wait()

	// 统计
	stat.Stats(startTime, *total, d, trans, transOK)
}
