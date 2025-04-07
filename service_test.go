package win_service

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/windows/svc"
)

func TestService(t *testing.T) {
	w, err := NewWinService(&testService{}, "testService2")
	if err != nil {
		log.Fatal("err")
	}

	err = svc.Run("testService2", w)
}

type testService struct {
}

func (s *testService) Run(ctx context.Context) error {
	f, err := os.OpenFile("C:/Users/yusuf/OneDrive/Masaüstü/test_serbice/test11.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	//f, err := os.OpenFile("./logt.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln(fmt.Errorf("error opening file: %v", err))
	}

	l := slog.New(slog.NewJSONHandler(f, nil))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
			l.Info("kudimmm")
		}
	}
}
