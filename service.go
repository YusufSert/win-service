package win_service

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

// todo: check how Viper works with logger or without logger
type WinService struct {
	r      Runable
	stopCh chan struct{}
	logger *eventlog.Log
	name   string
}

const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue

func NewWinService(r Runable, name string) (*WinService, error) {
	l, err := eventlog.Open(name)
	if err != nil {
		return nil, err
	}

	return &WinService{
		r:      r,
		stopCh: make(chan struct{}),
		logger: l,
		name:   name,
	}, nil
}

func (s *WinService) Execute(args []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}

	status <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	var ctx context.Context
	var cancel context.CancelFunc
	start := func() {
		ctx, cancel = context.WithCancel(context.Background())
		go s.runContext(ctx)
	}
	start()
	s.logger.Info(0, fmt.Sprintf("service %s running", s.name))

	defer cancel()

loop:
	for {
		select {
		case <-s.stopCh:
			break loop
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				s.logger.Info(0, "interrogate signal received")
				status <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				s.logger.Info(0, "stop|shutdown signal received")
				break loop
			case svc.Pause:
				s.logger.Info(0, "pause signal received")
				cancel()
				status <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				s.logger.Info(0, "continue signal received")
				start()
				status <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				s.logger.Warning(0, "unexpected signal received from scm")
			}
		}
	}

	status <- svc.Status{State: svc.StopPending}
	status <- svc.Status{State: svc.Stopped}

	s.logger.Info(0, fmt.Sprintf("service %s stopped", s.name))

	return false, 0
}

func (s *WinService) runContext(ctx context.Context) {
	err := s.r.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		close(s.stopCh)
		s.logger.Error(1, fmt.Sprintf("service %s stopped due to %s", s.name, err.Error()))
		return
	}
	s.logger.Warning(1, fmt.Sprintf("service %s canceled due to %s", s.name, err.Error()))
}

type Runable interface {
	Run(context.Context) error
}

// cmd to force a service to stop
// taskkill /F /PID <Service PID>
//sc config testService7 start= "auto" https://learn.microsoft.com/en-us/previous-versions/windows/it-pro/windows-server-2012-r2-and-2012/cc990290(v=ws.11)
