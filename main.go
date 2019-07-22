package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/Gimi/fly/hello"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type service struct{}

func (s service) Hi(srv hello.Connector_HiServer) error {
	log.Println("oh, someone just said 'Hi'")
	d, err := srv.Recv()
	if err != nil {
		log.Printf("receive error: %v", err)
		return err
	}
	log.Printf("got one message, going to sleep\n%s", d.Chunk)
	time.Sleep(24 * 60 * 60 * time.Second)
	if err := srv.SendAndClose(&empty.Empty{}); err != nil {
		log.Printf("send error: %v", err)
	}
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

const data = `We are actively soliciting feedback on all these proposals. We are especially interested in fact-based evidence illustrating why a proposal might not work well in practice, or problematic aspects we might have missed in the design. Convincing examples in support of a proposal are also very helpful. On the other hand, comments containing only personal opinions are less actionable: we can acknowledge them but we can’t address them in any constructive way. Before posting, please take the time to read the detailed design docs and prior feedback or feedback summaries. Especially in long discussions, your concern may have already been raised and discussed in earlier comments.

Unless there are strong reasons to not even proceed into the experimental phase with a given proposal, we are planning to have all these implemented at the start of the Go 1.14 cycle (beginning of August, 2019) so that they can be evaluated in practice. Per the proposal evaluation process, the final decision will be made at the end of the development cycle (beginning of November, 2019).

Thank you for helping making Go a better language!

By Robert Griesemer, for the Go team`

func startClient(addr string) {
	ctx, cancel := context.WithCancel(context.TODO())
	dialCtx, _ := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(dialCtx, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Printf("ERROR: failed to connect to server (%s): %v", addr, err)
		return
	}
	log.Printf("Connected to %s", addr)

	c := hello.NewConnectorClient(conn)
	stream, err := c.Hi(ctx)
	if err != nil {
		log.Printf("ERROR: failed to call Hi: %v", err)
		return
	}
	log.Printf("send %d bytes of data", len(data))
	var n int
	tick := time.Tick(1 * time.Nanosecond)
	for now := range tick {
		n += 1
		log.Printf("%v sending heartbeat: %d", now, n)
		if err := stream.Send(&hello.Data{
			Chunk: data,
		}); err != nil {
			log.Printf("send error: %v", err)
			break
		}
	}
	if _, err := stream.CloseAndRecv(); err != nil {
		log.Printf("receive error: %v", err)
		return
	}
	log.Printf("received and closed")
}

func main() {
	if len(os.Args) != 3 {
		log.Panic("Usage: " + os.Args[0] + " <server|client> <addr>")
	}
	switch os.Args[1] {
	case "server":
		startServer(os.Args[2])
	default:
		startClient(os.Args[2])
	}
}
