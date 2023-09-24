package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/repository/dao"
	"gitee.com/geekbang/basic-go/webook/internal/service/sms"
)

type SmsRepository interface {
	Select(ctx context.Context, status bool, active bool) ([]domain.SMSBO, error)
	Insert(ctx context.Context, u domain.SMSBO) error
	// status 是否发送成功
	// active 是否需要异步发送
	Update(ctx context.Context, id int, status bool, active bool) error
}

type SMSRepo struct {
	dao dao.SMSDaoInterface
}

func NewSMSRepo(dao dao.SMSDaoInterface) SmsRepository {
	return &SMSRepo{dao: dao}
}

func (S *SMSRepo) Select(ctx context.Context, status bool, active bool) ([]domain.SMSBO, error) {

	smsSlice, err := S.dao.Select(ctx, status, active)
	if err != nil {
		return nil, err
	}
	var res []domain.SMSBO
	res = make([]domain.SMSBO, len(smsSlice))
	for _, ms := range smsSlice {
		tmp, err := S.entityToDomain(ms)
		if err != nil {
			return nil, err
		}
		res = append(res, tmp)
	}
	return res, nil
}

func (S *SMSRepo) Insert(ctx context.Context, u domain.SMSBO) error {
	tmDao, err := S.domainToEntity(u)
	if err != nil {
		fmt.Println("json 转换失败")
		return err
	}
	return S.dao.Insert(ctx, tmDao)
}

func (S *SMSRepo) Update(ctx context.Context, id int, status, active bool) error {
	return S.dao.Update(ctx, id, status, active)
}

func (S *SMSRepo) entityToDomain(msg dao.SmsMsg) (domain.SMSBO, error) {
	var (
		tmpPhoneNumbers []string
		tmpArgVal       []sms.ArgVal
	)
	err := json.Unmarshal([]byte(msg.PhoneNumbers), &tmpPhoneNumbers)
	if err != nil {
		return domain.SMSBO{}, err
	}
	err = json.Unmarshal([]byte(msg.Args), &tmpArgVal)
	if err != nil {
		return domain.SMSBO{}, err
	}

	return domain.SMSBO{
		Id:           int(msg.Id),
		Biz:          msg.Biz,
		PhoneNumbers: tmpPhoneNumbers,
		Args:         tmpArgVal,
	}, nil
}

func (S *SMSRepo) domainToEntity(domain domain.SMSBO) (dao.SmsMsg, error) {
	var (
		tmpPhoneNumbers []byte
		tmpArgVal       []byte
		err             error
	)
	tmpPhoneNumbers, err = json.Marshal(&domain.PhoneNumbers)
	if err != nil {
		return dao.SmsMsg{}, err
	}
	tmpArgVal, err = json.Marshal(&domain.Args)
	if err != nil {
		return dao.SmsMsg{}, err
	}

	return dao.SmsMsg{
		Biz:          domain.Biz,
		PhoneNumbers: string(tmpPhoneNumbers),
		Args:         string(tmpArgVal),
	}, err
}
