package registry

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	paths "path"
	"strings"
	"time"
)

var (
	DEFAULT_CRI_BINARY = "docker"

	PULL_POLICY_IFNOTPRESENT="IfNotPresent"
	PULL_POLICY_ALWAYS="always"

	// offline image tar name of registry:2
	OFFLINE_IMAGE_NAME_OF_REGISTRY_V2="registry-v2.tar"
)



// Options is the configuration of registry
type Options struct {
	// Image registry image tag, default to "registry:2"
	Image string

	// DataPath is data dir in host path, which map to `/var/lib/registry` in the container
	// which is the data dir of docker image registry
	DataPath string

	// Container Path is the data dir of registry in container
	ContainerPath string

	// HostPort is port of host
	HostPort int

	// ContainerPort is port of registry container
	ContainerPort int

	// Container Name
	ContainerName string

	// PullPolicy is pull policy of the "registry:2" image
	PullPolicy string
}



// Opt apply changes to Options
type Opt func(opt *Options)

// Registry has ability to push images, dump to a tar file in a whole piece
type Registry interface {
	// GetOpts return the options of Registry
	GetOpts() Options

	// Push a image to registry, return error if something bad happens
	Push(image string,localTag string) error

	// Dump all data to a tar.gz under specified path, return error if something bad happens
	Dump(path string) error

	// List all images managed by this registry
	List() []string

	// Start a registry instance
	Start() error

	// Stop a registry instance
	Stop() error

	// Load extract the tar.gz to load images
	Load(path string) error

	// IsHealth check the registry is healthy
	IsHealth() bool
}
// registry implements Registry
type registry struct {
	// current container id of registry instance
	containerId string

	options *Options

	puller Puller

	// managed images list
	images map[string]string
}




func (r *registry) Start() error {
	binary,err:=FindBestBinary(DEFAULT_CRI_BINARY)
	if err != nil {
		return err
	}

	// check the image is present in local
	image:=r.options.Image
	exist,err:=r.puller.CheckIfPresent(image)
	if err != nil {
		return err
	}
	ctx,cancel:=context.WithDeadline(context.Background(),time.Now().Add(1*time.Hour))
	defer cancel()
	if exist {
		// pull the image if pull policy is "always"
		if r.options.PullPolicy == PULL_POLICY_ALWAYS {
			err:=r.puller.Pull(ctx,image)
			if err != nil {
				return err
			}
		}
		log.Println("skip image-pulling due to image policy,image:",image)
	}else{
		// pull the image if not exist
		err:=r.puller.Pull(ctx,image)
		if err != nil {
			return err
		}
	}


	// docker run -d -p 5000:5000 --name registry -v $(pwd)/data:/var/lib/registry registry:2
	volPair:=fmt.Sprintf("%s:%s",r.options.DataPath,r.options.ContainerPath)
	portPair:=fmt.Sprintf("%d:%d",r.options.HostPort,r.options.ContainerPort)

	cmd:=[]string{binary, "run", "-d", "-v", volPair,"-p",portPair,"--name", r.options.ContainerName, r.options.Image}

	runCommand:=exec.Command("/bin/bash","-c",strings.Join(cmd," "))
	output,err:=runCommand.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exit msg: %s, "+ string(output), err.Error())
	}
	r.containerId = strings.TrimSpace(string(output))
	log.Printf("start a new container registry, container id: %s \n", r.containerId)
	return nil
}

func (r *registry) Stop() error {
	binary,err:=FindBestBinary(DEFAULT_CRI_BINARY)

	if err != nil {
		return err
	}
	cmd:=[]string{binary,"rm","-f",r.containerId}
	cmdRun:=exec.Command("/bin/bash","-c",strings.Join(cmd," "))
	output,err:=cmdRun.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exit msg: %s, "+ string(output), err.Error())
	}
	log.Printf("container %s  successfully deleted \n",r.containerId)
	r.containerId = ""
	// clean up data volume
	cmd=[]string{"rm","-rf",r.options.DataPath}
	output2,err:=BashCommandExec(cmd...)
	if err != nil {
		return ErrorWithStderr(output2,err)
	}
	return nil
}

func (r *registry) GetOpts() Options {
	rp:=r.options
	return *rp
}

