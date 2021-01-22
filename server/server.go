package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/lucas-clemente/quic-go"
	"os"
	"time"
	"merry/common"
	"github.com/golang/protobuf/proto"
	"merry/proto"
	"runtime"
)

type Flag struct {
	s string
	t float64
	h bool
}

var ff = Flag{}

func init() {
	flag.StringVar(&ff.s, "s", "0.0.0.0:8282", "server binding address")
	flag.Float64Var(&ff.t, "t", 1, "sleep time")
	flag.BoolVar(&ff.h, "h", false, "print this help message")
}

func serverMain() error {
	defer glog.Flush()
	listener, err := quic.ListenAddr(ff.s, common.GenerateTLSConfig(), nil)
	if err != nil {
		return err
	}

	ch := make(chan int, 1)

	// bandwidth throttle
	go func (ch chan int) {
		for {
			ch <- 0
			time.Sleep(time.Duration(ff.t) * time.Microsecond)
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
			fmt.Println(err)
			continue
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
				ff.t = float64(1000000.0 / fileRequest.GetBandwidth())
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

				offset := fileRequest.GetOffset()
				fileResponse := &merry_proto.FileResponse{
					Status: merry_proto.StatusCode_kOK,
					Size: stat.Size() - offset,
				}
				err = common.RpcSendFileResponse(fileResponse, stream)
				if err != nil {
					fmt.Println(err)
					return
				}

				err = common.RpcSendFileChunk(ch, f, offset, stream)
				if err != nil {
					fmt.Println(err)
					return
				}

				fmt.Printf("file send done, size: %d\n", stat.Size() - offset)
			}
		}(ch, stream)
	}

	return nil
}

func main() {
	flag.Parse()
	if ff.h {
		flag.Usage()
		return
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	err := serverMain()
	if err != nil {
		panic(err)
	}
}
