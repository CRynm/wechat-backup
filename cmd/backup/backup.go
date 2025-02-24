package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"wechat-backup/internal/backup"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

		<-stop
	}()

	_ = backup.NewApp().Run(ctx)
}
