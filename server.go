package geerpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int // 标记这是一个geerpc的请求
	CodeType    codec.Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodeType:    codec.GobType,
}

// Server rpc 服务
type Server struct {
}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

func (s *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error", err)
			return
		}
		// 接收连接处理连接
		go s.ServeConn(conn)
	}
}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

// ServeConn 实现服务连接之后的解码工作处理
func (s *Server) ServeConn(conn io.ReadWriteCloser) {
	// 解析Option,header,body等事项
	defer func() { _ = conn.Close() }()
	var opt Option

	// 从conn中解析Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error", err)
		return
	}

	// 校验标记,是否是一个geerpc请求
	if opt.MagicNumber != MagicNumber {
		log.Println("rpc server: invalid magic number")
		return
	}

	// 从options中获取编码类型
	f := codec.NewCodecFuncMap[opt.CodeType]
	if f != nil {
		log.Printf("rpc server: invalid code type: %s", opt.CodeType)
		return
	}

	// 监听服务
	s.serveCodec(f(conn))
}

var invalidRequest = struct{}{}

func (s *Server) serveCodec(c codec.Codec) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	// 并发处理请求
	for {
		req, err := s.readRequest(c)
		if err != nil {
			if req == nil {
				break
			}
			req.h.Error = err.Error()
			s.sendResponse(c, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		go s.handleRequest(c, req, sending, wg)
	}
	wg.Wait()
	_ = c.Close()
}

// 请求处理相关
type request struct {
	h            *codec.Header
	argv, replyv reflect.Value
}

// 从请求中读取请求头
func (s *Server) readRequestHeader(c codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := c.ReadHeader(&h); err != nil {
		if err != io.EOF && !errors.Is(err, io.ErrUnexpectedEOF) {
			log.Println("rpc server: read head error:", err)
			return nil, err
		}
	}

	return &h, nil
}

// 读取请求
func (s *Server) readRequest(c codec.Codec) (*request, error) {
	h, err := s.readRequestHeader(c)
	if err != nil {
		log.Println("rpc server: read head error:", err)
	}

	req := &request{h: h}
	// reflect.TypeOf("") 创建一个字符串类型的反射对象
	// reflect.New() 创建一个新的string类型的零值,并返回一个指向它的指针
	req.argv = reflect.New(reflect.TypeOf(""))

	// req.argv.Interface() 将reflect.Value类型转换成interface{}类型,校验是否可以被ReadBody调用
	if err = c.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv error:", err)
	}

	return req, nil
}

// 处理请求
func (s *Server) handleRequest(c codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("geerpc resp %d", req.h.Seq))
	s.sendResponse(c, req.h, req.replyv.Interface(), sending)
}

// 响应请求
func (s *Server) sendResponse(c codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()

	if err := c.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}
