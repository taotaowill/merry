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
	"strings"
	"os"
)

type Flag struct {
	b int
	s string
	f string
	o string
	h bool
}

var ff = Flag{}

func init() {
	flag.IntVar(&ff.b, "b", -1, "bandwidth")
	flag.StringVar(&ff.s, "s", "127.0.0.1:8282", "server address")
	flag.StringVar(&ff.f, "f", "", "source file path")
	flag.StringVar(&ff.o, "o", "", "target file path")
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
		remoteFileName := ff.f
		localFieName := path.Base(remoteFileName)
		if ff.o != "" {
			if strings.HasSuffix(ff.o, "/") {
				localFieName = ff.o + localFieName
			} else {
				localFieName = ff.o
			}
		}

		dirName := path.Dir(localFieName)
		err = os.MkdirAll(dirName, 0755)
		if err != nil {
			return err
		}

		fileInfo, err := os.Stat(localFieName)
		var f *os.File
		offset := int64(0)
		if os.IsNotExist(err) {
			f, err = os.OpenFile(localFieName, os.O_CREATE | os.O_WRONLY | os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
		} else {
			offset = fileInfo.Size()
			f, err = os.OpenFile(localFieName, os.O_CREATE | os.O_WRONLY, 0755)
			if err != nil {
				return err
			}
		}
		f.Seek(offset, 0)
		defer f.Close()

		fileRequest := &merry_proto.FileRequest{
			Path:      remoteFileName,
			Offset:    offset,
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
			err = common.RpcReadFileChunk(f, stream)
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