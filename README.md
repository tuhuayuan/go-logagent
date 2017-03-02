# LogAgent 

## 依赖

包依赖管理工具http://glide.sh

curl https://glide.sh/get | sh

## 快速上手

glide up

glide install

go build

./logagent -sentinel -configs ./test/ -v 5

## 结构

采用可配置的插件结构

输入 -》 过滤器 -》输出

数据传输使用Go语言的channel

配置可以使用本地json文件，也可以使用ETCD

支持Sentinel模式，检测配置文件修改自动应用


## 插件 

输入
file
stdin
upd
http

过滤器
patch
grok

输出
stdout
redis
elastic

## 例子

参见 单元测试代码