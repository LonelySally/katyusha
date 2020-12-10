package krpc

import (
	"fmt"
	"github.com/gogf/gf/frame/g"
	"github.com/gogf/gf/net/gipv4"
	"github.com/gogf/gf/os/gcmd"
	"github.com/gogf/gf/os/glog"
	"github.com/gogf/gf/os/gproc"
	"github.com/gogf/gf/text/gstr"
	"github.com/gogf/katyusha/discovery"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// GrpcServer is the server for GRPC protocol.
type GrpcServer struct {
	Server    *grpc.Server
	Logger    *glog.Logger
	config    *GrpcServerConfig
	services  []*discovery.Service
	waitGroup sync.WaitGroup
}

// NewGrpcServer creates and returns a grpc server.
func (s *krpcServer) NewGrpcServer(conf ...*GrpcServerConfig) *GrpcServer {
	var config *GrpcServerConfig
	if len(conf) > 0 {
		config = conf[0]
	} else {
		config = s.NewGrpcServerConfig()
	}
	if config.Address == "" {
		g.Log().Fatal("server address cannot be empty")
	}
	if !gstr.Contains(config.Address, ":") {
		g.Log().Fatal("invalid service address, should contain listening port")
	}
	if config.Logger == nil {
		config.Logger = glog.New()
	}
	server := &GrpcServer{
		Logger: config.Logger,
		config: config,
	}
	server.config.Options = append([]grpc.ServerOption{
		s.ChainUnary(
			server.UnaryError,
			server.UnaryLogger,
			server.UnaryRecover,
		),
	}, server.config.Options...)
	server.Server = grpc.NewServer(server.config.Options...)
	return server
}

// Service binds service list to current server.
// Server will automatically register the service list after it starts.
func (s *GrpcServer) Service(services ...*discovery.Service) {
	var (
		serviceAddress string
		array          = gstr.Split(s.config.Address, ":")
	)
	if array[0] == "0.0.0.0" || array[0] == "" {
		intraIp, err := gipv4.GetIntranetIp()
		if err != nil {
			s.Logger.Fatal("retrieving intranet ip failed, please check your net card or manually assign the service address: " + err.Error())
		}
		serviceAddress = fmt.Sprintf(`%s:%s`, intraIp, array[1])
	} else {
		serviceAddress = s.config.Address
	}
	for _, service := range services {
		if service.Address == "" {
			service.Address = serviceAddress
		}
	}
	s.services = services
}

// Run starts the server in blocking way.
func (s *GrpcServer) Run() {
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		s.Logger.Fatal(err)
	}
	if len(s.services) == 0 {
		appId := gcmd.GetWithEnv(discovery.EnvKeyAppId).String()
		if appId != "" {
			// Automatically creating service if app id can be retrieved
			// from environment or command-line.
			s.Service(&discovery.Service{
				AppId: appId,
			})
		}
	}
	// Start listening.
	go func() {
		if err := s.Server.Serve(listener); err != nil {
			s.Logger.Fatal(err)
		}
	}()

	// Register service list after server starts.
	for _, service := range s.services {
		if err = discovery.Register(service); err != nil {
			s.Logger.Fatal(err)
		}
	}

	s.Logger.Printf("grpc server start listening on: %s, pid: %d", s.config.Address, gproc.Pid())

	// Signal listening and handling for gracefully shutdown.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(
		sigChan,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGKILL,
		syscall.SIGTERM,
		syscall.SIGABRT,
	)
	for {
		switch <-sigChan {
		case
			syscall.SIGINT,
			syscall.SIGQUIT,
			syscall.SIGKILL,
			syscall.SIGTERM,
			syscall.SIGABRT:
			s.Logger.Print("gracefully shutting down")
			for _, service := range s.services {
				discovery.Unregister(service)
			}
			time.Sleep(time.Second)
			s.Stop()
			return
		default:
		}
	}
}

// Start starts the server in no-blocking way.
func (s *GrpcServer) Start() {
	s.waitGroup.Add(1)
	go func() {
		defer s.waitGroup.Done()
		s.Run()
	}()
}

// Wait works with Start, which blocks current goroutine until the server stops.
func (s *GrpcServer) Wait() {
	s.waitGroup.Wait()
}

// Stop gracefully stops the server.
func (s *GrpcServer) Stop() {
	s.Server.GracefulStop()
}

// ChainUnary returns a ServerOption that specifies the chained interceptor
// for unary RPCs. The first interceptor will be the outer most,
// while the last interceptor will be the inner most wrapper around the real call.
// All unary interceptors added by this method will be chained.
func (s *krpcServer) ChainUnary(interceptors ...grpc.UnaryServerInterceptor) grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(interceptors...)
}

// ChainStream returns a ServerOption that specifies the chained interceptor
// for stream RPCs. The first interceptor will be the outer most,
// while the last interceptor will be the inner most wrapper around the real call.
// All stream interceptors added by this method will be chained.
func (s *krpcServer) ChainStream(interceptors ...grpc.StreamServerInterceptor) grpc.ServerOption {
	return grpc.ChainStreamInterceptor(interceptors...)
}
