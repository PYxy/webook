package integration

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

// ArticleTestSuite 测试套件
type ArticleTestSuite struct {
	suite.Suite
}

func (s *ArticleTestSuite) TestABC() {
	s.T().Log("hello，这是测试套件")
}

// TestArticle 启动测试用例
func TestArticle(t *testing.T) {
	suite.Run(t, &ArticleTestSuite{})
}
