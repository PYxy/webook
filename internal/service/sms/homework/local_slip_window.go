package homework

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

//这个Bucket  看情况去进行修改
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

//-----------------------------根据实际情况做一些变形

type SlipWindowv1 struct {
	mux      sync.RWMutex
	buckets  map[int64]Collect
	interval time.Duration

	//bucketNumInterval 表示统计的窗口个数对应的时间限制
	bucketNumInterval time.Duration
}

func NewSlipWindowv1(interval time.Duration, bucketNum int) *SlipWindowv1 {
	//bucketNum 统计的窗口个数
	return &SlipWindowv1{
		mux:     sync.RWMutex{},
		buckets: make(map[int64]Collect, bucketNum),

		bucketNumInterval: interval,
	}
}

// GetCurrentBucketV1 增加
func (s *SlipWindowv1) getCurrentBucketV1(timeStamp int64) Collect {
	now := timeStamp
	if v, ok := s.buckets[now]; ok {
		return v
	} else {
		newBucket := Collect{}
		s.buckets[now] = newBucket
		return newBucket
	}
}

// IncrementV1 增加
// averageResp  平均的请求响应时间
// longReq 长尾时间限制
func (s *SlipWindowv1) IncrementV1(collect MonitorCollect, timeStamp int64, averageResp, longReq int64) {
	s.mux.Lock()
	defer s.mux.Unlock()
	bucket := s.getCurrentBucketV1(timeStamp)
	if collect.err == nil {
		//处理个数加1
		atomic.AddInt64(&bucket.Count, 1)
		//看下是不是长尾请求 或者 高于 平均请求时间
		if collect.handleTime >= averageResp {
			atomic.AddInt64(&bucket.OverAverageResp, 1)
		}
		if collect.handleTime >= longReq {
			atomic.AddInt64(&bucket.LongResp, 1)
		}
	}
	if collect.err != nil && errors.Is(collect.err, context.DeadlineExceeded) {
		//超时错误
		atomic.AddInt64(&bucket.TimeOutErr, 1)
	} else {
		atomic.AddInt64(&bucket.OtherErr, 1)
	}

	//删除之前的过期Bucket
	s.RemoveOldBucketV1(timeStamp)

}

// RemoveOldBucketV1 删除过期的Bucket
func (s *SlipWindowv1) RemoveOldBucketV1(timeStamp int64) {
	//这个方法不需要加锁  调用方加了
	var (
		now               time.Time
		deadlineTimeStamp int64
	)
	now = time.Unix(timeStamp, 0)
	//法1
	//这里注意单位
	//这里拼接不需要写 单位  他可以自动识别
	m, err := time.ParseDuration(fmt.Sprintf("-%v", s.bucketNumInterval))
	fmt.Println(err)
	//fmt.Println(now.Format(time.StampNano))
	//fmt.Println(now.Add(m).Format(time.StampNano))

	//这里虽然是Add  实际是 减
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

func (s *SlipWindowv1) Statistics(timeStamp int64) (collect Collect) {
	s.mux.Lock()

	//删除之前的过期Bucket
	s.RemoveOldBucketV1(timeStamp)
	tmpMap := make(map[int64]Collect, len(s.buckets))
	for timeStamp1, bucket := range s.buckets {
		tmpMap[timeStamp1] = bucket

	}
	s.mux.Unlock()

	//遍历tmpMap
	for _, bucket := range tmpMap {
		//累计总处理的请求次数
		collect.Count += bucket.Count
		//
		//基于超时的实时检测（连续超时）这里用超时百分比
		collect.TimeOutErr += bucket.TimeOutErr
		//其他错误
		collect.OtherErr += bucket.OtherErr
		////基于响应时间的实时检测（比如说，平均响应时间上升 20%）
		collect.OverAverageResp += bucket.OverAverageResp
		////基于长尾请求的实时检测（比如说，响应时间超过 1s 的请求占比超过了 10%）
		collect.LongResp += bucket.LongResp
	}

	return
}
