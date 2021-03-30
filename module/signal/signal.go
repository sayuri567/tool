package signal

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sayuri567/gorun"
	"github.com/sayuri567/tool/module"
	"github.com/sirupsen/logrus"
)

type SignalModule struct {
	*module.DefaultModule

	signalHandles map[os.Signal]SignalHandle
	lock          sync.RWMutex
}

type SignalHandle func() error

var signalModule = &SignalModule{
	signalHandles: make(map[os.Signal]SignalHandle),
}

func GetSignalModule() *SignalModule {
	return signalModule
}

func SetHandle(handle SignalHandle, signals ...os.Signal) {
	signalModule.lock.Lock()
	for _, signal := range signals {
		signalModule.signalHandles[signal] = handle
	}
	signalModule.lock.Unlock()
}

func (this *SignalModule) Init() error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGSTOP, syscall.SIGUSR1, syscall.SIGUSR2)
	gorun.Go(this.checkSignal, sigs)
	return nil
}

func (this *SignalModule) checkSignal(sigs chan os.Signal) {
	for sig := range sigs {
		signalModule.lock.RLock()
		handle, ok := this.signalHandles[sig]
		signalModule.lock.RUnlock()
		if ok {
			err := handle()
			if err != nil {
				logrus.WithError(err).WithField("signal", sig).Error("signal handle error")
			}
		}
	}
}
