package connect

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/yanlong-li/hi-go-logger"
	"github.com/yanlong-li/hi-go-socket/connect"
	"github.com/yanlong-li/hi-go-socket/packet"
	"github.com/yanlong-li/hi-go-socket/route"
	baseStream "github.com/yanlong-li/hi-go-socket/stream"
	"github.com/yanlong-li/hi-go-socket/websocket/stream"
	"reflect"
)

// 处理每个连接
func (conn *WebSocketConnector) Connected() {

	//处理首次连接动作
	conn.ConnectedAction()
	// 处理连接断开后的动作
	defer conn.DisconnectAction()

	defer func() { // 必须要先声明defer，否则不能捕获到panic异常
		if err := recover(); err != nil {
			logger.Debug("一个连接发生异常", 0, err) // 这里的err其实就是panic传入的内容
		}
		conn.Disconnect()
	}()

	for {
		// 读取消息
		_, buf, err := conn.Conn.ReadMessage()

		if err != nil {
			logger.Debug("read:"+string(buf), 0, err)
			// 停止继续循环
			break
		}
		logger.Debug("recv: "+string(buf), 0)
		conn.HandleData(buf)
	}
}

func (conn *WebSocketConnector) Disconnect() {
	_ = conn.Conn.Close()
	logger.Debug("断开连接", 0)
}

// 处理数据包
func (conn *WebSocketConnector) HandleData(buf []byte) {
	// uint16 = 4 uint32 = 8 uint64 = 16
	var OpCodeType = packet.OpCodeLen * 2
	//监听动作
	if len(buf) >= int(OpCodeType) {
		OpCode, err := hex.DecodeString(string(buf[0:OpCodeType]))
		if err != nil {
			logger.Debug("获取动作错误", 0)
		} else {

			websocketPacketStream := stream.WebSocketPacketStream{}

			websocketPacketStream.OpCode = binary.LittleEndian.Uint32(OpCode)
			if websocketPacketStream.OpCode <= packet.ReservedCode {
				logger.Debug("OP码范围不正确", 0)
				return
			}

			websocketPacketStream.SetData(buf[OpCodeType:])
			if !conn.RecvAction(&websocketPacketStream) {
				return
			}

			f := route.Handle(conn.GetGroup(), websocketPacketStream.OpCode)
			if f != nil {
				in := websocketPacketStream.Unmarshal(f)
				in = append(in, reflect.ValueOf(conn))
				reflect.ValueOf(f).Call(in)
			} else {
				logger.Debug("未注册的包", 0, websocketPacketStream.OpCode)
			}
		}
	} else {
		logger.Debug("动作长度不足4位", 0)
	}
}

// 建立连接时
func (conn *WebSocketConnector) ConnectedAction() {
	go connect.Add(conn)

	f := route.Handle(conn.GetGroup(), packet.Connection)
	if f != nil {
		var in []reflect.Value
		in = append(in, reflect.ValueOf(conn))
		reflect.ValueOf(f).Call(in)
	} else {
		logger.Debug("没有设置连接成功动作:", 1)
	}
}

// 准备断开连接
func (conn *WebSocketConnector) DisconnectAction() {

	f := route.Handle(conn.GetGroup(), packet.Disconnection)
	if f != nil {
		//构造一个存放函数实参 Value 值的数纽
		var in []reflect.Value
		in = append(in, reflect.ValueOf(conn))
		reflect.ValueOf(f).Call(in)
	} else {
		logger.Debug("没有设置断开连接动作:", 1)
	}

	_ = conn.Conn.Close()

	go connect.Del(conn.GetId())

}

// 收到数据包时
func (conn *WebSocketConnector) RecvAction(bs baseStream.Interface) bool {
	f := route.Handle(conn.GetGroup(), packet.BeforeRecv)
	if f != nil {
		var in []reflect.Value
		in = append(in, reflect.ValueOf(bs))
		in = append(in, reflect.ValueOf(conn))
		result := reflect.ValueOf(f).Call(in)
		if len(result) >= 1 {
			return result[0].Bool()
		}
		return false
	} else {
		return true
	}
}

// 发送数据包
func (conn *WebSocketConnector) Send(PacketModel interface{}) error {
	ps := stream.WebSocketPacketStream{}
	ps.Marshal(conn.GetGroup(), PacketModel)
	data := ps.ToData()

	// 发送之前进行数据处理：加密、压缩
	f := route.Handle(conn.GetGroup(), packet.BeforeSending)
	if f != nil {
		var in []reflect.Value
		in = append(in, reflect.ValueOf(ps))
		in = append(in, reflect.ValueOf(conn))

		result := reflect.ValueOf(f).Call(in)
		if len(result) >= 1 {
			data = result[0].Bytes()
		}
	}

	err := conn.Conn.WriteMessage(2, data)
	if err != nil {
		logger.Debug("发送数据失败", 0, err)
	}
	return err
}

//广播数据包
func (conn *WebSocketConnector) Broadcast(model interface{}, yourself bool) {
	go connect.Broadcast(connect.BroadcastModel{Model: model, Conn: conn, Self: yourself})
}