func (r *registry) Push(image string,localTag string) error {
	// docker push localhost:5000/test
	fmt.Printf("pushing the image %s to registry\n",image)
	cmd,err:=DockerCmd("push",localTag)
	if err != nil {
		return err
	}
	output, err := BashCommandExec(cmd...)
	if err != nil {
		return ErrorWithStderr(output,err)
	}
	log.Printf("push image %s to registry successfuly \n",image)
	r.images[image]=localTag
	return nil
}
// Dump conforms to the following structure:
// - data.tar.gz: that's data volume of registry
// - images.json:  image pair list in text format(out of order)
// - registry.tar: offline docker images of registry:2
func (r *registry) Dump(path string) error {

	synthetic:=paths.Join("tmp",fmt.Sprintf("dump-%d",rand.Intn(1000)))

	// start a registry instance
	err:=r.Start()
	if err != nil {
		return err
	}
	// wait the instance to be healthy
	err=r.waitUtilHealthy()
	if err != nil {
		return err
	}

	pd:=NewDefaultParallelDocker(r.images,true)
	err=pd.PushImages()
	if err != nil {
		return err
	}

	// clean it whether success or failed
	defer func(){
		// stop the instance
		err1 :=r.Stop()
		if err1 != nil {
			fmt.Println(err.Error())
		}

		// remove the tmp
		err2 := os.RemoveAll("tmp")
		if err2 != nil {
			fmt.Println(err.Error())
		}
	}()


	// copy data volumes
	err=copyDataVolumes(synthetic,r.options.DataPath)

	if err != nil {
		return err
	}
	// docker save registry:2 > registry.tar
	err=SaveRegistryV2DockerImage(synthetic)
	if err != nil {
		return err
	}

	// image list pair: registry.xxx.com/xxx:tag => localhost:5000/xxx:tag
	log.Printf("writing image pair list to %s/%s \n", synthetic, "images.json")
	err=PersistentToFile(r.images,paths.Join(synthetic, "images.json"))
	if err != nil {
		return err
	}


	// compressing the dump files in a whole piece file named "images.tar.gz"
	err=TarCompressTo(path,synthetic)
	if err != nil {
		return err
	}


	return nil
}
// Load from tar.gz.  extract it to current directory
func (r *registry) Load(target string) error{
	// extract to the current directory
	err:=TarExtractFrom(target)
	if err != nil {
		return err
	}
	// load the image of registry:2
	err=LoadRegistryV2DockerImage("registry-v2.tar")
	if err != nil {
		return err
	}
	// start the instance
	err=r.Start()
	if err != nil {
		return err
	}

	defer func() {
		// stop the instance
		fmt.Println("stop the instance")
		err=r.Stop()
		if err != nil {
			fmt.Println(err.Error())
		}
		removeExtractedData()
	}()
	// waiting the registry up
	err=r.waitUtilHealthy()
	if err != nil {
		return err
	}
	// parse the images
	ret,err:=ParseFromFile("images.json")
	if err != nil {
		return err
	}
	r.images=ret

	// load images and retag to origin image tag
	pd:=NewDefaultParallelDocker(ret,true)
	err=pd.PullImages(false)
	if err != nil {
		return err
	}
	err=pd.RetagImages(false)
	if err != nil {
		return err
	}

	return nil
}

// removeExtractedData remove all tmp files extracted
func removeExtractedData(){
	// clean up the extracted data
	os.RemoveAll("data")
	os.RemoveAll("images.json")
	os.RemoveAll(OFFLINE_IMAGE_NAME_OF_REGISTRY_V2)
}
func copyDataVolumes(dst string, source string) error {
	err:=os.MkdirAll(dst,0644)
	if err != nil {
		return err
	}
	// copy the data directory
	log.Printf("copy data directory from %s to %s \n",source,dst)
	cmd:=[]string{"/bin/cp","-rf",source,dst}
	output,err:=BashCommandExec(cmd...)
	if err != nil {
		return ErrorWithStderr(output,err)
	}
	return nil
}


func (r *registry) List() []string {
	ret:=make([]string,0)
	for k,_:=range r.images{
		ret=append(ret,k)
	}
	return ret
}

func (r *registry) IsHealth() bool{
	resp,err:=http.Get("http://localhost:5000/v2/_catalog")
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusOK{
		return false
	}
	return true
}

func (r *registry) waitUtilHealthy() error{
	ctx,cancel:=context.WithTimeout(context.Background(),3*time.Minute)
	defer cancel()
	for !r.IsHealth() {
		select {
		case <-ctx.Done():
			fmt.Printf("waiting for registry up timeout,timeout duartion:%dmin \n",3)
			return ctx.Err()
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
	return nil
}



func NewDefaultOptions() []Opt{
	return []Opt{
		func(options *Options){
			options.ContainerPort= 5000
			options.HostPort=5000
			options.Image= "registry:2"
			wd,_:=os.Getwd()
			options.DataPath=paths.Join(wd,"data")
			options.ContainerPath="/var/lib/registry"
			options.ContainerName="registry"
			options.PullPolicy=PULL_POLICY_IFNOTPRESENT
		},
	}
}


func NewDefaultRegistry(opts... Opt) Registry{

	optPtr:=&Options{}

	for _,opt:=range opts{
		opt(optPtr)
	}
	return &registry{
		options: optPtr,
		puller: NewDefaultPuller(),
		images: make(map[string]string),
	}
}

func NewDefaultRegistryWithImagesPredefined(images map[string]string, opts... Opt) Registry{
	optPtr:=&Options{}

	for _,opt:=range opts{
		opt(optPtr)
	}
	var m map[string]string
	if images==nil{
		m=make(map[string]string)
	}else {
		m=images
	}
	return &registry{
		options: optPtr,
		puller: NewDefaultPuller(),
		images: m,
	}

}


// FindBestBinary find the available cri CLI binary under os path. default to docker
func FindBestBinary(cri string) (string,error){
	if strings.TrimSpace(cri) == ""{
		cri = DEFAULT_CRI_BINARY
	}
	// find absolute path of the 'ls' binary
	path,err:=exec.LookPath(cri)
	if err != nil {
		return "", err
	}
	return path, nil
}