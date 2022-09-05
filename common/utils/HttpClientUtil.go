package utils

import (
	"context"
	"crypto/tls"
	"github.com/gioco-play/gozzle"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"net/url"
	"strings"
	"time"
)

//Form
//func SubmitForm2(url string,param url.Values) (*http.Response,error){
//
//	fmt.Println(strings.NewReader(param.Encode()))
//	req, err := http.NewRequest("POST", url, strings.NewReader(param.Encode()))
//	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
//
//	client := &http.Client{}
//	resp, err := client.Do(req)
//	if err != nil{
//		fmt.Printf("%s",err.Error())
//		panic(err)
//	}
//	defer resp.Body.Close()
//	body, err := ioutil.ReadAll(resp.Body)
//	if err !=nil{
//		logx.Error(err.Error())
//	}
//	fmt.Println("response Status:", resp.Status)
//	fmt.Println("response Headers:", resp.Header)
//	fmt.Println("response Body:", string(body))
//	return resp,err
//}

func SubmitForm(url string, param url.Values, context context.Context) (*gozzle.Response, error) {
	logx.WithContext(context).Info(strings.NewReader(param.Encode()))
	span := trace.SpanFromContext(context)

	t := http.DefaultTransport.(*http.Transport)
	t.MaxConnsPerHost = 100 //最大連線池數
	resp, err := gozzle.Post(url).Transport(t).Timeout(20).Trace(span).Form(param)
	if err != nil {
		logx.WithContext(context).Error("渠道返回错误: ", err.Error())
	}
	logx.WithContext(context).Info("response Headers:", resp.Header)
	logx.WithContext(context).Info("response Status:", resp.Status())
	logx.WithContext(context).Info("response Body:", string(resp.Body()))

	return resp, nil
}

//Json
func SubmitJson() {

}

// Response HTTP-response struct
type Response struct {
	request *Request
	status  int
	headers http.Header
	cookies []*http.Cookie
	body    []byte
}

type Request struct {
	method  string
	url     string
	header  http.Header
	cookies []*http.Cookie
	body    []byte
	debug   DebugHandler
	options
}

type DebugHandler func(*Response)

type options struct {
	clientTransport http.RoundTripper
	clientTimeout   time.Duration
}

func init() {
	cfg := &tls.Config{
		InsecureSkipVerify: true,
	}
	http.DefaultClient.Transport = &http.Transport{
		TLSClientConfig: cfg,
	}
}

//func main() {
//	//httpposturl := "https://ayeshaj:9000/register"
//	parm := url.Values{}
//	parm.Add("client_id", "ayeshaj")
//	parm.Add("response_type", "code")
//	parm.Add("scope", "public_profile")
//	parm.Add("redirect_uri", "http://ayeshaj:8080/playground")
//
//	//resp ,err := SubmitForm2(httpposturl,parm)
//	//if err !=nil{
//	//	fmt.Println(err.Error())
//	//}
//	//fmt.Println(resp.Body)
//}
