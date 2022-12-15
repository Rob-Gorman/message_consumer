package grpcclient

import (
	"context"
	"delivery/logger"
	"delivery/proto"
	"delivery/types"
	"fmt"
	"io"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	grpcServerAddr = "platform:50051"
	stream         = "pb-stream"
	group          = "pb-delivery"
	consumerID     = 1
	consumer       = fmt.Sprintf("consumer-%d", consumerID)
)

// implements consumer.MessageConsumer
type GrpcClient struct {
	client   proto.PostbackStreamClient
	consData *proto.ConsumerData
	conn     *grpc.ClientConn
	log      logger.AppLogger
}

// calls grpcSubscribeStream method to begin reading from gRPC stream and
// parse messages onto the return channel
func (gc *GrpcClient) NewMessageChannel(ctx context.Context) <-chan *types.Message {
	ch := make(chan *types.Message)
	pbch := gc.grpcSubscribeStream(ctx)

	go func() {
		defer close(ch)
		for pb := range pbch { // will return when parent context closes pbch
			ch <- postbackToMessage(pb)
		}
	}()

	return ch
}

// initializes a client stream to AckPostback service. `ids` provides the message ids to ack
// a goroutine is initialized to continually read from `ids` channel and `Send()` via the stream client
func (gc *GrpcClient) AckMessages(ctx context.Context, ids <-chan interface{}) error {
	ack, err := gc.client.AckPostback(ctx, grpc.WaitForReady(true))
	if err != nil {
		gc.log.Error("failed to initialize Ack stream", err.Error())
	}

	go func() {
		// defer ack.CloseSend()
		for id := range ids {
			idstr, _ := id.(string) // type assertion for proto.AckData struct
			msg := &proto.AckData{MessageID: idstr}
			err = ack.Send(msg)
			if err != nil {
				gc.log.Error("failed to ack message:", idstr)
				// ideally we're persisting this data to attempt to ack later
				// depends how expensive it is to duplicate messages or check on the
				// recieving side
			}
			ack.Recv()
		}
	}()
	return err
}

// initializes a new stream to the Subscribe service and returns a live channel
// of messages from that stream
func (gc *GrpcClient) grpcSubscribeStream(ctx context.Context) <-chan *proto.Postback {
	ch := make(chan *proto.Postback)

	sub, err := gc.client.Subscribe(
		ctx,
		gc.consData,
		grpc.WaitForReady(gc.waitUntilReady()),
	)
	if err != nil {
		gc.log.Error("failed to subscribe to service", err.Error())
	}

	go func() {
		defer close(ch)
		for {
			pb, err := sub.Recv()
			if err == io.EOF {
				gc.log.Info("server has closed connection")
				return
			}
			if err != nil {
				gc.log.Error("error recieving from grpc server", err.Error())
				return
			}

			ch <- pb
		}
	}()

	return ch
}

func postbackToMessage(pb *proto.Postback) *types.Message {
	return &types.Message{
		Method: pb.GetMethod(),
		URL:    pb.GetUrl(),
		AckID:  pb.GetAckID(),
	}
}

func New(ctx context.Context, l logger.AppLogger) *GrpcClient {
	var (
		opts = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()), // not for prod
			grpc.WithBlock(), // ensures connection before invoking calls
			grpc.WithDisableRetry(),
		}
		conn *grpc.ClientConn
		err  error
		wg   sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		conn, err = grpc.DialContext(ctx, grpcServerAddr, opts...)
		if err != nil {
			l.Error("failed to connect to gRPC service", err.Error())
			return
		}

		wg.Done()
		defer conn.Close()
		<-ctx.Done() // keeps connection live without leaking when context cancelled
	}()

	wg.Wait() // ensures connection is live before use
	client := proto.NewPostbackStreamClient(conn)
	return &GrpcClient{
		client: client,
		consData: &proto.ConsumerData{
			StreamKey:  stream,
			GroupName:  group,
			ConsumerID: consumer,
		},
		conn: conn,
		log:  l,
	}
}

// thank you, stack overflow
func (gc *GrpcClient) waitUntilReady() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second) //define how long you want to wait for connection to be restored before giving up
	defer cancel()

	currentState := gc.conn.GetState()
	stillConnecting := true

	for currentState != connectivity.Ready && stillConnecting {
		//will return true when state has changed from thisState, false if timeout
		stillConnecting = gc.conn.WaitForStateChange(ctx, currentState)
		currentState = gc.conn.GetState()
		gc.log.Info(fmt.Sprintf("Attempting to reconnect, state has changed to: %+v", currentState))
	}

	if !stillConnecting {
		gc.log.Error("Connection attempt has timed out.")
		return false
	}

	return true
}
