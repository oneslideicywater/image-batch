package registry

import (
	"fmt"
	"testing"
	"time"
)

func TestNewDefaultQueue(t *testing.T) {

	queue:=NewDefaultQueue()
	count:=1

	go func() {
		for{
			count++
			err := queue.Enqueue(count)
			time.Sleep(1 * time.Second)
			if err != nil {
				t.Error(err.Error())
			}
		}

	}()

	go func() {
		for{
			ret:=queue.Dequeue()
			if ret == nil {
				time.Sleep(1 * time.Second)
			}
			fmt.Println(ret)
		}

	}()
	go func() {
		for{
			ret:=queue.Dequeue()
			if ret == nil {
				time.Sleep(1 * time.Second)
			}
			fmt.Println(ret)
		}

	}()
	time.Sleep(10 * time.Second)
}
