package utils

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)




//Form
func SubmitForm(url string,param url.Values) (*http.Response,error){

	fmt.Println(strings.NewReader(param.Encode()))
	req, err := http.NewRequest("POST", url, strings.NewReader(param.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil{
		fmt.Printf("%s",err.Error())
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))

	return resp,err
}


//Json
func SubmitJson(){

}


func init() {
	cfg := &tls.Config{
		InsecureSkipVerify: true,
	}
	http.DefaultClient.Transport = &http.Transport{
		TLSClientConfig: cfg,
	}
}

func main() {
	httpposturl := "https://ayeshaj:9000/register"
	parm := url.Values{}
	parm.Add("client_id", "ayeshaj")
	parm.Add("response_type","code")
	parm.Add("scope","public_profile")
	parm.Add("redirect_uri","http://ayeshaj:8080/playground")

	resp ,err := SubmitForm(httpposturl,parm)
	if err !=nil{
		fmt.Println(err.Error())
	}
	fmt.Println(resp.Body)
}