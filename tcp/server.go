package tcp

import (
	"context"
	"fmt"
	"GoRedis/interface/tcp"
	"GoRedis/lib/logger"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Config stores tcp server properties
type Config struct {
	Address    string        `yaml:"address"`
	MaxConnect uint32        `yaml:"max-connect"`
	Timeout    time.Duration `yaml:"timeout"`
}

// ClientCounter Record the number of clients in the current Godis server
var ClientCounter int32

// 监听并提供服务，并且在收到closeChan发来的消息后关闭通知后关闭
func ListenAndServe(listener net.Listener, handler tcp.Handler, closeChan <-chan struct{}, address string) {
	// 监听关闭通知
	go func() {
		<-closeChan
		logger.Infof("server %s shutting down...", address)
		// 停止监听， listener.Accept()会立即返回io.EOF
		_ = listener.Close()
		// 关闭应用层服务器
		_ = handler.Close()
	}()

	// 在异常退出后释放资源
	defer func() {
		_ = listener.Close()
		_ = handler.Close()
	}()

	ctx := context.Background()
	wg := sync.WaitGroup{}
	for {
		// 监听端口，阻塞直到收到新的连接或者出现错误
		conn, err := listener.Accept()
		if err != nil {
			break
		}
		// 开启groutine来处理新的连接
		logger.Infof("client %s accept link", conn.RemoteAddr().String())
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.Handle(ctx, conn)
		}()
	}
	wg.Wait()
}

// ListenAndServeWithSignal 监听中断信号并通过 closeChan 通知服务器关闭
func ListenAndServeWithSignal(cfg *Config, handler tcp.Handler) error {
	closeChan := make(chan struct{})
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigCh
		switch sig {
		case syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			closeChan <- struct{}{}
		}
	}()
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return err
	}
	logger.Info(fmt.Sprintf("bind: %s, start listening...", cfg.Address))
	ListenAndServe(listener, handler, closeChan, cfg.Address)
	return nil
}
