package main

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Gimi/fly/hello"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const data = `We are actively soliciting feedback on all these proposals. We are especially interested in fact-based evidence illustrating why a proposal might not work well in practice, or problematic aspects we might have missed in the design. Convincing examples in support of a proposal are also very helpful. On the other hand, comments containing only personal opinions are less actionable: we can acknowledge them but we can’t address them in any constructive way. Before posting, please take the time to read the detailed design docs and prior feedback or feedback summaries. Especially in long discussions, your concern may have already been raised and discussed in earlier comments.

Unless there are strong reasons to not even proceed into the experimental phase with a given proposal, we are planning to have all these implemented at the start of the Go 1.14 cycle (beginning of August, 2019) so that they can be evaluated in practice. Per the proposal evaluation process, the final decision will be made at the end of the development cycle (beginning of November, 2019).

Thank you for helping making Go a better language!

By Robert Griesemer, for the Go team`

type service struct{}

func (s service) Hi(_ *empty.Empty, stream hello.Connector_HiServer) error {
	log.Println("oh, someone just said 'Hi'")
	log.Printf("sending %d-byte length of data", len(data))
	for n := 0; n < 1000; n++ {
		batch := make([]string, 1000)
		for i := range batch {
			batch[i] = data
		}
		// log.Printf("sending data No. %d", n)
		if err := stream.Send(&hello.Data{
			Chunk: batch,
		}); err != nil {
			log.Printf("send error: %v", err)
			break
		}
	}
	log.Printf("Byebye")
	return nil
}

func startServer(addr string) {
	s := service{}
	conn := grpc.NewServer()
	hello.RegisterConnectorServer(conn, s)
	// Register reflection conn on gRPC server.
	reflection.Register(conn)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("Failed to listen %s ERROR: %v", addr, err)
		return
	}

	log.Println("listening income requests")
	if err := conn.Serve(lis); err != nil {
		log.Printf("ERROR: %v", err)
	}
	time.Sleep(5 * time.Second)
	log.Println("Bye bye")
}

func startClient(addr string, concurrency int, interval int) {
	if concurrency < 1 {
		concurrency = 1
	}

	iv := time.Duration(interval) * time.Second

	ctx, cancel := context.WithCancel(context.TODO())
	dialCtx, _ := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(dialCtx, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Printf("ERROR: failed to connect to server (%s): %v", addr, err)
		return
	}
	log.Printf("Connected to %s", addr)

	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			myCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			c := hello.NewConnectorClient(conn)

			stream, err := c.Hi(myCtx, &empty.Empty{})
			if err != nil {
				log.Printf("[#%d] ERROR: failed to call Hi: %v", id, err)
				return
			}

			n := 0
			time.Tick(iv)
			t := time.Now()
			for {
				data, err := stream.Recv()
				if err != nil {
					log.Printf("receive error: %v", err)
					break
				}
				n += len(data.Chunk)
				// log.Printf("got No. %d message (%d bytes), wait for another %f seconds", n, len(d.Chunk), iv.Seconds())
			}
			log.Printf("Done. Messages: %d, Time: %f seconds", n, time.Now().Sub(t).Seconds())
		}(i)
	}
	wg.Wait()
}

func main() {
	if len(os.Args) < 3 {
		log.Panic(`
Usage:

1. To run as a server:
  %s server <addr>

2. To run as a client:
  %s client <addr> [client-concurrency] [recv-interval-in-second]
		`,
			os.Args[0],
		)
	}
	switch os.Args[1] {
	case "server":
		startServer(os.Args[2])
	default:
		n := 1
		if len(os.Args) > 3 {
			n, _ = strconv.Atoi(os.Args[3])
		}

		t := 1
		if len(os.Args) > 4 {
			t, _ = strconv.Atoi(os.Args[4])
		}
		startClient(os.Args[2], n, t)
	}
}
