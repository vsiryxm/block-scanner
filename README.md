
# Block-Scanner

Block-Scanner是一个区块扫描程序，主要功能是将以太坊区块链中的满足条件的交易哈希采集到本地数据库中，同时将交易哈希加入到MQ队列中，供其他服务消费。

#### 功能
- 支持多线程采集；
- 支持从指定区块号开始采集交易；
- 支持从指定多个智能合约地址中采集交易；
- 支持重启服务后从上次采集处继续采集交易；
- 支持按交易时间升序排列交易哈希并存储。

#### 使用

1、配置.env文件，`APP_ENV`取值：`dev`-测试环境，`prod`-生产环境

2、导入数据库`database/db_scanner.sql`

3、配置`config/config.toml`文件
   填写以太坊节点提供者的URL、数据库连接URL、要监控的智能合约地址列表、指定开始扫描的区块号等。

4、运行
```
go run ./main.go
```

5、编译
```
// linux
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o block-scanner
```

6、启动
```
chmod 755 ./block-scanner
nohup ./block-scanner > logs/block-scanner.log 2>&1 &
```