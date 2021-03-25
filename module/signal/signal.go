package signal

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sayuri567/gorun"
	"github.com/sayuri567/tool/module"
	"github.com/sirupsen/logrus"
)

type SignalModule struct {
	*module.DefaultModule

	signalHandles map[os.Signal]SignalHandle
	inited        bool
}

type SignalHandle func() error

var signalModule = &SignalModule{
	signalHandles: make(map[os.Signal]SignalHandle),
}

func GetSignalModule() *SignalModule {
	return signalModule
}

func SetHandle(handle SignalHandle, signals ...os.Signal) {
	if signalModule.inited {
		return
	}
	for _, signal := range signals {
		signalModule.signalHandles[signal] = handle
	}
}

func (this *SignalModule) Init() error {
	this.inited = true
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGSTOP, syscall.SIGUSR1, syscall.SIGUSR2)
	gorun.Go(this.checkSignal, sigs)
	return nil
}

func (this *SignalModule) checkSignal(sigs chan os.Signal) {
	for {
		sig := <-sigs
		handle, ok := this.signalHandles[sig]
		if ok {
			err := handle()
			if err != nil {
				logrus.WithError(err).WithField("signal", sig).Error("signal handle error")
			}
		}
	}
}
