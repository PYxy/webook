package local

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/demdxx/gocast/v2"
	ca "github.com/patrickmn/go-cache"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/repository/cache"
)

type UserCache struct {
	client *ca.Cache
}

func NewUserCache() cache.UserCache {

	return &UserCache{
		client: ca.New(5*time.Minute, 10*time.Minute),
	}
}

func (c *UserCache) Get(ctx context.Context, id int64) (domain.User, error) {
	//TODO implement me
	val, ok := c.client.Get(gocast.Str(id))
	fmt.Println("local cache 获取用户id 信息")
	if ok {
		//数据在本地缓存
		u, ok := val.(domain.User)
		if !ok {
			return domain.User{}, errors.New("系统错误")
		}
		return u, nil
	} else {
		//数据在本地缓存中没有
		return domain.User{}, errors.New("对象不存在")
	}
}

func (c *UserCache) Set(ctx context.Context, u domain.User) error {
	//TODO implement me
	fmt.Println("local cache 保存用户id 信息")
	c.client.Set(gocast.Str(u.Id), u, time.Second*60)
	return nil
}
