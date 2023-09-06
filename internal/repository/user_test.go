package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	cache "gitee.com/geekbang/basic-go/webook/internal/repository/cache"
	cachemocks "gitee.com/geekbang/basic-go/webook/internal/repository/cache/mocks"
	"gitee.com/geekbang/basic-go/webook/internal/repository/dao"
	daomocks "gitee.com/geekbang/basic-go/webook/internal/repository/dao/mocks"
)

func TestCacheUserRepository_FindById(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name     string
		mock     func(ctl *gomock.Controller) (dao.UserDaoInterface, cache.UserCache)
		uid      int64
		wantUser domain.User
		wantErr  error
	}{
		{
			name: "缓存有数据",
			mock: func(ctl *gomock.Controller) (dao.UserDaoInterface, cache.UserCache) {
				uD := daomocks.NewMockUserDaoInterface(ctl)
				uC := cachemocks.NewMockUserCache(ctl)
				//注意这里的18  要进行类型转换 不然会报错  直接写gomock.Any() 也行
				uC.EXPECT().Get(gomock.Any(), int64(18)).Return(domain.User{Id: 18}, nil)
				return uD, uC
			},
			uid:      18,
			wantUser: domain.User{Id: 18},
			wantErr:  nil,
		},
		{
			name: "cache 没数据,dao 也没有数据",
			mock: func(ctl *gomock.Controller) (dao.UserDaoInterface, cache.UserCache) {
				uD := daomocks.NewMockUserDaoInterface(ctl)
				uC := cachemocks.NewMockUserCache(ctl)
				//注意这里的18  要进行类型转换 不然会报错  直接写gomock.Any() 也行
				uC.EXPECT().Get(gomock.Any(), int64(18)).Return(domain.User{}, errors.New("cache没找到"))
				uD.EXPECT().FindById(gomock.Any(), int64(18)).Return(dao.User{}, errors.New("dao也没找到"))
				return uD, uC
			},
			uid:      18,
			wantUser: domain.User{},
			wantErr:  errors.New("dao也没找到"),
		},
		{
			name: "cache 没数据,dao有数据",
			mock: func(ctl *gomock.Controller) (dao.UserDaoInterface, cache.UserCache) {
				uD := daomocks.NewMockUserDaoInterface(ctl)
				uC := cachemocks.NewMockUserCache(ctl)
				//注意这里的18  要进行类型转换 不然会报错  直接写gomock.Any() 也行
				uC.EXPECT().Get(gomock.Any(), int64(18)).Return(domain.User{}, errors.New("cache没找到"))
				uD.EXPECT().FindById(gomock.Any(), int64(18)).Return(dao.User{Id: 18, Ctime: now.UnixMilli()}, nil)
				uC.EXPECT().Set(gomock.Any(), domain.User{
					Id:    18,
					Ctime: time.UnixMilli(now.UnixMilli()),
				}).Return(nil)
				return uD, uC
			},
			uid:      18,
			wantUser: domain.User{Id: 18, Ctime: time.UnixMilli(now.UnixMilli())},
			wantErr:  nil,
		},
		{
			name: "cache 没数据,dao有数据,异步写入失败",
			mock: func(ctl *gomock.Controller) (dao.UserDaoInterface, cache.UserCache) {
				uD := daomocks.NewMockUserDaoInterface(ctl)
				uC := cachemocks.NewMockUserCache(ctl)
				//注意这里的18  要进行类型转换 不然会报错  直接写gomock.Any() 也行
				uC.EXPECT().Get(gomock.Any(), int64(18)).Return(domain.User{}, errors.New("cache没找到"))
				//异步设置缓存的时候失败了
				uC.EXPECT().Set(gomock.Any(), domain.User{
					Id:    18,
					Ctime: time.UnixMilli(now.UnixMilli()),
				}).Return(errors.New("特意写失败"))

				uD.EXPECT().FindById(gomock.Any(), int64(18)).Return(dao.User{Id: 18, Ctime: now.UnixMilli()}, nil)

				return uD, uC
			},
			uid:      18,
			wantUser: domain.User{Id: 18, Ctime: time.UnixMilli(now.UnixMilli())},
			wantErr:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			rP := NewUserRepository(tc.mock(ctrl))
			//ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			//defer cancel()
			resUser, err := rP.FindById(context.Background(), tc.uid)
			//留点时间给别人跑异步任务
			time.Sleep(time.Second)
			assert.Equal(t, tc.wantUser, resUser)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}
