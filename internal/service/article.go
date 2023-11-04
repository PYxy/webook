package service

import (
	"context"
	"time"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/events/article"
	"gitee.com/geekbang/basic-go/webook/internal/repository"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
)

type ArticleService interface {
	Save(ctx context.Context, art domain.Article) (int64, error)
	Publish(ctx context.Context, art domain.Article) (int64, error)
	Withdraw(ctx context.Context, uid, id int64) error
	PublishV1(ctx context.Context, art domain.Article) (int64, error)
	List(ctx context.Context, author int64,
		offset, limit int) ([]domain.Article, error)
	GetById(ctx context.Context, id int64) (domain.Article, error)

	// GetPublishedById 查找已经发表的
	// 剩下的这个是给读者用的服务，暂时放到这里
	// 正常来说在微服务架构下，读者服务和创作者服务会是两个独立的服务
	GetPublishedById(ctx context.Context, id int64, uid int64) (domain.Article, error)
}

type articleService struct {
	// 1. 在 service 这一层使用两个 repository
	author repository.ArticleAuthorRepository
	reader repository.ArticleReaderRepository

	// 2. 在 repo 里面处理制作库和线上库
	repo repository.ArticleRepository

	l        logger.LoggerV1
	producer article.Producer
}

func (a *articleService) Withdraw(ctx context.Context, uid, id int64) error {
	//TODO implement me
	return a.repo.SyncStatus(ctx, uid, id, domain.ArticleStatusPrivate)
}

func (a *articleService) List(ctx context.Context, author int64, offset, limit int) ([]domain.Article, error) {
	//TODO implement me
	return a.repo.List(ctx, author, offset, limit)

}

func (a *articleService) GetById(ctx context.Context, id int64) (domain.Article, error) {
	return a.repo.GetByID(ctx, id)
}

func (a *articleService) GetPublishedById(ctx context.Context, id, uid int64) (domain.Article, error) {

	art, err := a.repo.GetPublishedById(ctx, id)
	if err == nil {
		go func() {
			er := a.producer.ProduceReadEvent(ctx, article.ReadEvent{
				Uid: uid,
				Aid: id,
			})
			if er != nil {
				a.l.Error("发送读者阅读事件失败")
			}
		}()
	}
	return art, err
}

func (a *articleService) Publish(ctx context.Context, art domain.Article) (int64, error) {
	art.Status = domain.ArticleStatusPublished

	return a.repo.Sync(ctx, art)
}

func (a *articleService) PublishV1(ctx context.Context, art domain.Article) (int64, error) {
	var (
		id  = art.Id
		err error
	)
	if art.Id > 0 {
		err = a.author.Update(ctx, art)
	} else {
		id, err = a.author.Create(ctx, art)
	}
	if err != nil {
		return 0, err
	}
	art.Id = id
	for i := 0; i < 3; i++ {
		time.Sleep(time.Second * time.Duration(i))
		id, err = a.reader.Save(ctx, art)
		if err == nil {
			break
		}
		a.l.Error("部分失败，保存到线上库失败",
			logger.Int64("art_id", art.Id),
			logger.Error(err))
	}
	if err != nil {
		a.l.Error("部分失败，重试彻底失败",
			logger.Int64("art_id", art.Id),
			logger.Error(err))
		// 接入你的告警系统，手工处理一下
		// 走异步，我直接保存到本地文件
		// 走 Canal
		// 打 MQ
	}
	return id, err
}

func NewArticleService(repo repository.ArticleRepository, l logger.LoggerV1, producer article.Producer) ArticleService {
	return &articleService{
		repo:     repo,
		l:        l,
		producer: producer,
	}
}

func NewArticleServiceV1(author repository.ArticleAuthorRepository,
	reader repository.ArticleReaderRepository, l logger.LoggerV1) ArticleService {
	return &articleService{
		author: author,
		reader: reader,
		l:      l,
	}
}

func (a *articleService) Save(ctx context.Context, art domain.Article) (int64, error) {
	// 设置为未发表
	art.Status = domain.ArticleStatusUnpublished
	if art.Id > 0 {
		err := a.repo.Update(ctx, art)
		return art.Id, err
	}
	return a.repo.Create(ctx, art)
}

func (a *articleService) update(ctx context.Context, art domain.Article) error {
	// 只要你不更新 author_id
	// 但是性能比较差
	//artInDB := a.repo.FindById(ctx, art.Id)
	//if art.Author.Id != artInDB.Author.Id {
	//	return errors.New("更新别人的数据")
	//}
	return a.repo.Update(ctx, art)
}
