package client

import (
	"context"
	"flag"
	"log"
	"net"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/jiajunhuang/natproxy/dial"
	"github.com/jiajunhuang/natproxy/pb"
	"github.com/jiajunhuang/natproxy/tools"
	"google.golang.org/grpc/metadata"
)

const (
	version = "0.0.9"
	arch    = runtime.GOARCH
	os      = runtime.GOOS
)

var (
	localAddr        = flag.String("local", "127.0.0.1:8080", "-local=<你本地需要转发的地址>")
	serverAddr       = flag.String("server", "natproxy.laizuoceshi.com:8443", "-server=<你的服务器地址>")
	token            = flag.String("token", "", "-token=<你的token>")
	useTLS           = flag.Bool("tls", true, "-tls=true 默认使用TLS加密")
	clientDisconnect int32
)

func checkAnnoncements() {
	annoncement := tools.GetAnnouncement()
	if annoncement != "" {
		log.Printf("最新公告: %s", annoncement)
	}
}

func checkClientStatus() {
	for {
		func() {
			disconnect, err := tools.GetConnectionStatusByToken(*token)
			if err != nil {
				log.Printf("无法连接服务器: %s", err)
				return
			}
			log.Printf("检查当前服务端是否已经把本账号设置成断开连接: %t", disconnect)
			if disconnect == true {
				atomic.StoreInt32(&clientDisconnect, 1)
			} else {
				atomic.StoreInt32(&clientDisconnect, 0)
			}
		}()

		time.Sleep(time.Minute * 5)
	}
}

func connectServer(stream pb.ServerService_MsgClient, addr string) {
	if atomic.LoadInt32(&clientDisconnect) == 1 {
		if err := stream.Send(&pb.MsgRequest{Type: pb.MsgType_DisConnect}); err != nil {
			log.Printf("无法发送消息到服务器: %s", err)
			return
		}
		log.Printf("服务端已经设置为拒绝连接")
		return
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("无法连接服务器(服务器地址%s): %s", addr, err)
		return
	}
	defer conn.Close()

	localConn, err := net.Dial("tcp", *localAddr)
	if err != nil {
		log.Printf("无法连接本地目标地址(%s): %s", *localAddr, err)
		return
	}
	defer localConn.Close()

	dial.Join(conn, localConn)
}

func waitMsgFromServer(addr string) error {
	md := metadata.Pairs("natproxy-token", *token)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	client, conn, err := dial.WithServer(ctx, *serverAddr, *useTLS)
	if err != nil {
		log.Printf("无法连接服务器: %s", err)
		return err
	}
	defer conn.Close()

	log.Printf("准备连接到服务器(%s)...", *serverAddr)

	stream, err := client.Msg(ctx)
	if err != nil {
		log.Printf("无法与服务器通信: %s", err)
		return err
	}
	log.Printf("成功连接到服务器(%s)", *serverAddr)

	// report client version info
	data, err := proto.Marshal(&pb.ClientInfo{Os: os, Arch: arch, Version: version})
	if err != nil {
		log.Printf("无法压缩信息: %s", err)
		return err
	}
	if err := stream.Send(&pb.MsgRequest{Type: pb.MsgType_Report, Data: data}); err != nil {
		log.Printf("无法发送消息到服务器: %s", err)
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			log.Printf("无法从服务器接收消息: %s", err)
			return err
		}

		switch resp.Type {
		case pb.MsgType_Connect:
			log.Printf("服务器要求发起新连接至%s", resp.Data)
			go connectServer(stream, string(resp.Data))
		case pb.MsgType_WANAddr:
			log.Printf("服务器分配的公网地址是%s", resp.Data)
		default:
			log.Printf("当前版本客户端不支持本消息(%s)，请升级", resp.Data)
		}
	}
}

// Start client
func Start(connect, disconnect bool) {
	if *token == "" {
		log.Printf("token不能为空")
		return
	}

	if disconnect {
		err := tools.Disconnect(*token, disconnect)
		log.Printf("通知服务器将本客户端设置为断开连接结果: %s", err)
		return
	}
	if connect {
		err := tools.Disconnect(*token, connect)
		log.Printf("通知服务器将本客户端设置为正常连接结果: %s", err)
		return
	}

	go checkClientStatus()
	go checkAnnoncements()

	for {
		err := waitMsgFromServer(*serverAddr)
		errMsg := err.Error()
		if strings.Contains(errMsg, "token not valid") {
			log.Printf("您的token不对，请检查是否正确配置，参考：https://jiajunhuang.com/natproxy")
			break
		}
		time.Sleep(time.Second * 5)
	}
}
