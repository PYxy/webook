//go:build wireinject

package startup

import (
	"github.com/google/wire"

	repository2 "gitee.com/geekbang/basic-go/webook/interactive/repository"
	cache2 "gitee.com/geekbang/basic-go/webook/interactive/repository/cache"
	dao2 "gitee.com/geekbang/basic-go/webook/interactive/repository/dao"
	service2 "gitee.com/geekbang/basic-go/webook/interactive/service"
)

var thirdProvider = wire.NewSet(InitRedis,
	InitTestDB, InitLog)
var interactiveSvcProvider = wire.NewSet(
	service2.NewInteractiveService,
	repository2.NewCachedInteractiveRepositoryv2,
	dao2.NewGORMInteractiveDAO,
	cache2.NewRedisInteractiveCache,
)

func InitInteractiveService() service2.InteractiveService {
	wire.Build(thirdProvider, interactiveSvcProvider)
	return service2.NewInteractiveService(nil, nil)
}
