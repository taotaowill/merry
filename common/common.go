package common

import (
	"unsafe"
	"reflect"
	"merry/proto"
	"bytes"
	"io"
	"os"
	"github.com/golang/protobuf/proto"
	"github.com/lucas-clemente/quic-go"
	"crypto/tls"
	"crypto/rsa"
	"crypto/rand"
	"math/big"
	"encoding/pem"
	"crypto/x509"
	"bufio"
)

const headSize = int(unsafe.Sizeof(RpcHead{}))
const buffSize = 1024

type RpcHead struct {
	magic int
	size int
}

type SliceMock struct {
	addr uintptr
	len int
	cap int
}

func (obj *RpcHead) Encode() []byte {
	testBytes := &SliceMock{
		addr: uintptr(unsafe.Pointer(obj)),
		cap: headSize,
		len: headSize,
	}
	return *(*[]byte)(unsafe.Pointer(testBytes))
}

func Decode(b []byte) *RpcHead {
	return (*RpcHead)(unsafe.Pointer(
		(*reflect.SliceHeader)(unsafe.Pointer(&b)).Data,
	))
}

func RpcReadFileRequest(req *merry_proto.FileRequest, stream quic.Stream) error {
	headBuff := make([]byte, headSize)
	_, err := io.ReadAtLeast(stream, headBuff, headSize)
	if err != nil {
		return err
	}

	head := Decode(headBuff)
	bodyBuff := make([]byte, head.size)
	_, err = io.ReadAtLeast(stream, bodyBuff, head.size)
	if err != nil {
		return err
	}

	err = proto.Unmarshal(bodyBuff, req)
	if err != nil {
		return err
	}

	return nil
}

func RpcSendFileResponse(res *merry_proto.FileResponse, stream quic.Stream) error {
	bodyBuff, err := proto.Marshal(res)
	if err != nil {
		return err
	}

	head := &RpcHead{
		magic: 1,
		size: len(bodyBuff),
	}

	headBuff := head.Encode()
	_, err = stream.Write(headBuff)
	if err != nil {
		return err
	}

	_, err = stream.Write(bodyBuff)
	if err != nil {
		return err
	}

	return nil
}

func RpcSendFileChunk(ch chan int, f *os.File, stream quic.Stream) error {
	stat, err := f.Stat()
	if err != nil {
		return err
	}

	// send head
	head := &RpcHead{
		magic: 1,
		size:  int(stat.Size()),
	}
	headBuff := head.Encode()
	_, err = stream.Write(headBuff)
	if err != nil {
		return err
	}

	// send body
	bs := make([]byte, buffSize)
	for {
		<-ch
		//fmt.Printf("---: %d\n", time.Now().UnixNano())
		n, err := f.Read(bs)
		if n == 0 {
			break
		}

		if err != nil {
			return err
		}

		_, err = stream.Write(bs[:n])
		if err != nil {
			return err
		}

		//time.Sleep(time.Duration(1) * time.Millisecond)
	}

	return nil
}

func RpcSendFileRequest(req *merry_proto.FileRequest, stream quic.Stream) error {
	bodyBuff, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	head := &RpcHead{
		magic: 1,
		size: len(bodyBuff),
	}
	headBuff := head.Encode()
	var buffer bytes.Buffer
	buffer.Write(headBuff)
	buffer.Write(bodyBuff)
	buff := buffer.Bytes()
	_, err = stream.Write(buff)
	if err != nil {
		return err
	}
	return nil
}

func RpcReadFileResponse(res *merry_proto.FileResponse, stream quic.Stream) error {
	headBuff := make([]byte, headSize)
	_, err := io.ReadAtLeast(stream, headBuff, headSize)
	if err != nil {
		return err
	}

	head := Decode(headBuff)
	bodyBuff := make([]byte, head.size)
	_, err = io.ReadAtLeast(stream, bodyBuff, head.size)
	if err != nil {
		return err
	}

	err = proto.Unmarshal(bodyBuff, res)
	if err != nil {
		return err
	}

	return nil
}

func RpcReadFileChunk(fileName string, stream quic.Stream) error {
	headBuff := make([]byte, headSize)
	_, err := io.ReadAtLeast(stream, headBuff, headSize)
	if err != nil {
		return err
	}

	head := Decode(headBuff)
	bodyBuff := make([]byte, buffSize)
	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()

	count := 0
	for {
		n, err := stream.Read(bodyBuff)
		if err != nil {
			return err
		}

		if err != nil {
			return err
		}

		w.Write(bodyBuff[:n])
		count += n
		if count >= head.size {
			break
		}
	}

	return nil
}

func GenerateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-echo-example"},
	}
}
