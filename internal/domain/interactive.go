package domain

// Interactive 这个是总体交互的计数
type Interactive struct {
	ReadCnt    int64 `json:"read_cnt"`
	LikeCnt    int64 `json:"like_cnt"`
	CollectCnt int64 `json:"collect_cnt"`
	// 这个是当下这个资源，你有没有点赞或者收集
	// 你也可以考虑把这两个字段分离出去，作为一个单独的结构体
	Liked     bool `json:"liked"`
	Collected bool `json:"collected"`
}

// 记录 TOP N 的数据
type TopInteractive struct {
	Id    int64 `json:"id"`
	BizId int64 `json:"biz_id"`
	// 我这里biz 用的是 string，有些公司枚举使用的是 int 类型
	// 0-article
	// 1- xxx
	// 默认是 BLOB/TEXT 类型
	Biz string `json:"biz"`
	// 这个是阅读计数
	ReadCnt int64 `json:"read_cnt"`
	LikeCnt int64 `json:"like_cnt"`
}
