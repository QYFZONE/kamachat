package main

import (
	"fmt"
	"kama_chat_server/internal/config"
	"kama_chat_server/internal/https_server"
	//"kama_chat_server/internal/service/chat"
	//"kama_chat_server/internal/service/kafka"
	myredis "kama_chat_server/internal/service/redis"
	"kama_chat_server/pkg/zlog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	conf := config.GetConfig()
	host := conf.MainConfig.Host
	port := conf.MainConfig.Port

	// 本地先不启 chat / kafka
	// kafkaConfig := conf.KafkaConfig
	// if kafkaConfig.MessageMode == "kafka" {
	// 	kafka.KafkaService.KafkaInit()
	// }
	//
	// if kafkaConfig.MessageMode == "channel" {
	// 	go chat.ChatServer.Start()
	// } else {
	// 	go chat.KafkaChatServer.Start()
	// }

	go func() {
		if err := https_server.GE.Run(fmt.Sprintf("%s:%d", host, port)); err != nil {
			zlog.Fatal("server running fault")
			return
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 本地先不关 chat / kafka
	// if kafkaConfig.MessageMode == "kafka" {
	// 	kafka.KafkaService.KafkaClose()
	// }
	// chat.ChatServer.Close()

	zlog.Info("关闭服务器...")

	if err := myredis.DeleteAllRedisKeys(); err != nil {
		zlog.Error(err.Error())
	} else {
		zlog.Info("所有Redis键已删除")
	}

	zlog.Info("服务器已关闭")
}
