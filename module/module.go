package module

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Module interface {
	Init() error
	Run() error
	Stop()
}

type DefaultModule struct {
}

func (this *DefaultModule) Init() error {
	return nil
}

func (this *DefaultModule) Run() error {
	return nil
}

func (this *DefaultModule) Stop() {

}

// DefaultModuleManager default module manager
type DefaultModuleManager struct {
	Module
	Modules []Module
}

func NewDefaultModuleManager() *DefaultModuleManager {
	return &DefaultModuleManager{
		Modules: make([]Module, 0, 5),
	}
}

func (this *DefaultModuleManager) Init() error {
	for i := 0; i < len(this.Modules); i++ {
		err := this.Modules[i].Init()
		if err != nil {
			return fmt.Errorf("DefaultModuleManager:Init index:%d,module:%v,%v", i, this.Modules[i], err)
		}
	}
	return nil
}

func (this *DefaultModuleManager) Run() error {
	var err error
	for i := 0; i < len(this.Modules); i++ {
		err = this.Modules[i].Run()
		if err != nil {
			break
		}
	}

	return err
}

func (this *DefaultModuleManager) Stop() {
	var wg sync.WaitGroup
	for i := 0; i < len(this.Modules); i++ {
		wg.Add(1)
		go func(module Module) {
			module.Stop()
			wg.Done()
		}(this.Modules[i])
	}
	wg.Wait()
}

func (this *DefaultModuleManager) AppendModule(module Module) Module {
	this.Modules = append(this.Modules, module)
	return module
}

// WaitTerminateSignal wait signal to end the program
func WaitForTerminate() os.Signal {
	exitChan := make(chan struct{})
	signalChan := make(chan os.Signal, 1)
	var sign os.Signal
	go func() {
		sign = <-signalChan
		close(exitChan)
	}()
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-exitChan
	return sign
}
