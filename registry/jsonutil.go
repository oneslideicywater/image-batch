package registry

import (
	"encoding/json"
	"os"
	"strings"
)
// PersistentToFile persistent image pair into file
func PersistentToFile(m map[string]string, file string) error{
	bytes, err := json.Marshal(m)
	if err != nil {
		return err
	}
	err = os.WriteFile(file, bytes, 0644)
	if err != nil {
		return err
	}
	return nil
}

func ParseFromFile(file string) (map[string]string,error){
	bytes,err:=os.ReadFile(file)
	if err != nil {
		return map[string]string{},err
	}
	var ret map[string]string
	err=json.Unmarshal(bytes, &ret)
	if err != nil {
		return map[string]string{},err
	}
	return ret,nil
}
// ConfirmDaemonJson deal with /etc/docker/daemon.json, return true if modified the `daemon.json`
//   - if not exist, write a basic insecure-registry config
//   - if existed, confirm that insecure-registry `http://localhost:5000` is set, if not set, add it
// also note that, this function should be called before instance start,as it'll not reboot the docker daemon
func ConfirmDaemonJson() (bool,error){
	configFile:="/etc/docker/daemon.json"

	err := os.MkdirAll("/etc/docker", 0766)
	if err != nil {
		return false,err
	}

	_, err = os.Stat(configFile)
	if err != nil {
		if err == os.ErrNotExist {
			content:= `
{
  "registry-mirrors": [
     "http://hub-mirror.c.163.com",
  ],
  "insecure-registries":["http://localhost:5000"]
}`
			err:=os.WriteFile(configFile,[]byte(content),0666)
			if err != nil {
				return false, err
			}
		}else{
			return false,err
		}
	}

	// if existed, confirm the settings is correct
	bytes,err:=os.ReadFile(configFile)
	if err != nil {
		return false,err
	}
	var result map[string]interface{}
	err=json.Unmarshal(bytes, &result)
	if err != nil {
		return false,err
	}
	v,ok:=result["insecure-registries"]
	if !ok{
		result["insecure-registries"]=[]string{
			"http://localhost:5000",
		}
	}else{

		list:=v.([]interface{})
		for _,line:=range list{
			if strings.TrimSpace(line.(string)) == "http://localhost:5000" {

				return false,nil
			}
		}
		// not found
		list=append(list, "http://localhost:5000")
		result["insecure-registries"] =list

	}
	// persistent into the file
	bytes,err=json.MarshalIndent(result,"","  ")
	if err != nil {
		return false,err
	}
	err=os.WriteFile(configFile,bytes,0666)
	if err != nil {
		return false,err
	}
	return true,nil
}