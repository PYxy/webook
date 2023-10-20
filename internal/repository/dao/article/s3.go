package article

import (
	"bytes"
	"context"
	"gitee.com/geekbang/basic-go/webook/internal/domain"
	_ "github.com/aws/aws-sdk-go"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/ecodeclub/ekit"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strconv"
	"time"
)

var statusPrivate = domain.ArticleStatusPrivate.ToUint8()

type S3DAO struct {
	oss *s3.S3
	// 通过组合 GORMArticleDAO 来简化操作
	// 当然在实践中，你是不太会有组合的机会
	GORMArticleDAO
	bucket *string
}

// NewOssDAO 因为组合 GORMArticleDAO 是一个内部实现细节
// 所以这里要直接传入 DB
func NewOssDAO(oss *s3.S3, db *gorm.DB) ArticleDAO {
	return &S3DAO{
		oss: oss,
		// 你也可以考虑利用依赖注入来传入。
		// 但是事实上这个很少变，所以你可以延迟到必要的时候再注入
		bucket: ekit.ToPtr[string]("webook-1314583317"),
		GORMArticleDAO: GORMArticleDAO{
			db: db,
		},
	}
}

func (o *S3DAO) Sync(ctx context.Context, art Article) (int64, error) {
	// 这一部分和 GORM 的实现差不多
	err := o.db.Transaction(func(tx *gorm.DB) error {
		var (
			id  = art.Id
			err error
		)
		now := time.Now().UnixMilli()
		txDAO := NewGORMArticleDAO(tx)
		if id == 0 {
			id, err = txDAO.Insert(ctx, art)
		} else {
			err = txDAO.UpdateById(ctx, art)
		}
		if err != nil {
			return err
		}
		art.Id = id
		// PublishedArticleV1 不具备 Content
		publishArt := PublishedArticleV1{
			Id:       art.Id,
			Title:    art.Title,
			AuthorId: art.AuthorId,
			Status:   art.Status,
			Ctime:    now,
			Utime:    now,
		}
		return tx.Clauses(clause.OnConflict{
			// ID 冲突的时候。实际上，在 MYSQL 里面你写不写都可以
			Columns: []clause.Column{{Name: "id"}},
			// 这里没有更新 Content，
			//
			DoUpdates: clause.Assignments(map[string]interface{}{
				"title":  publishArt.Title,
				"status": publishArt.Status,
				"utime":  now,
			}),
		}).Create(&publishArt).Error
	})
	if err != nil {
		return 0, err
	}

	// 最后同步到 OSS 上，但是只同步了 Content
	_, err = o.oss.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:      o.bucket,
		Key:         ekit.ToPtr[string](strconv.FormatInt(art.Id, 10)),
		Body:        bytes.NewReader([]byte(art.Content)),
		ContentType: ekit.ToPtr[string]("text/plain;charset=utf-8"),
	})
	return art.Id, err
}

func (o *S3DAO) SyncStatus(ctx context.Context, author, id int64, status uint8) error {
	err := o.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&Article{}).
			Where("id=? AND author_id = ?", id, author).
			Update("status", status)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrPossibleIncorrectAuthor
		}

		res = tx.Model(&PublishedArticleV1{}).
			Where("id=? AND author_id = ?", id, author).Update("status", status)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrPossibleIncorrectAuthor
		}
		return nil
	})
	if err != nil {
		return err
	}
	if status == statusPrivate {
		_, err = o.oss.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
			Bucket: o.bucket,
			Key:    ekit.ToPtr[string](strconv.FormatInt(id, 10)),
		})
	}
	return err
}
