package ioc

import (
	"gitee.com/geekbang/basic-go/webook/interactive/repository/dao"
	"gitee.com/geekbang/basic-go/webook/ioc"
	"gitee.com/geekbang/basic-go/webook/pkg/ginx"

	"github.com/IBM/sarama"
	"github.com/gin-gonic/gin"
	prometheus2 "github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"

	"gitee.com/geekbang/basic-go/webook/pkg/gormx/connpool"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
	"gitee.com/geekbang/basic-go/webook/pkg/migrator/events"
	"gitee.com/geekbang/basic-go/webook/pkg/migrator/events/fixer"
	"gitee.com/geekbang/basic-go/webook/pkg/migrator/scheduler"
)

const topic = "migrator_interactives"

func InitFixDataConsumer(l logger.LoggerV1,
	src ioc.SrcDB,
	dst ioc.DstDB,
	client sarama.Client) *fixer.Consumer[dao.Interactive] {
	res, err := fixer.NewConsumer[dao.Interactive](client, l,
		topic, src, dst)
	if err != nil {
		panic(err)
	}
	return res
}

func InitMigradatorProducer(p sarama.SyncProducer) events.Producer {
	return events.NewSaramaProducer(p, topic)
}

func InitMigratorWeb(
	l logger.LoggerV1,
	src ioc.SrcDB,
	dst ioc.DstDB,
	pool *connpool.DoubleWritePool,
	producer events.Producer) *ginx.Server {
	web := gin.Default()
	ginx.InitCounter(prometheus2.CounterOpts{
		Namespace: "geekbang_daming",
		Subsystem: "webook_intr",
		Name:      "http_biz_code",
		Help:      "GIN 中 HTTP 请求",
		ConstLabels: map[string]string{
			"instance_id": "my-instance-1",
		},
	})
	intrs := scheduler.NewScheduler[dao.Interactive](l, src, dst, pool, producer)
	intrs.RegisterRoutes(web.Group("/intr"))
	addr := viper.GetString("migrator.http.addr")
	// 你在这里加别的
	return &ginx.Server{
		Engine: web,
		Addr:   addr,
	}
}

func InitDoubleWritePool(src ioc.SrcDB, dst ioc.DstDB) *connpool.DoubleWritePool {
	return connpool.NewDoubleWritePool(src, dst)
}
