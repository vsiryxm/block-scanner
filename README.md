
# Block-Scanner

Block-Scanner是一个区块扫描程序，主要功能是将以太坊区块链中的满足条件的交易哈希采集到到MQ队列中，供其他服务消费。

#### 功能
- 支持多线程采集；
- 支持从指定区块号开始采集交易；
- 支持从指定多个智能合约地址中采集交易；
- 支持重启服务后从上次采集处继续采集交易；
- 支持按交易时间升序排列交易哈希并存储。

#### 使用

1、修改配置文件

将`config/config.yaml.example`修改成`config/config.yaml`，填写以太坊节点提供者的URL、数据库连接信息、要监控的智能合约地址列表、指定开始扫描的区块号等。

2、运行
```
go run ./main.go
```

3、编译
```
// linux
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o block-scanner
```

4、启动
```
chmod +x ./block-scanner
nohup ./block-scanner > logs/block-scanner.log 2>&1 &
```

#### 打赏作者

如果对你有所帮助，可以打赏一下哦

![](./data/me.jpg)


