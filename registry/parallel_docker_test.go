package registry

import "testing"

func TestParallelDocker_PullImages(t *testing.T) {

	m:=map[string]string{
		"busybox":"",
		"nginx":"",
		"alpine:latest":"",
	}
	multiThreadPuller:=NewDefaultParallelDocker(m,false)
	err:=multiThreadPuller.PullImages(true)
	if err != nil {
		t.Error("err:",err.Error())
	}
}
func TestParallelDocker_PushImages(t *testing.T) {

	m:=map[string]string{
		"busybox":"localhost:5000/busybox",
		"nginx":"nginx",
		"alpine:latest":"alpine:latest",
	}
	multiThreadPuller:=NewDefaultParallelDocker(m,true)
	err:=multiThreadPuller.PushImages()
	if err != nil {
		t.Error("err:",err.Error())
	}
}

func TestParallelDocker_RetagImages(t *testing.T) {

	m:=map[string]string{
		"busybox":"localhost:5000/busybox",
		"nginx":"localhost:5000/nginx",
		"alpine:latest":"localhost:5000/alpine:latest",
	}
	multiThreadPuller:=NewDefaultParallelDocker(m,true)
	err:=multiThreadPuller.RetagImages(true)
	if err != nil {
		t.Error("err:",err.Error())
	}
}