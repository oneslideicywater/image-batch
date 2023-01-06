package registry

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

type Puller interface {
	Pull(ctx context.Context,image string) error

	CheckIfPresent(image string) (bool,error)
}

type Pusher interface {
	Push(ctx context.Context,image string) error
}


type docker struct {}

func (d docker) CheckIfPresent(image string) (bool,error) {
	path,err:=FindBestBinary("")
	if err != nil {
		return false,err
	}
	cmd:=[]string{path,"image","list","-q",image}
	output,err:= BashCommandExec(cmd...)
	if err != nil {
		return false,err
	}
	if strings.TrimSpace(output) == ""{
		return false,nil
	}
	log.Printf("image %s exist,image id: %s",image,strings.TrimSpace(output))
	return true,nil
}

func (d docker) Push(ctx context.Context,image string) error{
	path,err:=FindBestBinary("")
	if err != nil {
		return err
	}
	pushfinished:=make(chan struct{},1)
	errfinished:=make(chan error,1)
	log.Printf("push image %s ... \n",image)
	cmd:=[]string{path,"push",image}

	go ExecAndSignal(cmd,pushfinished,errfinished)

	for{
		select {
		case <-pushfinished:
			return <-errfinished
		case <-ctx.Done():
			return fmt.Errorf("push image %s exceed deadline",image)
		default:
			time.Sleep(1*time.Second)
		}
	}


}



func (d docker) Pull(ctx context.Context,image string) error {
	path,err:=FindBestBinary("")
	if err != nil {
		return err
	}
	pullfinished:=make(chan struct{},1)
	errfinished:=make(chan error,1)
	log.Printf("pulling image %s ... \n",image)
	cmd:=[]string{path,"pull",image}

	go ExecAndSignal(cmd,pullfinished,errfinished)

	for{
		select {
		case <-pullfinished:
			return <-errfinished
		case <-ctx.Done():
			return fmt.Errorf("pull image %s exceed deadline",image)
		default:
			time.Sleep(1*time.Second)
		}
	}

}




func NewDefaultPuller() Puller{
	return docker{}
}
func NewDefaultPusher() Pusher{
	return docker{}
}

