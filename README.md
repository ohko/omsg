# omsg
通过TCP建立连接通讯，解决拆包和粘包的问题。

# 使用
```
$ go get -u github.com/ohko/omsg
```

# Server
```
s = NewServer(onServerData, onNewClient, onServerClose)
s.StartServer("0.0.0.0:1234")

func onServerData(conn net.Conn, data []byte) {
	// 收到客户端数据
}

func onNewClient(conn net.Conn) {
	// 新的客户端连接
}

func onServerClose(conn net.Conn) {
	// 客户端断开
}
```

# Client
```
c = NewClient(onClientData, onClientClose)
if err := c.Connect("0.0.0.0:1234"); err != nil {
    log.Fatalln("[C] connect error:", err)
}
c.Send([]byte("hello"))

func onClientData(data []byte) {
	// 收到服务器数据
}

func onClientClose() {
	// 与服务器断开
}

```