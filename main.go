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

type service struct {
	sleepSeconds time.Duration
}

func newService(sleepSeconds int) service {
	return service{time.Duration(sleepSeconds) * time.Second}
}

func (s service) Hi(srv hello.Connector_HiServer) error {
	log.Println("oh, someone just said 'Hi'")
	for {
		d, err := srv.Recv()
		if err != nil {
			log.Printf("receive error: %v", err)
			break
		}
		log.Printf("got one message (%d bytes), going to sleep %f", len(d.Chunk), s.sleepSeconds.Seconds())
		time.Sleep(s.sleepSeconds)
	}
	log.Printf("try to read buffered messages")
	n := 0
	var err error
	for err == nil {
		_, err = srv.Recv()
		if err == nil {
			n += 1
		}
	}
	log.Printf("got %d buffered messages, Recv() error: %v, closing stream", n, err)
	if err := srv.SendAndClose(&empty.Empty{}); err != nil {
		log.Printf("close stream error: %v", err)
	}
	return nil
}

func startServer(addr string, sleepSeconds int) {
	s := newService(sleepSeconds)
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

const data = `We are actively soliciting feedback on all these proposals. We are especially interested in fact-based evidence illustrating why a proposal might not work well in practice, or problematic aspects we might have missed in the design. Convincing examples in support of a proposal are also very helpful. On the other hand, comments containing only personal opinions are less actionable: we can acknowledge them but we can’t address them in any constructive way. Before posting, please take the time to read the detailed design docs and prior feedback or feedback summaries. Especially in long discussions, your concern may have already been raised and discussed in earlier comments.

Unless there are strong reasons to not even proceed into the experimental phase with a given proposal, we are planning to have all these implemented at the start of the Go 1.14 cycle (beginning of August, 2019) so that they can be evaluated in practice. Per the proposal evaluation process, the final decision will be made at the end of the development cycle (beginning of November, 2019).

Thank you for helping making Go a better language!

By Robert Griesemer, for the Go team`

func startClient(addr string, concurrency int) {
	if concurrency < 1 {
		concurrency = 1
	}

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

			reconnect := true
			for reconnect {
				c := hello.NewConnectorClient(conn)
				stream, err := c.Hi(ctx)
				if err != nil {
					log.Printf("[#%d] ERROR: failed to call Hi: %v", id, err)
					break
				}
				log.Printf("[#%d] send %d bytes of data", id, len(data))
				var n int
				tick := time.Tick(1 * time.Nanosecond)
				for range tick {
					n += 1
					log.Printf("[#%d] sending heartbeat: %d", id, n)
					if err := stream.Send(&hello.Data{
						Chunk: data,
					}); err != nil {
						log.Printf("[#%d] send error: %v", id, err)
						if err.Error() != "EOF" {
							reconnect = false
						}
						break
					}
				}
				if _, err := stream.CloseAndRecv(); err != nil {
					log.Printf("[#%d] receive error: %v", id, err)
					continue
				}
				log.Printf("[#%d] received and closed", id)
			}
		}(i)
	}
	wg.Wait()
}

func main() {
	if len(os.Args) < 3 {
		log.Panic(`
Usage:

1. To run as a server:
  %s server <addr> [seconds-to-sleep-before-close-stream]

2. To run as a client:
  %s client <addr> [client-concurrency]
		`,
			os.Args[0],
		)
	}
	switch os.Args[1] {
	case "server":
		t := 24 * 60 * 60
		if len(os.Args) > 3 {
			t, _ = strconv.Atoi(os.Args[3])
		}
		startServer(os.Args[2], t)
	default:
		n := 1
		if len(os.Args) > 3 {
			n, _ = strconv.Atoi(os.Args[3])
		}
		startClient(os.Args[2], n)
	}
}
