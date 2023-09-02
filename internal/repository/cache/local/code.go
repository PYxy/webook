package local

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"gitee.com/geekbang/basic-go/webook/internal/repository/cache"
)

import (
	ca "github.com/patrickmn/go-cache"
)

type SmsCache struct {
	client *ca.Cache
	onDo   map[string]*inner
	mutex  sync.RWMutex
}

func NewLocalSmsCache() cache.SmsCache {
	return &SmsCache{
		// 设置超时时间和清理时间
		client: ca.New(5*time.Minute, 10*time.Minute),
		onDo:   make(map[string]*inner, 10),
		mutex:  sync.RWMutex{},
	}
}

func (c *SmsCache) Set(ctx context.Context, biz, phone, code string, cnt int) error {
	//TODO implement me
	//c.client.Set("nihao", "a", time.Second*10)
	//val, err := c.client.Get("nihao")
	//先判断有没有在操作该key
	key := c.GenerateKey(biz, phone)
	//c.mutex.RLock()
	//in, ok := c.onDo[key]
	////有人在操作
	//if ok {
	//	//等待人数+1
	//	atomic.AddInt64(&in.waitCnt, 1)
	//	ch := in.robChan
	//	c.mutex.RUnlock()
	//	//有人在操作 等待
	//	//直接在缓存中获取到该对象
	//	select {
	//	case <-ctx.Done():
	//		//超时直接退出
	//		atomic.AddInt64(&in.waitCnt, -1)
	//		return ctx.Err()
	//	case <-ch:
	//		//到我们操作了,不需要加锁  只有一个人能收到信号
	//		return c.Put(key, 1, code, cnt)
	//	}
	//	//没人在操作
	//} else {
	//
	//	c.mutex.RUnlock()
	//	//没有人操作
	//	c.mutex.Lock() //这里可能有人抢锁了
	//
	//	if ctx.Err() != nil {
	//		c.mutex.Unlock()
	//		return ctx.Err()
	//	}
	//	//抢到锁了  再读一次
	//	in, ok = c.onDo[key]
	//	//有人在抢锁前创建了对象
	//	if ok {
	//		c.mutex.Unlock()
	//		//等待人数+1
	//		atomic.AddInt64(&in.waitCnt, 1)
	//		select {
	//		case <-ctx.Done():
	//			//超时直接退出
	//			atomic.AddInt64(&in.waitCnt, -1)
	//			return ctx.Err()
	//		case <-in.robChan:
	//			//到我们操作了,不需要加锁  只有一个人能收到信号
	//			return c.Put(key, 1, code, cnt)
	//		}
	//	} else {
	//		//没有人在抢锁前创建对象
	//
	//		newIn := &inner{
	//			waitCnt: 0,
	//			robChan: make(chan struct{}, 1),
	//		}
	//		c.onDo[key] = newIn
	//		c.mutex.Unlock()
	//		//直接操作内存
	//		return c.Put(key, 0, code, cnt)
	//
	//	}
	//
	//}

	//为了兼容2个参数返回
	_, err := c.Do(ctx, key, code, cnt, "PUT")
	return err
}

func (c *SmsCache) Verify(ctx context.Context, biz, phone, code string) (bool, error) {
	key := c.GenerateKey(biz, phone)
	return c.Do(ctx, key, code, 0, "GET")
}

// Do 兼容原有的接口必须2个返回值
func (c *SmsCache) Do(ctx context.Context, key string, code string, cnt int, action string) (bool, error) {
	c.mutex.RLock()
	in, ok := c.onDo[key]
	//有人在操作
	if ok {
		//等待人数+1
		atomic.AddInt64(&in.waitCnt, 1)
		ch := in.robChan
		c.mutex.RUnlock()
		//有人在操作 等待
		//直接在缓存中获取到该对象
		select {
		case <-ctx.Done():
			//超时直接退出
			atomic.AddInt64(&in.waitCnt, -1)
			return true, ctx.Err()
		case <-ch:
			//到我们操作了,不需要加锁  只有一个人能收到信号
			switch action {
			case "PUT":
				return true, c.Put(key, 1, code, cnt)
			default:
				return c.Get(key, 1, code)
			}
		}
		//没人在操作
	} else {

		c.mutex.RUnlock()
		//没有人操作
		c.mutex.Lock() //这里可能有人抢锁了
		//这里可能等待的时间会很长 拿到锁马上判断一次是不是超时了
		if ctx.Err() != nil {
			c.mutex.Unlock()
			return true, ctx.Err()
		}
		//抢到锁了  再读一次
		in, ok = c.onDo[key]
		//有人在抢锁前创建了对象
		if ok {
			c.mutex.Unlock()
			//等待人数+1
			atomic.AddInt64(&in.waitCnt, 1)
			select {
			case <-ctx.Done():
				//超时直接退出
				atomic.AddInt64(&in.waitCnt, -1)
				return true, ctx.Err()
			case <-in.robChan:
				switch action {
				case "PUT":
					return true, c.Put(key, 1, code, cnt)
				default:
					return c.Get(key, 1, code)
				}
				//到我们操作了,不需要加锁  只有一个人能收到信号

			}
		} else {
			//没有人在抢锁前创建对象

			newIn := &inner{
				waitCnt: 0,
				robChan: make(chan struct{}, 1),
			}
			c.onDo[key] = newIn
			c.mutex.Unlock()
			//直接操作内存
			switch action {
			case "PUT":
				return true, c.Put(key, 0, code, cnt)
			default:
				return c.Get(key, 0, code)
			}

		}

	}
}

