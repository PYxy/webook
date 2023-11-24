package test_demo

import (
	"fmt"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

func TestParser(t *testing.T) {
	var parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom |
		cron.Month | cron.Dow | cron.Descriptor)

	//s, _ := parser.Parse("@every 90s")
	//now := time.Now()
	//fmt.Println(now)
	//fmt.Println(s.Next(now))

	s, _ := parser.Parse("0 0 15 */3 *")
	now := time.Now()
	fmt.Println(now)
	fmt.Println(s.Next(now))

}
