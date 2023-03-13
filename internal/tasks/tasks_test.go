package tasks_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/assert"
	"github.com/scaleway/scaleway-cli/v2/internal/tasks"
)

func TestGeneric(t *testing.T) {
	ts := tasks.Begin()

	tasks.Add(ts, "convert int to string", func(t *tasks.Task, args int) (nextArgs string, err error) {
		return fmt.Sprintf("%d", args), nil
	})
	tasks.Add(ts, "convert string to int and divide by 4", func(t *tasks.Task, args string) (nextArgs int, err error) {
		i, err := strconv.ParseInt(args, 10, 32)
		if err != nil {
			return 0, err
		}
		return int(i) / 4, nil
	})

	res, err := ts.Execute(context.Background(), 12)
	assert.Nil(t, err)
	assert.Equal(t, 3, res)
}

func TestInvalidGeneric(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	ts := tasks.Begin()

	tasks.Add(ts, "convert int to string", func(t *tasks.Task, args int) (nextArgs string, err error) {
		return fmt.Sprintf("%d", args), nil
	})
	tasks.Add(ts, "divide by 4", func(t *tasks.Task, args int) (nextArgs int, err error) {
		return args / 4, nil
	})
}

func TestCleanup(t *testing.T) {
	ts := tasks.Begin()

	clean := 0

	tasks.Add(ts, "TaskFunc 1", func(task *tasks.Task, args interface{}) (nextArgs interface{}, err error) {
		task.AddToCleanUp(func(ctx context.Context) error {
			clean++
			return nil
		})
		return nil, nil
	})
	tasks.Add(ts, "TaskFunc 2", func(task *tasks.Task, args interface{}) (nextArgs interface{}, err error) {
		task.AddToCleanUp(func(ctx context.Context) error {
			clean++
			return nil
		})
		return nil, nil
	})
	tasks.Add(ts, "TaskFunc 3", func(task *tasks.Task, args interface{}) (nextArgs interface{}, err error) {
		task.AddToCleanUp(func(ctx context.Context) error {
			clean++
			return nil
		})
		return nil, fmt.Errorf("fail")
	})

	_, err := ts.Execute(context.Background(), nil)
	assert.NotNil(t, err, "Execute should return error after cleanup")
	assert.Equal(t, clean, 2, "2 task cleanup should have been executed")
}

func TestCleanupOnContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Cannot send signal on windows")
	}
	ts := tasks.Begin()

	clean := 0
	ctx := context.Background()

	tasks.Add(ts, "TaskFunc 1", func(task *tasks.Task, args interface{}) (nextArgs interface{}, err error) {
		task.AddToCleanUp(func(ctx context.Context) error {
			clean++
			return nil
		})
		return nil, nil
	})
	tasks.Add(ts, "TaskFunc 2", func(task *tasks.Task, args interface{}) (nextArgs interface{}, err error) {
		task.AddToCleanUp(func(ctx context.Context) error {
			clean++
			return nil
		})
		return nil, nil
	})
	tasks.Add(ts, "TaskFunc 3", func(task *tasks.Task, args interface{}) (nextArgs interface{}, err error) {
		task.AddToCleanUp(func(ctx context.Context) error {
			clean++
			return nil
		})
		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			return nil, err
		}

		// Interrupt tasks, as done with Ctrl-C
		err = p.Signal(os.Interrupt)
		if err != nil {
			t.Fatal(err)
		}

		select {
		case <-task.Ctx.Done():
			return nil, fmt.Errorf("interrupted")
		case <-time.After(time.Second * 3):
			return nil, nil
		}
	})

	_, err := ts.Execute(ctx, nil)
	assert.NotNil(t, err, "context should have been interrupted")
	assert.True(t, strings.Contains(err.Error(), "interrupted"), "error is not interrupted: %s", err)
	assert.Equal(t, clean, 2, "2 task cleanup should have been executed")
}
