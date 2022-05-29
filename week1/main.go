package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

func main() {
	//通过withcontext创建一个带取消的Group
	g, ctx := errgroup.WithContext(context.Background())
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})
	serverOut := make(chan struct{})
	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		serverOut <- struct{}{}
	})
	server := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}
	g.Go(func() error {
		return server.ListenAndServe()
	})
	//接收 serverOut 消息
	g.Go(func() error {
		select {
		case <-ctx.Done():
			log.Println("errgroup exit...")
		case <-serverOut:
			log.Println("server will out...")
		}
		timeoutCtx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		//defer cancel()
		log.Println("shutting down server...")
		return server.Shutdown(timeoutCtx)
	})
	//信号退出
	//go 方法传入一个func() error 内部会启动一个goroutine去处理
	g.Go(func() error {
		quit := make(chan os.Signal, 0)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case sig := <-quit:
			return errors.Errorf("get os signal: %v", sig)
		}
	})
	fmt.Printf("errgroup exiting: %+v\n", g.Wait())
}
