package domain

import "time"

type Article struct {
	Id      int64
	Title   string
	Content string
	Status  ArticleStatus
	Author  Author
	Ctime   time.Time
	Utime   time.Time
}

func (a Article) Abstract() string {
	cs := []rune(a.Content)
	if len(cs) < 100 {
		return a.Content
	}
	return string(cs[:100])
}

func (a Article) Published() bool {
	return a.Status == ArticleStatusPublished
}

type ArticleStatus uint8

//go:inline
func (s ArticleStatus) ToUint8() uint8 {
	return uint8(s)
}

const (
	// ArticleStatusUnknown 未知状态
	ArticleStatusUnknown ArticleStatus = iota
	// ArticleStatusUnpublished 未发表
	ArticleStatusUnpublished
	// ArticleStatusPublished 已发表
	ArticleStatusPublished
	// ArticleStatusPrivate 仅自己可见
	ArticleStatusPrivate
)

type Author struct {
	Id   int64
	Name string
}
