package win_service

import (
	"context"
	"errors"

	"golang.org/x/sys/windows/svc"
)

// todo: check how Viper works with logger or without logger
type WinService struct {
	r      Runable
	stopCh chan struct{}
}

const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue

func NewWinService(r Runable, name string) (*WinService, error) {

	return &WinService{
		r:      r,
		stopCh: make(chan struct{}),
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

	defer cancel()

loop:
	for {
		select {
		case <-s.stopCh:
			break loop
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				status <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			case svc.Pause:
				cancel()
				status <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				start()
				status <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				//panic("unexpected signal from scm")
			}
		}
	}

	status <- svc.Status{State: svc.StopPending}
	status <- svc.Status{State: svc.Stopped}
	return false, 0
}

func (s *WinService) runContext(ctx context.Context) {
	err := s.r.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		close(s.stopCh)
	}
}

type Runable interface {
	Run(context.Context) error
}

// cmd to force a service to stop
// taskkill /F /PID <Service PID>
//sc config testService7 start= "auto" https://learn.microsoft.com/en-us/previous-versions/windows/it-pro/windows-server-2012-r2-and-2012/cc990290(v=ws.11)
