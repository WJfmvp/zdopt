package Actor

// actor/network.go

import (
	"context"
	"runtime"
	"strconv"
	"sync"

	"github.com/xtaci/kcp-go"
)

type Message struct {
	Data []byte
}

// Parse 解析并保存接收到的数据
func (m *Message) Parse(data []byte) {
	m.Data = make([]byte, len(data))
	copy(m.Data, data)
}

// messagePool 全局消息对象池，避免频繁内存分配
var messagePool = sync.Pool{
	New: func() interface{} {
		return new(Message)
	},
}

// KCPConn 使用连接池优化网络层
type KCPConn struct {
	connPool sync.Pool        // 存储 *kcp.UDPSession 连接对象
	sessions sync.Map         // 存储会话（根据需要扩展）
	messages chan interface{} // 用于传递解析后的消息
	ctx      context.Context  // 上下文控制停止
}

// NewKCPConn 创建KCPConn实例，监听指定端口
func NewKCPConn(port int, ctx context.Context) *KCPConn {
	return &KCPConn{
		connPool: sync.Pool{
			New: func() interface{} {
				conn, err := kcp.DialWithOptions(":"+strconv.Itoa(port), nil, 10, 3)
				if err != nil {
					// 这里简单处理错误，实际使用中建议做更完善的错误处理
					panic(err)
				}
				return conn
			},
		},
		messages: make(chan interface{}, 1024),
		ctx:      ctx,
	}
}

// Start 启动网络工作协作
func (k *KCPConn) Start() {
	for i := 0; i < runtime.NumCPU(); i++ {
		go k.readWorker()
	}
}

func (k *KCPConn) readWorker() {
	for {
		select {
		case <-k.ctx.Done():
			return
		default:
			// 从连接池中获取连接，注意类型断言为 *kcp.UDPSession
			conn := k.connPool.Get().(*kcp.UDPSession)
			data := make([]byte, 4096)

			// 使用零拷贝优化读取数据
			n, err := conn.Read(data)
			if err != nil {
				// 读取失败，将连接放回连接池后继续
				k.connPool.Put(conn)
				continue
			}

			// 从消息池获取 Message 对象，避免频繁内存分配
			msg := messagePool.Get().(*Message)
			msg.Parse(data[:n])

			// 非阻塞方式将消息投递到消息通道
			select {
			case k.messages <- msg:
			default:
				// 通道满时，将消息对象返回对象池，快速失败
				messagePool.Put(msg)
			}

			// 将连接放回连接池
			k.connPool.Put(conn)
		}
	}
}
