//go:build !k8s

// asdsf go:build dev
// sdd go:build test
// dsf 34

// 没有k8s 这个编译标签
package config

var Config = config{
	DB: DBConfig{
		// 本地连接
		DSN: "root:r4t7u#8i9s@tcp(156.236.71.5:3306)/webook",
	},
	Redis: RedisConfig{
		Addr: "156.236.71.5:6879",
	},
}
