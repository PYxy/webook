package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
)

var redirectURI = url.PathEscape("https://xxxxx/oauth2/wechat/callback")

type Service interface {
	AuthURL(ctx context.Context, state string) (string, error)
	VerifyCode(ctx context.Context, code string) (domain.WechatInfo, error)
}

type service struct {
	appId     string
	appSecret string
	client    *http.Client
}

func NewServiceV1(appId string, appSecret string, client *http.Client) Service {
	return &service{
		appId:     appId,
		appSecret: appSecret,
		client:    client,
	}
}

func NewService(appId string, appSecret string) Service {
	return &service{
		appId:     appId,
		appSecret: appSecret,
		// 依赖注入，但是没完全注入
		client: http.DefaultClient,
	}
}

// AuthURL
// 前端点击微信登录 这边返回一个回调的url 给前端 让它进行跳转
func (s *service) AuthURL(ctx context.Context, state string) (string, error) {
	const urlPattern = "https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s#wechat_redirect"
	//保存state  避免别人拿着微信的临时码 去绑定它自己的账号
	// 如果在这里存 state，假如说我存 redis

	//s.cmd.Set(ctx, "唯一键"+state, state, time.Minute)

	return fmt.Sprintf(urlPattern, s.appId, redirectURI, state), nil
}

// VerifyCode 根据获取到的 临时授权码 再去请求微信拿到详细信息
func (s *service) VerifyCode(ctx context.Context, code string) (domain.WechatInfo, error) {
	// 攻击者的 state
	//str := s.cmd.Get(ctx, "my-state"+state).String()
	//if str != state {
	//	// 不相等
	//}
	const targetPattern = "https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code"
	target := fmt.Sprintf(targetPattern, s.appId, s.appSecret, code)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	//req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return domain.WechatInfo{}, err
	}
	// 会产生复制，性能极差，比如说你的 URL 很长
	//req = req.WithContext(ctx)
	resp, err := s.client.Do(req)
	if err != nil {
		return domain.WechatInfo{}, err
	}
	// 只读一遍
	decoder := json.NewDecoder(resp.Body)
	var res Result
	err = decoder.Decode(&res)

	// 整个响应都读出来，不推荐，因为 Unmarshal 再读一遍，合计两遍
	//body, err := io.ReadAll(resp.Body)
	//err = json.Unmarshal(body, &res)

	if err != nil {
		return domain.WechatInfo{}, err
	}

	if res.ErrCode != 0 {
		return domain.WechatInfo{},
			fmt.Errorf("微信返回错误响应，错误码：%d，错误信息：%s", res.ErrCode, res.ErrMsg)
	}

	return domain.WechatInfo{
		OpenID:  res.OpenID,
		UnionID: res.UnionID,
	}, nil
}

type Result struct {
	ErrCode int64  `json:"errcode"`
	ErrMsg  string `json:"errmsg"`

	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`

	OpenID  string `json:"openid"`
	Scope   string `json:"scope"`
	UnionID string `json:"unionid"`
}
