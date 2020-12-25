package main

import (
	"crypto/tls"
	"github.com/lucas-clemente/quic-go"
	"flag"
	"fmt"
	"context"
	"time"
	"merry/proto"
	"merry/common"
	"path"
)

type Flag struct {
	b int
	f string
	s string
	h bool
}

var ff = Flag{}

func init() {
	flag.IntVar(&ff.b, "b", -1, "bandwidth")
	flag.StringVar(&ff.f, "f", "", "file path")
	flag.StringVar(&ff.s, "s", "127.0.0.1:8282", "server address")
	flag.BoolVar(&ff.h, "h", false, "print this help message")
}

func clientMain() error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos: []string{"quic-echo-example"},
	}
	config := &quic.Config {
		MaxIdleTimeout: time.Duration(10000) * time.Millisecond,
	}
	session, err := quic.DialAddr(ff.s, tlsConf, config)
	if err != nil {
		return err
	}

	stream, err := session.OpenStreamSync(context.Background())
	if err != nil {
		return err
	}

	sNow := time.Now().Unix()

	if ff.b > 0 {
		// set bandwidth
		fileRequest := &merry_proto.FileRequest{
			Offset: 0,
			Bandwidth: int32(ff.b),
		}
		err = common.RpcSendFileRequest(fileRequest, stream)
		if err != nil {
			fmt.Println(err)
			return err
		}

		fileResponse := &merry_proto.FileResponse{}
		err = common.RpcReadFileResponse(fileResponse, stream)
		if err != nil {
			fmt.Println(err)
			return err
		}

		fmt.Printf("set bandwidth: %s\n", fileResponse.GetStatus())
	} else {
		// send file request
		fileName := ff.f
		fileRequest := &merry_proto.FileRequest{
			Path: fileName,
			Offset: 0,
			Bandwidth: -1,
		}
		err = common.RpcSendFileRequest(fileRequest, stream)
		if err != nil {
			fmt.Println(err)
			return err
		}

		fileResponse := &merry_proto.FileResponse{}
		err = common.RpcReadFileResponse(fileResponse, stream)
		if err != nil {
			fmt.Println(err)
			return err
		}

		if fileResponse.GetSize() > 0 {
			localFieName := path.Base(fileName)
			err = common.RpcReadFileChunk(localFieName, stream)
			if err != nil {
				fmt.Println(err)
				return err
			}
		}
		fmt.Printf("get file: %d\n", fileResponse.GetSize())
	}

	eNow := time.Now().Unix()
	fmt.Printf("time: %d\n", eNow - sNow)

	return nil
}

func main() {
	flag.Parse()
	if ff.h {
		flag.Usage()
		return
	}

	err := clientMain()
	if err != nil {
		panic(err)
	}
}