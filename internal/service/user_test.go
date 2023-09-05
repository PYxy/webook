package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/repository"
	svcmocks "gitee.com/geekbang/basic-go/webook/internal/repository/mocks"
)

func Test_userService_Login(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name      string
		mock      func(ctl *gomock.Controller) repository.UserRepository
		wantError error
		wantUser  domain.User
		email     string
		pwd       string
	}{
		{
			name: "登录成功",
			mock: func(ctl *gomock.Controller) repository.UserRepository {
				mockRepo := svcmocks.NewMockUserRepository(ctl)
				mockRepo.EXPECT().FindByEmail(gomock.Any(), "asxxxxxxxxxx@163.com").
					Return(domain.User{
						Id:       0,
						Email:    "asxxxxxxxxxx@163.com",
						Password: "$2a$10$eUWOwR1e2oFBO6Jpi3h9A..CGPuiiXH9ipl7AsGPz8Nf0Ts8UcmXW",
						Ctime:    now,
					}, nil)

				return mockRepo
			},
			wantError: nil,
			wantUser: domain.User{
				Id:       0,
				Email:    "asxxxxxxxxxx@163.com",
				Password: "$2a$10$eUWOwR1e2oFBO6Jpi3h9A..CGPuiiXH9ipl7AsGPz8Nf0Ts8UcmXW",
				Ctime:    now,
			},
			email: "asxxxxxxxxxx@163.com",
			pwd:   "xxxx@1xxxxxxxxx",
		},
		{
			name: "找不到该用户",
			mock: func(ctl *gomock.Controller) repository.UserRepository {
				mockRepo := svcmocks.NewMockUserRepository(ctl)
				mockRepo.EXPECT().FindByEmail(gomock.Any(), gomock.Any()).
					Return(domain.User{}, ErrInvalidUserOrPassword)

				return mockRepo
			},
			wantError: ErrInvalidUserOrPassword,
			wantUser:  domain.User{},
			email:     "asxxxxxxxxxx@163.com",
			pwd:       "xxxx@1xxxxxxxxx",
		},
		{
			name: "其他错误",
			mock: func(ctl *gomock.Controller) repository.UserRepository {
				mockRepo := svcmocks.NewMockUserRepository(ctl)
				mockRepo.EXPECT().FindByEmail(gomock.Any(), gomock.Any()).
					Return(domain.User{}, errors.New("其他错误"))

				return mockRepo
			},
			wantError: errors.New("其他错误"),
			wantUser:  domain.User{},
			email:     "asxxxxxxxxxx@163.com",
			pwd:       "xxxx@1xxxxxxxxx",
		},
		{
			name: "密码错误",
			mock: func(ctl *gomock.Controller) repository.UserRepository {
				mockRepo := svcmocks.NewMockUserRepository(ctl)
				mockRepo.EXPECT().FindByEmail(gomock.Any(), "asxxxxxxxxxx@163.com").
					Return(domain.User{
						Id:       0,
						Email:    "asxxxxxxxxxx@163.com",
						Password: "$2a$10$eUWOwR1e2oFBO6Jpi3h9A..CGPuiiXH9ipl7AsGPz8Nf0Ts8UcmXW",
						Ctime:    now,
					}, nil)

				return mockRepo
			},
			wantError: ErrInvalidUserOrPassword,
			wantUser:  domain.User{},
			email:     "asxxxxxxxxxx@163.com",
			pwd:       "xxxx@1xxxxxxxx1x",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			defer cancel()
			ctl := gomock.NewController(t)
			uService := NewUserService(tc.mock(ctl))
			u, e := uService.Login(ctx, tc.email, tc.pwd)
			assert.Equal(t, tc.wantUser, u)
			assert.Equal(t, tc.wantError, e)
		})
	}

}

func Test_pwd(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("xxxx@1xxxxxxxxx"), bcrypt.DefaultCost)
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(string(hash))
}