func (c *SmsCache) GenerateKey(biz, phone string) string {
	return fmt.Sprintf("code:%s:%s", biz, phone)
}

func (c *SmsCache) Put(key string, onWait int64, code string, cnt int) (err error) {
	//直接操作内存
	defer func() {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		//获取实时情况
		in, _ := c.onDo[key]
		//先减1 再判断

		atomic.AddInt64(&in.waitCnt, -1*onWait)
		fmt.Println("内存操作完毕：", atomic.LoadInt64(&in.waitCnt))
		if atomic.LoadInt64(&in.waitCnt) <= 0 {
			//通知下一个操作人
			fmt.Println("删除")
			delete(c.onDo, key)

		} else {
			fmt.Println("通知下一个")
			in.robChan <- struct{}{}

		}
	}()
	_, timeExpire, ok := c.client.GetWithExpiration(key)
	if ok {

		//之前已经保存成功
		//判断过期时间是不是永不过期
		if timeExpire.IsZero() {
			return cache.ErrUnknownForCode
		}
		//判断是不是少于10
		fmt.Println(time.Now(), timeExpire)
		//间隔10秒 才能重新设置
		if timeExpire.Sub(time.Now()).Seconds() > 50 {
			return cache.ErrFrequentlyForSend
		}
		//正常设置
		curVal := inCache{
			cnt:  cnt,
			code: code,
		}
		fmt.Println("修改内存的值1")
		c.client.Set(key, curVal, time.Second*60)
	} else {
		//没有保存过或者已经过期
		curVal := inCache{
			cnt:  cnt,
			code: code,
		}
		fmt.Println("修改内存的值2")
		c.client.Set(key, curVal, time.Second*60)

	}

	return err

}

func (c *SmsCache) Get(key string, onWait int64, code string) (bool, error) {
	//return false, nil
	//直接操作内存
	defer func() {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		//获取实时情况
		in, _ := c.onDo[key]
		//先减1 再判断

		atomic.AddInt64(&in.waitCnt, -1*onWait)
		fmt.Println("内存操作完毕：", atomic.LoadInt64(&in.waitCnt))
		if atomic.LoadInt64(&in.waitCnt) <= 0 {
			//通知下一个操作人
			fmt.Println("删除")
			delete(c.onDo, key)

		} else {
			fmt.Println("通知下一个")
			in.robChan <- struct{}{}

		}
	}()
	val, timeExpire, ok := c.client.GetWithExpiration(key)
	if !ok {
		//验证的时候查询不到对的key
		fmt.Println("是攻击 或者 是上一次登录成功 又登录")
		return false, cache.ErrAttack
		//判断是不是少于10
	} else {
		tmpInCache := val.(inCache)
		if tmpInCache.cnt <= 0 {
			fmt.Println("错误次数超标,还在尝试")
			//用户一直输错验证码  大概率有人攻击
			return false, cache.ErrCodeVerifyTooManyTimes
		} else if tmpInCache.code == code {
			fmt.Println("密码正确")
			tmpInCache.cnt = -1
			//重新设置  按照之前的过期时间
			c.client.Set(key, tmpInCache, time.Second*60)
			//直接删除 可能会有大量攻击
			//c.client.Delete(key)
			return true, nil
		} else {
			fmt.Println("验证码输入错误")
			//验证码输入错误
			tmpInCache.cnt -= 1
			//重新设置  按照之前的过期时间

			c.client.Set(key, tmpInCache, time.Duration(timeExpire.Second())*time.Second)
			return false, nil
		}

	}

}

type inner struct {
	//记录请求次数
	waitCnt int64
	//
	robChan chan struct{}
}

type inCache struct {
	//记录请求次数
	cnt int
	//验证码
	code string
}
