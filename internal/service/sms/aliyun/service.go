package aliyun

//阿里云短信发送服务

import (
	"context"
	"fmt"
	"strings"
	"time"

	dysmsapi "github.com/aliyun/alibaba-cloud-sdk-go/services/dysmsapi"
	"github.com/bytedance/sonic"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"

	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
)

type Service struct {
	client          *dysmsapi.Client
	accessKey       string
	accessKeySecret string
	signName        string
	templateCode    string
	regionId        string
}

// NewAliyunService 使用wrie 要多包一层 或者变量用全局变量
func NewAliyunService(
	accessID,
	accessKeySecret,
	regionId,
	signName,
	templateCode string) *Service {
	config := sdk.NewConfig()
	config.WithTimeout(time.Second * 5)
	credential := credentials.NewAccessKeyCredential(accessID, accessKeySecret)
	client, err := dysmsapi.NewClientWithOptions(regionId, config, credential)
	if err != nil {
		panic(err)
	}
	return &Service{
		client:       client,
		signName:     signName,
		templateCode: templateCode,
	}
}

// Send  短信发送
// ArgVal 可以根据实际需求灵活变换数据结构
// tpl 是业务类型  aliyun 这里用不着
func (s *Service) Send(ctx context.Context, tpl string, phoneNumbers []string, args []sms.ArgVal) error {
	//TODO implement me
	request := dysmsapi.CreateSendSmsRequest()
	fmt.Println("alyun业务类型:", tpl)
	request.Scheme = "https"
	request.SignName = s.signName
	request.TemplateCode = s.templateCode
	//电话,电话，..
	request.PhoneNumbers = strings.Join(phoneNumbers, ",")
	//参数信息
	tmpMap := make(map[string]string, len(args))
	for _, arg := range args {
		tmpMap[arg.Name] = arg.Val
	}

	//map  转json 字符串
	byteCode, err := sonic.Marshal(tmpMap)
	if err != nil {
		return err
	}
	request.TemplateParam = string(byteCode)
	response, err := s.client.SendSms(request)
	if err != nil {
		return err
	}
	fmt.Printf("response is %#v\n", response)
	return nil
}
