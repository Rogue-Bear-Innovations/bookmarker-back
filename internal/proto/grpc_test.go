package proto

import (
	"context"
	"fmt"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	app := fx.New(
		Module,
		fx.Provide(
			func() (*zap.SugaredLogger, error) {
				l, err := zap.NewDevelopment()
				if err != nil {
					return nil, err
				}

				s := l.Sugar()
				return s, nil
			},
		),
		fx.Invoke(func(server *BookmarkerServerImpl) {

		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	go func() {
		err := app.Start(ctx)
		fmt.Printf("app finished with %v\n", err)
	}()

	<-time.After(time.Second)

	var conn *grpc.ClientConn
	conn, err := grpc.Dial(":9000", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()

	c := NewBookmarkerClient(conn)

	response, err := c.GetBookmarks(context.Background(), &GetBookmarksRequest{})
	if err != nil {
		log.Fatalf("Error when calling SayHello: %s", err)
	}
	log.Printf("Response from server: %s", response)
}
