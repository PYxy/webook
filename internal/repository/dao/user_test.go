package dao

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestUserDAO_Insert(t *testing.T) {
	testCases := []struct {
		name    string
		mock    func(t *testing.T) *sql.DB
		user    User
		wantErr error
	}{
		{
			name: "邮箱冲突",
			mock: func(t *testing.T) *sql.DB {
				mockDB, mock, err := sqlmock.New()
				//res := sqlmock.NewResult(1, 1)
				mock.ExpectExec("INSERT INTO `users` .*").WillReturnError(&mysql.MySQLError{Number: 1062})
				require.NoError(t, err)
				return mockDB
			},
			wantErr: ErrUserDuplicate,
		},
		{
			name: "插入成功",
			mock: func(t *testing.T) *sql.DB {
				mockDB, mock, err := sqlmock.New()
				res := sqlmock.NewResult(1, 1)
				mock.ExpectExec("INSERT INTO `users` .*").WillReturnResult(res)
				require.NoError(t, err)
				return mockDB
			},
			wantErr: nil,
		},
		{
			name: "其他错误",
			mock: func(t *testing.T) *sql.DB {
				mockDB, mock, err := sqlmock.New()
				//res := sqlmock.NewResult(1, 1)
				mock.ExpectExec("INSERT INTO `users` .*").WillReturnError(errors.New("其他错误"))
				require.NoError(t, err)
				return mockDB
			},
			wantErr: errors.New("其他错误"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(gormMysql.New(gormMysql.Config{
				Conn: tc.mock(t),
				// SELECT VERSION;
				SkipInitializeWithVersion: true,
			}), &gorm.Config{
				// 你 mock DB 不需要 ping
				DisableAutomaticPing: true,
				// 事务开启
				SkipDefaultTransaction: true,
			})
			ud := NewUserDAO(db)
			err = ud.Insert(context.Background(), tc.user)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}
