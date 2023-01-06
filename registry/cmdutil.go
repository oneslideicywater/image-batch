package registry

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	paths "path"
	"strings"
)

// simplify cmd.go exc


func BashCommandExec(args... string) (string,error){
	cmdRun:=exec.Command("/bin/bash","-c",strings.Join(args," "))
	output,err:=cmdRun.CombinedOutput()
	if err != nil {
		return "",fmt.Errorf("exit msg: %s, "+ string(output), err.Error())
	}
	return string(output), nil
}
// ExecAndSignal useful in async task, this function put finished indicates finish. put error into e
func ExecAndSignal(cmd []string,finished chan struct{},e chan error){
	output,err:= BashCommandExec(cmd...)
	if err != nil {
		finished<- struct{}{}
		e<-err
		return
	}
	log.Printf("%s done \n",strings.Join(cmd," "))
	log.Println(output)
	finished<- struct{}{}
	e<-nil
}

func DockerCmd(args... string) ([]string,error){
	path,err:=FindBestBinary("")
	if err != nil {
		return []string{},err
	}
	cmd:=[]string{path}
	cmd=append(cmd,args...)
	return cmd,nil
}




func ErrorWithStderr(output string,err error) error{
	return fmt.Errorf("exit msg: %s, "+ string(output), err.Error())
}

// ParseImagesFromFile parse image from file, with one remote image one line
func ParseImagesFromFile(path string) ([]string,error){
	// pull the image from running registry
	content,err:=os.ReadFile(path)
	if err != nil {
		return []string{},err
	}
	strContent:=string(content)
	list:=strings.Split(strContent,"\n")

	ret:=make([]string,0)
	for _, e := range list {
		if strings.TrimSpace(e) == "" {
			continue
		}
		ret=append(ret,strings.TrimSpace(e))
	}
	fmt.Println(strings.Join(ret,","))
	return ret,nil
}

// TransformImageTag transform image tag from registry.xx.com/repo/artifact:tag => localhost:5000/artifact:tag
func TransformImageTag(images []string) (map[string]string,error){
	localReg:="localhost:5000/"
	ret:=make(map[string]string)
	for _,image:=range images{
		tmpList:=strings.Split(image,"/")
		baseRaw:=tmpList[len(tmpList)-1]
		base:=strings.TrimSpace(baseRaw)
		if base == ""{
			return map[string]string{},fmt.Errorf("wrong image format %s",image)
		}
		//such as localhost:5000/example:v2
		base=localReg+base
		ret[image]=base
	}
	return ret,nil
}


func RetagOne(kv string) error{
	cmd,err:=DockerCmd("tag",kv)
	if err != nil {
		return err
	}
	output,err:=BashCommandExec(cmd...)
	if err != nil {
		return ErrorWithStderr(output,err)
	}
	fmt.Printf("%s \n",strings.Join(cmd," "))
	return nil
}

// LoadRegistryV2DockerImage load registry:2 docker offline images
func LoadRegistryV2DockerImage(path string) error{
	loadCmd, _ :=DockerCmd("load","-i",path)
	output,err:=BashCommandExec(loadCmd...)
	if err != nil {
		return ErrorWithStderr(output,err)
	}
	return nil
}
// SaveRegistryV2DockerImage docker save registry:2 > ${path}/registry.tar
func SaveRegistryV2DockerImage(path string) error{
	log.Printf("docker save registry:2 > %s/registry.tar \n", path)
	cmd, err := DockerCmd("save", "registry:2", ">", paths.Join(path,OFFLINE_IMAGE_NAME_OF_REGISTRY_V2))
	if err != nil {
		return err
	}
	output,err:=BashCommandExec(cmd...)
	if err != nil {
		return ErrorWithStderr(output,err)
	}
	return nil
}



// TarExtractFrom extract tar.gz from specified target to current dir
func TarExtractFrom(target string) error{
	cmd:=[]string{"tar","--strip-components=2","-xzf",target}
	output,err:=BashCommandExec(cmd...)
	if err != nil {
		return ErrorWithStderr(output,err)
	}
	return nil
}
// TarCompressTo compress `source` into dst, in tar.gz format
func TarCompressTo(dst string, source string) error{
	log.Printf("compressing the dump files in a whole piece from %s to: %s \n",source,dst)
	cmd2:=[]string{"tar","czf",dst,source}
	output,err:=BashCommandExec(cmd2...)
	if err != nil {
		return ErrorWithStderr(output,err)
	}
	return nil
}