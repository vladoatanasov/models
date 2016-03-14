package grpcUpdatablePositions

import (
	"errors"
	"log"
	"net"
	"strings"

	"google.golang.org/grpc"

	"golang.org/x/net/context"

	"github.com/coreos/go-etcd/etcd"
	"github.com/squirrel-land/models/mobilityManagers/grpcUpdatablePositions/pb"
	"github.com/squirrel-land/squirrel"
)

type grpcUpdatablePositions struct {
	pm  squirrel.PositionManager
	lis net.Listener

	empty *pb.Empty
}

func NewGRPCUpdatablePositions() squirrel.MobilityManager {
	return &grpcUpdatablePositions{empty: new(pb.Empty)}
}

func (m *grpcUpdatablePositions) ParametersHelp() string {
	return `gRPCUpdatablePositions is a mobility manager that serves a gRPC service, through which another process can update nodes' positions.

  "address":  string, required;
						a TCP address that gRPC service should listen on. e.g. ":1234"
    `
}

func (m *grpcUpdatablePositions) Configure(conf *etcd.Node) (err error) {
	if conf == nil {
		err = errors.New("grpcUpdatablePositions: conf (*etcd.Node) is nil")
		return
	}

	var laddr string

	for _, node := range conf.Nodes {
		if !node.Dir && strings.HasSuffix(node.Key, "/address") {
			laddr = node.Value
		}
	}

	if laddr == "" {
		err = errors.New("address is missing from config")
		return
	}

	m.lis, err = net.Listen("tcp", laddr)
	if err != nil {
		return
	}

	return
}

func (m *grpcUpdatablePositions) Initialize(positionManager squirrel.PositionManager) {
	m.pm = positionManager
	gs := grpc.NewServer()
	pb.RegisterPositionServiceServer(gs, m)
	go func() {
		if err := gs.Serve(m.lis); err != nil {
			log.Fatalf("initializing gRPC server error: %s", err.Error())
		}
	}()
}

func (m *grpcUpdatablePositions) GetPosition(ctx context.Context, req *pb.GetPositionRequest) (pos *pb.Position, err error) {
	var p squirrel.Position
	p, err = m.pm.GetAddr(req.HardwareAddress)
	if err != nil {
		return
	}
	pos = &pb.Position{X: p.X, Y: p.Y, H: p.Height}
	return
}

func (m *grpcUpdatablePositions) SetPosition(ctx context.Context, req *pb.SetPositionRequest) (empty *pb.Empty, err error) {
	if er := m.pm.SetAddr(req.HardwareAddress, req.Position.X, req.Position.Y, req.Position.H); er != nil {
		log.Printf("setting position for %s error: %s", req.HardwareAddress, err.Error())
	}
	empty = m.empty
	return
}
