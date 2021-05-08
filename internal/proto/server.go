package proto

import (
	"context"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"log"
	"net"
)

type BookmarkerServerImpl struct {
	UnimplementedBookmarkerServer
}

func NewGRPCServer(lc fx.Lifecycle, logger *zap.SugaredLogger) *BookmarkerServerImpl {
	instance := BookmarkerServerImpl{}

	grpcServer := grpc.NewServer()

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				lis, err := net.Listen("tcp", ":9000")
				if err != nil {
					log.Fatalf("failed to listen: %v", err)
				}

				RegisterBookmarkerServer(grpcServer, &instance)

				if err := grpcServer.Serve(lis); err != nil {
					log.Fatalf("failed to serve: %s", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping GRPC server.")
			grpcServer.GracefulStop()
			return nil
		},
	})

	return &instance
}

func (s *BookmarkerServerImpl) GetBookmarks(ctx context.Context, request *GetBookmarksRequest) (*GetBookmarksResponse, error) {
	name := "name"
	b := Bookmark{
		Id:          1,
		Name:        &name,
		Link:        nil,
		Description: nil,
	}
	return &GetBookmarksResponse{
		Items: []*Bookmark{
			&b,
		},
	}, nil
}
