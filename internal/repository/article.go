package repository

import (
	"context"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"gorm.io/gorm"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/repository/cache"
	"gitee.com/geekbang/basic-go/webook/internal/repository/dao/article"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
)

type ArticleRepository interface {
	Create(ctx context.Context, art domain.Article) (int64, error)
	Update(ctx context.Context, art domain.Article) error
	List(ctx context.Context, author int64,
		offset int, limit int) ([]domain.Article, error)
	GetByID(ctx context.Context, id int64) (domain.Article, error)
	// Sync 本身要求先保存到制作库，再同步到线上库
	Sync(ctx context.Context, art domain.Article) (int64, error)
	// SyncStatus 仅仅同步状态
	SyncStatus(ctx context.Context, uid, id int64, status domain.ArticleStatus) error
	GetById(ctx context.Context, id int64) (domain.Article, error)

	GetPublishedById(ctx context.Context, id int64) (domain.Article, error)
	ListPub(ctx context.Context, startTime time.Time, offset int, limit int) ([]domain.Article, error)
}

type ArticleAuthorRepository interface {
	Create(ctx context.Context, art domain.Article) (int64, error)
	Update(ctx context.Context, art domain.Article) error
	//FindById(ctx context.Context, id int64) domain.Article
}
type ArticleReaderRepository interface {
	Create(ctx context.Context, art domain.Article) (int64, error)
	Update(ctx context.Context, art domain.Article) error
	Save(ctx context.Context, art domain.Article) (int64, error)
	//FindById(ctx context.Context, id int64) domain.Article
}

type CachedArticleRepository struct {
	// 操作单一的库
	dao      article.ArticleDAO
	userRepo UserRepository
	// SyncV1 用
	authorDAO article.ArticleAuthorDAO
	readerDAO article.ArticleReaderDAO

	// SyncV2 用
	db    *gorm.DB
	cache cache.ArticleCache
	l     logger.LoggerV1
}

func (repo *CachedArticleRepository) ListPub(ctx context.Context, utime time.Time, offset int, limit int) ([]domain.Article, error) {
	val, err := repo.dao.ListPubByUtime(ctx, utime, offset, limit)
	if err != nil {
		return nil, err
	}
	return slice.Map[article.PublishedArticle, domain.Article](val, func(idx int, src article.PublishedArticle) domain.Article {
		// 偷懒写法
		return repo.toDomain(article.Article(src))
	}), nil
}

func (c *CachedArticleRepository) GetByID(ctx context.Context, id int64) (domain.Article, error) {
	data, err := c.dao.GetById(ctx, id)
	if err != nil {
		return domain.Article{}, err
	}
	return c.toDomain(data), nil
}

func (c *CachedArticleRepository) List(ctx context.Context, author int64, offset int, limit int) ([]domain.Article, error) {
	// 你在这个地方，集成你的复杂的缓存方案
	// 你只缓存这一页
	if offset == 0 && limit <= 100 {
		data, err := c.cache.GetFirstPage(ctx, author)
		if err == nil {
			go func() {
				//缓存
				c.preCache(ctx, data)
			}()
			//return data[:limit], err
			return data, err
		}
	}
	//查询mysql
	res, err := c.dao.GetByAuthor(ctx, author, offset, limit)
	if err != nil {
		return nil, err
	}
	data := slice.Map[article.Article, domain.Article](res, func(idx int, src article.Article) domain.Article {
		return c.toDomain(src)
	})
	// 回写缓存的时候，可以同步，也可以异步
	go func() {
		err := c.cache.SetFirstPage(ctx, author, data)
		c.l.Error("回写缓存失败", logger.Error(err))
		c.preCache(ctx, data)
	}()
	return data, nil
}

func (c *CachedArticleRepository) preCache(ctx context.Context, data []domain.Article) {
	if len(data) > 0 && len(data[0].Content) < 1024*1024 {
		err := c.cache.Set(ctx, data[0])
		if err != nil {
			c.l.Error("提前预加载缓存失败", logger.Error(err))
		}
	}
}

func (c *CachedArticleRepository) Sync(ctx context.Context, art domain.Article) (int64, error) {
	id, err := c.dao.Sync(ctx, c.toEntity(art))
	if err != nil {
		return 0, err
	}
	return id, err
}

func (c *CachedArticleRepository) SyncStatus(ctx context.Context, uid, id int64, status domain.ArticleStatus) error {
	//TODO implement me
	return c.dao.SyncStatus(ctx, uid, id, status.ToUint8())
}

func (c *CachedArticleRepository) GetById(ctx context.Context, id int64) (domain.Article, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CachedArticleRepository) GetPublishedById(ctx context.Context, id int64) (domain.Article, error) {
	//TODO implement me
	//获取文章信息
	pubArt, err := c.dao.GetPubById(ctx, id)
	if err != nil {
		return domain.Article{}, err
	}
	//获取文章对应的用户信息
	user, err := c.userRepo.FindById(ctx, pubArt.AuthorId)
	if err != nil {
		return domain.Article{}, err
	}
	return domain.Article{
		Id:      pubArt.Id,
		Title:   pubArt.Title,
		Content: pubArt.Content,
		Status:  domain.ArticleStatus(pubArt.Status),
		Author: domain.Author{
			Id:   user.Id,
			Name: user.NickName,
		},
		Ctime: time.UnixMilli(pubArt.Ctime),
		Utime: time.UnixMilli(pubArt.Utime),
	}, nil
}

func (c *CachedArticleRepository) Create(ctx context.Context, art domain.Article) (int64, error) {
	//return c.dao.Insert(ctx, article.Article{
	//	Title:    art.Title,
	//	Content:  art.Content,
	//	AuthorId: art.Author.Id,
	//})
	return c.dao.Insert(ctx, c.toEntity(art))
}

func (c *CachedArticleRepository) Update(ctx context.Context, art domain.Article) error {
	//return c.dao.UpdateById(ctx, article.Article{
	//	Id:       art.Id,
	//	Title:    art.Title,
	//	Content:  art.Content,
	//	AuthorId: art.Author.Id,
	//})
	return c.dao.UpdateById(ctx, c.toEntity(art))
}

func (repo *CachedArticleRepository) toDomain(art article.Article) domain.Article {
	return domain.Article{
		Id:      art.Id,
		Title:   art.Title,
		Status:  domain.ArticleStatus(art.Status),
		Content: art.Content,
		Author: domain.Author{
			Id: art.AuthorId,
		},
	}
}

func (repo *CachedArticleRepository) toEntity(art domain.Article) article.Article {
	return article.Article{
		Id:       art.Id,
		Title:    art.Title,
		Content:  art.Content,
		AuthorId: art.Author.Id,
		// 这一步，就是将领域状态转化为存储状态。
		// 这里我们就是直接转换，
		// 有些情况下，这里可能是借助一个 map 来转
		Status: uint8(art.Status),
	}
}

func NewArticleRepository(dao article.ArticleDAO, l logger.LoggerV1) ArticleRepository {
	return &CachedArticleRepository{
		dao: dao,
		l:   l,
	}
}
