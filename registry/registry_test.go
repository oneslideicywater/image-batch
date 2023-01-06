package registry

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestNewDefaultOptions(t *testing.T) {
	opts:=NewDefaultOptions()
	opt:=&Options{

	}
	opts[0](opt)
	fmt.Println(opt)
}

func TestNewDefaultRegistry(t *testing.T) {
	opts:=NewDefaultOptions()

	registry:=NewDefaultRegistry(opts...)

	err:=registry.Start()
	if err != nil {
		t.Error(err.Error())
	}
	output,err:=exec.Command("/bin/bash","-c","docker ps").CombinedOutput()
	if err != nil {
		t.Error(string(output))
	}
	fmt.Println(registry)
	t.Log(string(output))

}

func TestDockerPuller_Pull(t *testing.T) {
	puller:=NewDefaultPuller()
	ctx, cancel :=context.WithDeadline(context.Background(),time.Now().Add(1 * time.Hour))
	defer cancel()
	err:=puller.Pull(ctx,"busybox")
	if err != nil {
		t.Error(err.Error())
	}
}

func TestDockerPuller_CheckIfPresent(t *testing.T) {
	puller:=NewDefaultPuller()
	ok,err:=puller.CheckIfPresent("registry:3")
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Println(ok)
}

func TestPushImage(t *testing.T)  {
	opts:=NewDefaultOptions()
	registry:=NewDefaultRegistry(opts...)

	err := registry.Push("localhost:5000/tlayer","")
	if err != nil {
		t.Error(err.Error())
	}
}

func TestDump(t *testing.T){
	opts:=NewDefaultOptions()

	registry:=NewDefaultRegistry(opts...)

	err:=registry.Dump("/root/registry/images.tar.gz")
	if err != nil {
		t.Error(err.Error())
	}
}
func TestLoad(t *testing.T){
	opts:=NewDefaultOptions()

	registry:=NewDefaultRegistry(opts...)

	err:=registry.Load("/root/registry/images.tar.gz")
	if err != nil {
		t.Error(err.Error())
	}
}

func TestParseImages(t *testing.T){
	list,err:=ParseImagesFromFile("/root/GolandProjects/imagebatcher/imagelist")
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Println(strings.Join(list,","))
}

func TestTransformImageTag(t *testing.T) {

	list:=[]string{
		"registry.geoway.com/cicd/cicd-wiki:latest",
		"tlayer",
		"registry.geoway.com/cicd/jenkins:v1",

	}
	tList,err:=TransformImageTag(list)
	if err != nil {
		t.Error(err.Error())
	}
	for k, s := range tList {
		fmt.Println(k,s)
	}
}

func TestRegistry_IsHealth(t *testing.T) {
	opts:=NewDefaultOptions()
	registry:=NewDefaultRegistry(opts...)
	if registry.IsHealth(){
		fmt.Println("health")
	}else{
		fmt.Println("not health")
	}
}

func TestConfirmDaemonJson(t *testing.T) {
	modified,err := ConfirmDaemonJson()
	if err != nil {
		t.Error(err.Error())
		return
	}
	fmt.Println(modified)
}