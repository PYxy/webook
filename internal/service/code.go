package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"gitee.com/geekbang/basic-go/webook/internal/repository"
	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
)

var (
	ErrFrequentlyForSend      = repository.ErrFrequentlyForSend
	ErrUnknownForCode         = repository.ErrUnknownForCode
	ErrCodeVerifyTooManyTimes = repository.ErrCodeVerifyTooManyTimes
	ErrAttack                 = repository.ErrAttack
	ErrUnKnow                 = repository.ErrKnow
)

func init() {
	rand.Seed(time.Nanosecond.Nanoseconds())
}

type CodeService interface {
	Send(
		ctx context.Context,
		//根据不同的业务场景,使用不同的字符串
		biz string,
		//电话号码
		phone string) error
	Verify(ctx context.Context, biz, phone, code string) (bool, error)
}

// 短信+验证服务
type codeService struct {
	//短信发送服务
	sms sms.Service

	//短信验证服务
	repo repository.CodeRepository
	//code mock
}

func NewCodeService(sms sms.Service, repo repository.CodeRepository) CodeService {
	return &codeService{
		sms:  sms,
		repo: repo,
	}
}

func (c *codeService) Send(
	ctx context.Context,
	//根据不同的业务场景,使用不同的字符串
	biz string,
	//电话号码
	phone string) error {

	code := c.generateCode()
	fmt.Println("生成的验证码: ", code)

	//先保存在数据库
	err := c.repo.Set(ctx, biz, phone, code, 3)

	if err != nil {
		return err
	}

	//发送验证码
	//biz 可以写死 或者从前端哪里获取(jwt 的使用)
	err = c.sms.Send(ctx, biz, []string{phone}, []sms.ArgVal{
		{
			Name: "code",
			Val:  code,
		},
	})
	//直接将结果给到前面
	return err

}

func (c *codeService) Verify(ctx context.Context, biz, phone, code string) (bool, error) {

	return c.repo.Verify(ctx, biz, phone, code)
}

func (c *codeService) generateCode() string {
	// 六位数，num 在 0, 999999 之间，包含 0 和 999999
	num := rand.Intn(1000000)
	// 不够六位的，加上前导 0
	// 000001
	return fmt.Sprintf("%06d", num)
}
