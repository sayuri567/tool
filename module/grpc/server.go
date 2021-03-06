package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/sayuri567/gorun"
	"github.com/sayuri567/tool/module"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// ServerConfig ServerConfig
type ServerConfig struct {
	ServerCertFile  string
	ServerKeyFile   string
	CaCertFile      string
	Port            int
	AccessLog       bool
	RegisterService func(*grpc.Server) error
	ServerOptions   []grpc.ServerOption
}

type ServerModule struct {
	*module.DefaultModule

	config *ServerConfig
	server *grpc.Server
}

var serverModule = New()

func GetServerModule() *ServerModule {
	return serverModule
}

func New() *ServerModule {
	return &ServerModule{}
}

func SetConfig(config *ServerConfig) {
	serverModule.SerConfig(config)
}
func (this *ServerModule) SerConfig(config *ServerConfig) {
	this.config = config
}

func (this *ServerModule) Init() error {
	if this.config == nil {
		return errors.New("grpc server config not set")
	}

	opts := []grpc.ServerOption{}
	if len(this.config.ServerCertFile) > 0 && len(this.config.ServerKeyFile) > 0 && len(this.config.CaCertFile) > 0 {
		creds, err := this.genCreds()
		if err != nil {
			return err
		}
		opts = append(opts, creds)
	}

	opts = append(opts, grpc.UnaryInterceptor(this.interceptor))

	if len(this.config.ServerOptions) > 0 {
		opts = append(opts, this.config.ServerOptions...)
	}

	this.server = grpc.NewServer(opts...)
	if err := this.config.RegisterService(this.server); err != nil {
		logrus.WithError(err).Error("failed to register service")
		return err
	}

	logrus.Info("grpcServer module inited")
	return nil
}

func (this *ServerModule) Run() error {
	addr := fmt.Sprintf(":%v", this.config.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logrus.WithError(err).Error("failed to listen port")
		return err
	}

	gorun.Go(this.server.Serve, lis)
	logrus.WithField("addr", addr).Info("grpcServer started")
	return nil
}

func (this *ServerModule) Stop() {
	logrus.Info("Stopping grpcServer")
	this.server.Stop()
	logrus.Info("Stopped grpcServer")
}

func (this *ServerModule) genCreds() (grpc.ServerOption, error) {
	cert, err := tls.LoadX509KeyPair(this.config.ServerCertFile, this.config.ServerKeyFile)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{"cert": this.config.ServerCertFile, "key": this.config.ServerKeyFile}).Error("failed to load cert file")
		return nil, err
	}

	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(this.config.CaCertFile)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{"cert": this.config.CaCertFile}).Error("failed to load ca file")
		return nil, err
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		err := errors.New("failed to append certs from pem")
		logrus.WithError(err).Error("failed to append certs from pem")
		return nil, err
	}

	c := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	})
	return grpc.Creds(c), nil
}

func (this *ServerModule) interceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	defer gorun.Recover("grpc panic")
	if this.config.AccessLog {
		logrus.WithFields(logrus.Fields{"method": info.FullMethod}).Info("grpc access log")
	}
	resp, err := handler(ctx, req)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{"method": info.FullMethod, "req": req}).Warn("grpc error")
	}
	return resp, err
}
