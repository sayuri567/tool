package signal

import (
	"os"
	"os/signal"

	"github.com/sayuri567/gorun"
	"github.com/sayuri567/tool/module"
	"github.com/sirupsen/logrus"
)

type SignalModule struct {
	*module.DefaultModule

	signalHandles map[os.Signal]SignalHandle
	sigs          []os.Signal
	inited        bool
}

type SignalHandle func() error

var signalModule = &SignalModule{
	signalHandles: make(map[os.Signal]SignalHandle),
	sigs:          make([]os.Signal, 0),
}

func GetSignalModule() *SignalModule {
	return signalModule
}

func SetHandle(handle SignalHandle, signals ...os.Signal) {
	if signalModule.inited {
		panic("signal module has been inited")
	}
	for _, signal := range signals {
		signalModule.signalHandles[signal] = handle
		signalModule.sigs = append(signalModule.sigs, signal)
	}
}

func (this *SignalModule) Init() error {
	this.inited = true
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, this.sigs...)
	gorun.Go(this.checkSignal, sigs)
	return nil
}

func (this *SignalModule) checkSignal(sigs chan os.Signal) {
	for sig := range sigs {
		handle, ok := this.signalHandles[sig]
		if ok {
			err := handle()
			if err != nil {
				logrus.WithError(err).WithField("signal", sig).Error("signal handle error")
			}
		}
	}
}
