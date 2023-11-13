package ioc

import (
	"fmt"
	"time"

	promsdk "github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
	"gorm.io/plugin/prometheus"

	"gitee.com/geekbang/basic-go/webook/config"
	"gitee.com/geekbang/basic-go/webook/internal/repository/dao"
)

func InitMysql() *gorm.DB {
	// 3s 1m 1h
	//viper.GetDuration()
	//没有也不会报错, 会返回类型的零值
	dsn := viper.GetString("db.mysql.dsn")
	fmt.Println(dsn)
	type Config struct {
		DSN string `yaml:"dsn"`
	}
	var cfg Config
	//这里可以设置默认值，即使配置文件没有db.mysql 下面没有DSN 配置项
	//适用
	/*
		redis:
		  dsn: "webook-redis-service:6380"
	*/
	//不适用  会显示为 ""
	/*
		db.mysql:
		  dsn: ""

		redis:
		  dsn: "webook-redis-service:6380"
	*/
	cfg.DSN = "root:root@tcp(127.0.0.1:3306)/webook_default"
	err := viper.UnmarshalKey("db.mysql", &cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println(cfg)

	//dsn 获取为空 它都不会报错的 所有需要做一些默认值的设置
	db, err := gorm.Open(mysql.Open(fmt.Sprintf("%s?charset=utf8&timeout=4s", config.Config.DB.DSN)))
	if err != nil {
		// 我只会在初始化过程中 panic
		// panic 相当于整个 goroutine 结束
		// 一旦初始化过程出错，应用就不要启动了
		panic(err)
	}
	err = db.Use(prometheus.New(prometheus.Config{
		DBName:          "webook",
		RefreshInterval: 15,
		StartServer:     false,
		MetricsCollector: []prometheus.MetricsCollector{
			&prometheus.MySQL{
				VariableNames: []string{"thread_running"},
			},
		},
	}))
	//dao.NewUserDAOv1(func() *gorm.DB {
	//	// 动态获取db 配置
	//	viper.OnConfigChange(func(in fsnotify.Event) {
	//		dsn = viper.GetString("mysql.dsn")
	//		newdb, err := gorm.Open(mysql.Open(dsn))
	//		if err != nil {
	//			fmt.Println("动态获取新配置文件时,mysql 初始化失败")
	//		}
	//		pt := unsafe.Pointer(db)
	//
	//		atomic.StorePointer(&pt, unsafe.Pointer(newdb))
	//
	//	})
	//	//要用原子操作
	//	//pt := unsafe.Pointer(&db)
	//	//val :=atomic.LoadPointer(&pt)
	//	//return (*gorm.DB)(val)
	//	return db
	//})
	pcb := newCallbacks()
	//pcb.registerAll(db)
	//注册钩子 Use 的时候做了 Initialize 的动作
	_ = db.Use(pcb)

	_ = db.Use(tracing.NewPlugin(tracing.WithDBName("webook"),
		tracing.WithQueryFormatter(func(query string) string {
			//l.Debug("", logger.String("query", query))
			fmt.Println("query:", query)
			return query

		}),
		// 不要记录 metrics
		tracing.WithoutMetrics(),
		// 不要记录查询参数
		tracing.WithoutQueryVariables()),
	)

	err = dao.InitTable(db)
	if err != nil {
		panic(err)
	}
	return db
}

type Callbacks struct {
	vector *promsdk.SummaryVec
}

func (pcb *Callbacks) Name() string {
	return "prometheus-query"
}

func (pcb *Callbacks) Initialize(db *gorm.DB) error {
	pcb.registerAll(db)
	return nil
}

func newCallbacks() *Callbacks {
	vector := promsdk.NewSummaryVec(promsdk.SummaryOpts{
		// 在这边，你要考虑设置各种 Namespace
		Namespace: "webook_orm",
		Subsystem: "webook",
		Name:      "gorm_query_time",
		Help:      "统计 GORM 的执行时间",
		ConstLabels: map[string]string{
			"db": "webook",
		},
		Objectives: map[float64]float64{
			0.5:   0.01,
			0.9:   0.01,
			0.99:  0.005,
			0.999: 0.0001,
		},
	},
		// 如果是 JOIN 查询，table 就是 JOIN 在一起的
		// 或者 table 就是主表，A JOIN B，记录的是 A
		[]string{"type", "table"})

	pcb := &Callbacks{
		vector: vector,
	}
	promsdk.MustRegister(vector)
	return pcb
}

func (pcb *Callbacks) registerAll(db *gorm.DB) {
	// 作用于 INSERT 语句
	err := db.Callback().Create().Before("*").
		Register("prometheus_create_before", pcb.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Create().After("*").
		Register("prometheus_create_after", pcb.after("create"))
	if err != nil {
		panic(err)
	}

	err = db.Callback().Update().Before("*").
		Register("prometheus_update_before", pcb.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Update().After("*").
		Register("prometheus_update_after", pcb.after("update"))
	if err != nil {
		panic(err)
	}

	err = db.Callback().Delete().Before("*").
		Register("prometheus_delete_before", pcb.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Delete().After("*").
		Register("prometheus_delete_after", pcb.after("delete"))
	if err != nil {
		panic(err)
	}

	err = db.Callback().Raw().Before("*").
		Register("prometheus_raw_before", pcb.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Raw().After("*").
		Register("prometheus_raw_after", pcb.after("raw"))
	if err != nil {
		panic(err)
	}

	err = db.Callback().Row().Before("*").
		Register("prometheus_row_before", pcb.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Row().After("*").
		Register("prometheus_row_after", pcb.after("row"))
	if err != nil {
		panic(err)
	}
}

func (c *Callbacks) before() func(db *gorm.DB) {
	return func(db *gorm.DB) {
		startTime := time.Now()
		db.Set("start_time", startTime)
	}
}

func (c *Callbacks) after(typ string) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		val, _ := db.Get("start_time")
		startTime, ok := val.(time.Time)
		if !ok {
			// 你啥都干不了
			return
		}
		table := db.Statement.Table
		if table == "" {
			table = "unknown"
		}
		c.vector.WithLabelValues(typ, table).
			Observe(float64(time.Since(startTime).Milliseconds()))
	}
}
