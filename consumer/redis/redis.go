package redis

// not used in current implementation - ported over to separate gRPC API service

import (
	"context"
	"delivery/logger"
	"delivery/types"
	"fmt"
	"time"

	"github.com/go-redis/redis/v9"
)

// configuration variables hardcoded for convenience
var (
	Stream              = "pb-stream"
	ConsGroup           = "pb-delivery"
	ConsumerID          = 1
	MsgQueryCount int64 = 100
	redisAddr           = "redis:6379"
	redisPW             = ""
	redisDB             = 0
)

// RedisConsumer implements MessageConsumer interface
type RedisConsumer struct {
	Client     *redis.Client
	Stream     string
	Group      string
	Consumer   string
	QueryCount int64
	Log        logger.AppLogger
}

// Calls redisMsgChan to instantiate a new channel of *redis.XMessage
// Pulls messages from that channel, translates each message, and pushes them
// onto the return channel for processing
func (r *RedisConsumer) NewMessageChannel(ctx context.Context) <-chan *types.Message {
	ch := make(chan *types.Message)
	rch := r.redisMsgChan(ctx)

	go func() {
		defer close(ch)
		for rm := range rch {
			m := redisMsgToMessage(rm)
			ch <- m
		}
	}()

	return ch
}

// executes redis XACK request for the given id and r's consumer data
func (r *RedisConsumer) AckMessages(ctx context.Context, ids <-chan interface{}) error {
	go func() {
		for id := range ids {
			idstr, _ := id.(string)
			ack := r.Client.XAck(ctx, r.Stream, r.Group, idstr)
			_, err := ack.Result()
			if err != nil {
				r.Log.Error("failed to ack message:", idstr)
			}
		}
	}()

	return nil
}

// instantiates a new Redis client to read from stream using embedded config variables
func New(l logger.AppLogger) *RedisConsumer { // Add logger to instantiate
	return &RedisConsumer{
		Client: redis.NewClient(&redis.Options{
			Addr:     redisAddr,
			Password: redisPW,
			DB:       redisDB,
		}),
		Stream:     Stream,
		Group:      ConsGroup,
		Consumer:   fmt.Sprintf("consumer-%d", ConsumerID),
		QueryCount: int64(MsgQueryCount),
		Log:        l,
	}
}

// rawMsgChan continually queries the stream and pushes each redis message
// to a buffered channel.
func (r *RedisConsumer) redisMsgChan(ctx context.Context) <-chan *redis.XMessage {
	mchan := make(chan *redis.XMessage, r.QueryCount)
	r.createGroup(ctx)

	go func() {
		defer close(mchan)
		for {
			select {

			case <-ctx.Done():
				r.Log.Info("context cancelled - closing redis stream channel")
				return

			default:
				messages, err := r.readStream(ctx) // blocks for
				if err != nil {
					r.Log.Error(fmt.Sprintf("failed to read stream: %v", err))
				}
				for _, msg := range messages {
					mchan <- &msg
				}
			}
		}
	}()

	return mchan
}

// parses redis stream message into consumer message
// no validation explicitly performed by this function (see parseMsgVal)
func redisMsgToMessage(m *redis.XMessage) *types.Message {
	method := parseMsgVal(m, "method")
	url := parseMsgVal(m, "url")

	return &types.Message{
		Method:    method,
		URL:       url,
		Timestamp: time.Time{},
		AckID:     nil,
	}
}

// Parses entry in stream message values map corresponding with the given key.
// A value that can't be parsed as a string is processed as the empty string
func parseMsgVal(m *redis.XMessage, key string) string {
	iface, ok := m.Values[key]
	if !ok {
		iface = ""
	}

	// val initialized to "" if type assertion fails
	val, _ := iface.(string)
	return val
}

// Performs single query of stream for the Consumer Group (the application) and Consumer (application instance)
// using Redis's `XReadGroup` command. Returns `r.QueryCount` messages at a time
// (Docs are unclear if XReadGroup is safe for concurrent use.)
func (r *RedisConsumer) readStream(ctx context.Context) ([]redis.XMessage, error) {
	queryArgs := r.xReadGrArgs(">") // ">" special ID to query undelivered messages
	query := r.Client.XReadGroup(ctx, queryArgs)

	streams, err := query.Result()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	for _, stream := range streams {
		return stream.Messages, nil
	}

	return nil, nil
}

func (r *RedisConsumer) createGroup(ctx context.Context) {
	rCmd := r.Client.XGroupCreate(context.Background(), Stream, r.Group, "0")
	result, err := rCmd.Result()
	r.Log.Error(fmt.Sprintln("group create", result, err))
}

// prebuilds arguments for XReadGroup query, starting after the given message ID in the stream
func (r *RedisConsumer) xReadGrArgs(queryStartID string) *redis.XReadGroupArgs {
	return &redis.XReadGroupArgs{
		Group:    r.Group,
		Consumer: r.Consumer,
		Streams:  []string{Stream, queryStartID},
		Count:    r.QueryCount,
		Block:    10000, // how long readStream
		NoAck:    false,
	}
}

func (r *RedisConsumer) claimIdleMsgs() ([]redis.XMessage, error) {
	return nil, nil
}
