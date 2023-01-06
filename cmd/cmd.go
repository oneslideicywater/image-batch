package cmd

import (
	"fmt"
	"imagebatcher/registry"
	"log"
	"os"
	"strings"
)
import "github.com/docopt/docopt-go"

var usage = `image-batch
Usage:
  image-batch dump -f <filename> <tarfile>  
  image-batch load <tarfile>              
`

type Options struct {

}

func checkFileValid(opts docopt.Opts) bool{
	// check that the filename is not empty
	filename:=opts["<filename>"].(string)
	if strings.TrimSpace(filename) == "" {
		return false
	}
	return true
}


func Parse() {
	opts, _ := docopt.ParseArgs(usage,os.Args[1:],"v1.0")

	modified,err := registry.ConfirmDaemonJson()
	if err != nil {
		log.Fatal(err.Error())
	}
	if modified{
		fmt.Println("please restart the docker daemon,using `systemctl restart docker` to make modified content in `/etc/docker/daemon.json` take effect")
		return
	}

	// parse dump
	isDump:=opts["dump"].(bool)
	if isDump {

		tarfile:=strings.TrimSpace(opts["<tarfile>"].(string))
		if tarfile == ""{
			log.Fatal("tarfile can't be empty")
		}
		if !checkFileValid(opts){
			log.Fatal("filename can't be empty")
		}
		err:=BatchDump(opts["<filename>"].(string),tarfile)
		if err != nil {
			log.Fatal(err.Error())
		}
		return
	}
	// parse load
	isLoad:=opts["load"].(bool)
	if isLoad{
		tarfile:=strings.TrimSpace(opts["<tarfile>"].(string))
		if tarfile == ""{
			log.Fatal("tarfile can't be empty")
		}
		err:=BatchLoad(tarfile)
		if err != nil {
			log.Fatal(err.Error())
		}
	}


}

// BatchDump dump images in filename to tar.gz file specified by tarfile
// it implements function provided by `image-batch dump -f <filename> <tarfile>`
func BatchDump(filename string,tarfile string) error{

	// parse the image list
	list,err:=registry.ParseImagesFromFile(filename)
	if err != nil {
		return err
	}
	// transform the image tag from registry.xx.com/repo/artifact:tag => localhost:5000/artifact:tag
	tagFromRemoteToLocal, err:=registry.TransformImageTag(list)
	if err != nil {
		return err
	}

	pd:=registry.NewDefaultParallelDocker(tagFromRemoteToLocal,true)
	// pull the images
	err=pd.PullImages(true)
	if err != nil {
		return err
	}
	// retag the images
	err=pd.RetagImages(true)
	if err != nil {
		return err
	}

	opts:=registry.NewDefaultOptions()
	reg:=registry.NewDefaultRegistryWithImagesPredefined(tagFromRemoteToLocal,opts...)


	// dump the compress tag.gz file
	err=reg.Dump(tarfile)
	if err != nil {
		return err
	}

	return nil
}


// BatchLoad load images specified by tarfile
// it implements function provided by `image-batch load <tarfile>`
func BatchLoad(tarFile string) error{

	opts:=registry.NewDefaultOptions()
	reg:=registry.NewDefaultRegistry(opts...)
	err:=reg.Load(tarFile)
	if err != nil {
		return err
	}
	return nil
}