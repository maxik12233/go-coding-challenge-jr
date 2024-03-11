package timer

import (
	"challenge/pkg/api/timercheck"
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

type Ping struct {
	TimerName   string
	SecondsLeft int
}

type Timer struct {
	timerChecker timercheck.TimerCheck
}

func NewTimer(timerChecker timercheck.TimerCheck) *Timer {
	return &Timer{
		timerChecker: timerChecker,
	}
}

// StartOrSubscribe creating new streaming channel which gets timer updates with given frequency
//
//	Streaming was created only if there was no errors in return
//
// You can use context cancellation function to interrupt streaming from outside
func (t *Timer) StartOrSubscribe(timerName string, timerSeconds int, freq int) (<-chan Ping, context.CancelFunc, error) {

	_, _, err := t.timerChecker.CheckTimer(timerName)
	if err != nil {
		if errors.Is(err, timercheck.ErrTimedOut) || errors.Is(err, timercheck.ErrNotExists) {
			log.Println("timer doesn't exist, creating new timer with name: " + timerName)
			if err := t.timerChecker.CreateTimer(timerName, timerSeconds); err != nil {
				return nil, nil, fmt.Errorf("%w: %v", err, "timer creation failed")
			}
		} else {
			return nil, nil, fmt.Errorf("%w: %v", err, "something wrong on foreign api side")
		}
	}

	ticker := time.NewTicker(time.Duration(freq) * time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	ping := make(chan Ping)
	go func() {
		defer func() {
			close(ping)
			ticker.Stop()
			log.Println("closing timer timer subscription goroutine")
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r, _, err := t.timerChecker.CheckTimer(timerName)
				if err != nil {
					if errors.Is(err, timercheck.ErrTimedOut) {
						return
					}
					log.Println("error when checking timer: ", err)
					return
				}
				ping <- Ping{
					TimerName:   timerName,
					SecondsLeft: r,
				}
			}
		}

	}()

	return ping, cancel, nil
}
