package repository

import (
	"context"

	"gorm.io/gorm"

	"gitee.com/geekbang/basic-go/webook/internal/domain"
	"gitee.com/geekbang/basic-go/webook/internal/repository/dao/article"
	"gitee.com/geekbang/basic-go/webook/pkg/logger"
)

type ArticleRepository interface {
	Create(ctx context.Context, art domain.Article) (int64, error)
	Update(ctx context.Context, art domain.Article) error
	List(ctx context.Context, author int64,
		offset int, limit int) ([]domain.Article, error)

	// Sync 本身要求先保存到制作库，再同步到线上库
	Sync(ctx context.Context, art domain.Article) (int64, error)
	// SyncStatus 仅仅同步状态
	SyncStatus(ctx context.Context, uid, id int64, status domain.ArticleStatus) error
	GetById(ctx context.Context, id int64) (domain.Article, error)

	GetPublishedById(ctx context.Context, id int64) (domain.Article, error)
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
	dao article.ArticleDAO

	// SyncV1 用
	authorDAO article.ArticleAuthorDAO
	readerDAO article.ArticleReaderDAO

	// SyncV2 用
	db *gorm.DB
	l  logger.LoggerV1
}

func (c *CachedArticleRepository) List(ctx context.Context, author int64, offset int, limit int) ([]domain.Article, error) {
	//TODO implement me
	panic("implement me")
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
	panic("implement me")
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

func NewArticleRepository(dao article.ArticleDAO) ArticleRepository {
	return &CachedArticleRepository{
		dao: dao,
	}
}
