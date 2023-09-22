package homework

import (
	"fmt"
	"sync"
	"time"
)

//这个Bucket  看情况去进行修改
//目前是单个字段只统计请求数
//字典的情况可以进行统计(请求数,成功数,失败数 延时值等)
//延时值 可以在搞一个直方图进行统计
//type Buckets struct {
//	block map[string]uint16
//}

type SlipWindow struct {
	mux      sync.RWMutex
	buckets  map[int64]map[string]uint16
	interval time.Duration

	//bucketNumInterval 表示统计的窗口个数对应的时间限制
	bucketNumInterval time.Duration
}

func NewSlipWindow(interval time.Duration, bucketNum int64) *SlipWindow {
	//bucketNum 统计的窗口个数
	return &SlipWindow{
		mux:      sync.RWMutex{},
		buckets:  make(map[int64]map[string]uint16),
		interval: interval,
		//先统计了 时间限度 方便del 的时候
		bucketNumInterval: time.Duration(bucketNum) * interval,
	}
}

func (s *SlipWindow) GetCurrentBucket() map[string]uint16 {
	now := time.Now().Unix()

	if v, ok := s.buckets[now]; ok {
		return v
	} else {

		newBucket := make(map[string]uint16)
		s.buckets[now] = newBucket
		return newBucket
	}
}

// Increment 增加
func (s *SlipWindow) Increment(key string) {

	s.mux.Lock()
	defer s.mux.Unlock()
	bucket := s.GetCurrentBucket()
	bucket[key] += 1
	//fmt.Println(bucket)
	//fmt.Println(s.buckets)
	s.RemoveOldBucket()

}

// RemoveOldBucket 删除过期的Bucket
func (s *SlipWindow) RemoveOldBucket() {
	//这个方法不需要加锁  调用方加了
	var (
		now               time.Time
		deadlineTimeStamp int64
	)
	now = time.Now()
	//法1
	//这里注意单位
	//这里拼接不需要写 单位  他可以自动识别
	m, err := time.ParseDuration(fmt.Sprintf("-%v", s.bucketNumInterval))
	fmt.Println(err)
	//fmt.Println(now.Format(time.StampNano))
	//fmt.Println(now.Add(m).Format(time.StampNano))

	deadlineTimeStamp = now.Add(m).Unix()
	//fmt.Println("法1:", deadlineTimeStamp)
	//法2 有误
	//deadlineTimeStamp = now.Unix() - int64(s.interval*10)
	//fmt.Println("法2", deadlineTimeStamp)
	for tempTimeStamp := range s.buckets {
		if tempTimeStamp <= deadlineTimeStamp {
			delete(s.buckets, tempTimeStamp)
		}
	}
}
