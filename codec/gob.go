package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

/**
 * GobCodec
 * conn: 构建函数传入,通常是通过tcp或者Unix建立socket时得到的链接实例
 * buf: 防止阻塞创建的带缓冲的Writer
 */

type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

func (c GobCodec) Close() error {
	return c.conn.Close()
}

// ReadHeader 读取头部信息,用定义好的解码器解码
func (c GobCodec) ReadHeader(header *Header) error {
	return c.dec.Decode(header)
}

// ReadBody 读取body信息,用定义好的解码器解码
func (c GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

// 写信息到头部,用编码器编码再写入
func (c GobCodec) Write(header *Header, body interface{}) (err error) {
	// 如果defer中需要用到返回值,那返回不能只定义一个类型需要定义具体的变量
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()

	if err := c.enc.Encode(header); err != nil {
		log.Println("rpc codec: gob error encoding header: ", err)
		return err
	}

	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body: ", err)
		return err
	}

	return nil
}

var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}
