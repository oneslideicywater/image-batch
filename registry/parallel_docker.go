package registry

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)
var (
	PULL_TIMEOUT_MINUTE = 10 * time.Minute
)
// this section implements a multi-go-routine image puller base on single routine Puller impl

type ParallelDocker struct {
	// Parallelism indicates the number of go routine,default to the number of cpu core
	Parallelism int

	// Images record the images list or images pair.
	// the relationship of k and v is usually remote-local (e.g. example.com/busybox:v1 => localhost:5000/busybox:v1)
	Images map[string]string

	//if PairMode is false, only key of Images take effect.
	PairMode bool

	Puller Puller

	Pusher Pusher

	workqueue Queue
}
// PullImages pull all images in a multi-go-routine. each go routine fetch one item at a time from work queue
func(p *ParallelDocker) PullImages(k bool) error{
	var wg sync.WaitGroup
	wg.Add(len(p.Images))

	if k {
		// add all images to work queue
		p.loadQueue(true,false,false,false)
	}else {
		p.loadQueue(false,true,false,false)
	}


	errResult := make([]error,0)
	errResultLock:=sync.Mutex{}


	imagePull:= func() {
		// forever fetch a new image to pull,util workqueue return nil
		for{

			ctx,cancel:=context.WithTimeout(context.Background(),PULL_TIMEOUT_MINUTE)
			defer cancel()
			imageUntyped :=p.workqueue.Dequeue()
			if imageUntyped == nil{
				return
			}
			image:=imageUntyped.(string)
			fmt.Printf("pulling image %s \n",image)
			err := p.Puller.Pull(ctx, image)
			if err != nil {
				fmt.Printf("image %s pulling failed. \n",image)
				errResultLock.Lock()
				errResult=append(errResult,err) // mark the errResult
				errResultLock.Unlock()
				wg.Done() // mark it as finished, even the result is failed
				return
			}
			wg.Done() // mark it as success
		}
	}
	// pull the images
	p.parallelRun(imagePull)
	wg.Wait()

	// check err
	if len(errResult) !=0{
		return SummaryError(errResult)
	}
	return nil
}

func(p *ParallelDocker) PushImages() error{
	if !p.PairMode {
		return fmt.Errorf("you can't push with pair mode disabled")
	}
	var wg sync.WaitGroup
	wg.Add(len(p.Images))

	// add all images to work queue
	p.loadQueue(false,true,false,false)

	errResult := make([]error,0)
	errResultLock:=sync.Mutex{}


	imagePush:= func() {
		// forever fetch a new image to pull,util workqueue return nil
		for{
			ctx,cancel:=context.WithTimeout(context.Background(),PULL_TIMEOUT_MINUTE)
			defer cancel()
			imageUntyped :=p.workqueue.Dequeue()
			if imageUntyped == nil{
				return
			}
			image:=imageUntyped.(string)
			err := p.Pusher.Push(ctx, image)
			if err != nil {
				fmt.Printf("image %s push failed. \n",image)
				errResultLock.Lock()
				errResult=append(errResult,err) // mark the errResult
				errResultLock.Unlock()
				wg.Done() // mark it as finished, even the result is failed
				return
			}
			wg.Done() // mark it as success
		}
	}
	// push the images

	p.parallelRun(imagePush)
	wg.Wait()

	// check err
	if len(errResult) !=0{
		return SummaryError(errResult)
	}

	return nil
}

func(p *ParallelDocker) RetagImages(kTov bool) error{
	var wg sync.WaitGroup
	wg.Add(len(p.Images))

	// add all images to work queue
	if kTov {
		p.loadQueue(false,false,true,false)
	}else{
		p.loadQueue(false,false,false,true)
	}

	errResult := make([]error,0)
	errResultLock:=sync.Mutex{}


	imageRetag:= func() {
		// forever fetch a new image to pull,util workqueue return nil
		for{
			imageUntyped :=p.workqueue.Dequeue()
			if imageUntyped == nil{
				return
			}
			image:=imageUntyped.(string)
			err := RetagOne(image)
			if err != nil {
				fmt.Printf("image %s retag failed. \n",image)
				errResultLock.Lock()
				errResult=append(errResult,err) // mark the errResult
				errResultLock.Unlock()
				wg.Done() // mark it as finished, even the result is failed
				return
			}
			wg.Done() // mark it as success
		}
	}
	// retag the images
	p.parallelRun(imageRetag)
	wg.Wait()

	// check err
	if len(errResult) !=0{
		return SummaryError(errResult)
	}

	return nil
}


func(p *ParallelDocker) loadQueue(k bool,v bool, kv bool, vk bool){
	if k {
		// add all images to work queue
		for k, _:=range p.Images{
			// not handle err, cause map can guard type assertion
			_ = p.workqueue.Enqueue(k)
		}
		return
	}
	if v{
		// add all images to work queue
		for _, v:=range p.Images{
			// not handle err, cause map can guard type assertion
			_ = p.workqueue.Enqueue(v)
		}
		return
	}

	if kv || vk{
		// add all images to work queue
		for k, v:=range p.Images{
			if kv{
				// not handle err, cause map can guard type assertion
				_ = p.workqueue.Enqueue(fmt.Sprintf("%s %s",k,v))
				continue
			}
			if vk{
				// not handle err, cause map can guard type assertion
				_ = p.workqueue.Enqueue(fmt.Sprintf("%s %s",v,k))
				continue
			}
		}
		return
	}


}
func(p *ParallelDocker) parallelRun(task func()){
	for i := 0; i < p.Parallelism; i++ {
		go task()
	}
}


func SummaryError(errs []error) error {
	list:=make([]string,0)
	for _,err:=range errs{
		list=append(list,err.Error())
	}
	return fmt.Errorf("[%s]",strings.Join(list,","))
}


func NewDefaultParallelDocker(images map[string]string,pairMode bool) *ParallelDocker {
	return &ParallelDocker{
		Images: images,
		Parallelism: runtime.NumCPU() ,
		PairMode: pairMode,
		Puller: NewDefaultPuller(),
		Pusher: NewDefaultPusher(),
		workqueue: NewDefaultQueue(),
	}
}