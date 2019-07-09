# rpc2d

#### 项目介绍
rpc2d 双向 RPC 调用，可以实现从服务器 CALLBACK 客户端 API，基于 "net/rpc" 原生库

use: SEE `test/server.go` 、 `test/client.go`

#### 软件架构
软件架构说明


#### 安装教程

1. xxxx
2. xxxx
3. xxxx

#### 使用说明

    type ProviderType
    type RpcNode
        func Accept(l net.Listener, provider interface{}) (*RpcNode, error)
        func NewRpcNode(provider interface{}) *RpcNode
        func NewRpcNodeByConn(provider interface{}, conn io.ReadWriteCloser) *RpcNode
        func (self *RpcNode) Close()
        func (self *RpcNode) Dial(addr string) error

#### 参与贡献

1. Fork 本项目
2. 新建 Feat_xxx 分支
3. 提交代码
4. 新建 Pull Request


#### 码云特技

1. 使用 Readme\_XXX.md 来支持不同的语言，例如 Readme\_en.md, Readme\_zh.md
2. 码云官方博客 [blog.gitee.com](https://blog.gitee.com)
3. 你可以 [https://gitee.com/explore](https://gitee.com/explore) 这个地址来了解码云上的优秀开源项目
4. [GVP](https://gitee.com/gvp) 全称是码云最有价值开源项目，是码云综合评定出的优秀开源项目
5. 码云官方提供的使用手册 [https://gitee.com/help](https://gitee.com/help)
6. 码云封面人物是一档用来展示码云会员风采的栏目 [https://gitee.com/gitee-stars/](https://gitee.com/gitee-stars/)