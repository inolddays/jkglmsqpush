package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	nsq "github.com/bitly/go-nsq"
	gocron "github.com/jasonlvhit/gocron"
)

var users Users
var version = "1.0.0PR1"
var filepath = "config.toml"
var modulename = "jkglmsqpush"

func main() {

	args := os.Args
	if len(args) == 2 && (args[1] == "-v") {
		fmt.Println("看好了兄弟，现在的版本是【", version, "】，可别弄错了")
	} else {

		fmt.Println("运行环境初始化进行中...")
		//初始化运行环境
		c := EnvBuild(filepath)
		if c.err != nil {
			panic(c.err)
		}
		fmt.Println("运行环境初始化完毕...")
		//从HMP库中构建users..
		if err := users.BuildFromDb(c.Db1, c.Db2); err != nil {
			Logger.Critical(err)
		}
		//启动处理更改下载时间事件..
		go users.ModifyUsersStarttime(MsgChan)

		for _, v := range users.Sl {
			fmt.Printf("userid:[%d],finishcount:[%d]\n", v.Userid, v.Chufang.Count())
		}

		//对接NSQ，消费上传消息
		consumer, err := NewConsummer(c.Nsqtopic, modulename)
		if err != nil {
			panic(err)
		}

		//Consumer运行，消费消息..
		go func(consumer *nsq.Consumer) {

			err := ConsumerRun(consumer, c.Nsqtopic, c.Nsqaddress)
			if err != nil {
				panic(err)
			}
		}(consumer)

		//debug on 立刻执行
		if strings.EqualFold(c.Debug, "on") {

			TaskWenjuan1(&users, &c)
			TaskWenjuan2(&users, &c)
			//休息10S
			time.Sleep(10 * time.Second)

		} else if strings.EqualFold(c.Debug, "off") {

			//0点1分触发处方完成率滚动任务
			gocron.Every(1).Day().At("00:01").Do(TaskGundong, &users)

			//在指定时间触发固定问卷任务和处方完成率任务
			gocron.Every(1).Day().At(c.Sendtime).Do(TaskWenjuan1, &users, &c)
			gocron.Every(1).Day().At(c.Sendtime).Do(TaskWenjuan2, &users, &c)
			// function Start start all the pending jobs
			<-gocron.Start()
		}
	}
}