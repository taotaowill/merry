package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/lucas-clemente/quic-go"
	"os"
	"time"
	"runtime"
	"merry/common"
	"github.com/golang/protobuf/proto"
	"merry/proto"
)

const SERVER_ADDR = "0.0.0.0:8282"
var (SLEEP_TIME float64)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	err := echoServer()
	if err != nil {
		panic(err)
	}
}

func echoServer() error {
	flag.Parse()
	defer glog.Flush()

	SLEEP_TIME = 1
	listener, err := quic.ListenAddr(SERVER_ADDR, common.GenerateTLSConfig(), nil)
	if err != nil {
		return err
	}

	ch := make(chan int, 1)

	// bandwidth throttle
	go func (ch chan int) {
		for {
			ch <- 0
			time.Sleep(time.Duration(SLEEP_TIME) * time.Microsecond)
			//fmt.Printf("sleep time: %f\n", SLEEP_TIME)
		}
	} (ch)

	for {
		session, err := listener.Accept(context.Background())
		if err != nil {
			return err
		}

		fmt.Printf("new connection: %s ...\n", session.RemoteAddr().String())
		stream, err := session.AcceptStream(context.Background())
		if err != nil {
			panic(err)
		}

		// go func
		go func(ch chan int, stream quic.Stream) {
			fileRequest := &merry_proto.FileRequest{}
			err = common.RpcReadFileRequest(fileRequest, stream)
			if err != nil {
				return
			}

			fmt.Printf("file request recv:\n%s", proto.MarshalTextString(fileRequest))
			if fileRequest.GetBandwidth() > 0 {
				SLEEP_TIME = float64(1000000.0 / fileRequest.GetBandwidth())
				fileResponse := &merry_proto.FileResponse{
					Status: merry_proto.StatusCode_kOK,
				}
				err = common.RpcSendFileResponse(fileResponse, stream)
				if err != nil {
					fmt.Println(err)
					return
				}

				fmt.Printf("file response done, set bandwidth: %dKB/s\n", fileRequest.GetBandwidth())
				<- ch
			} else {
				f, err := os.Open(fileRequest.GetPath())
				defer f.Close()
				if err != nil {
					fmt.Println(err)
					return
				}

				stat, err := f.Stat()
				if err != nil {
					fmt.Println(err)
					return
				}

				fileResponse := &merry_proto.FileResponse{
					Status: merry_proto.StatusCode_kOK,
					Size: stat.Size(),
				}
				err = common.RpcSendFileResponse(fileResponse, stream)
				if err != nil {
					fmt.Println(err)
					return
				}

				err = common.RpcSendFileChunk(ch, f, stream)
				if err != nil {
					fmt.Println(err)
					return
				}

				fmt.Printf("file send done, size: %d\n", stat.Size())
			}
		}(ch, stream)
	}

	return nil
}
