# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

## [v0.0.4](https://github.com/Duke1616/ework-runner/releases/tag/v0.0.4) - 2026-03-30

- [`a9e3531`](https://github.com/Duke1616/ework-runner/commit/a9e353104674109fe2dc60a0c1b2f5dbaa13f47a) fix: 执行出现错误，修改 exection 里面的状态为 Faield
- [`77029f0`](https://github.com/Duke1616/ework-runner/commit/77029f006b02c5262ab710f13f9280805d88d320) chore: web 调用 添加 headers 传递
- [`566a758`](https://github.com/Duke1616/ework-runner/commit/566a758fc4158419024a370f4a312ba8428dba74) chore:  去除无用的 metadata 传递
- [`edd4923`](https://github.com/Duke1616/ework-runner/commit/edd492341a9eb8da08ab32445a69f9737e16e4b3) fix: vo 层数据转换
- [`dedb262`](https://github.com/Duke1616/ework-runner/commit/dedb262043643082f32755968e8bcd0a76b4c622) chore: 方法封装，提供 Reslove 方法
- [`32369de`](https://github.com/Duke1616/ework-runner/commit/32369deafb65c4cd8478662b0ff659f15f771a7d) chore: 调度
- [`1aa79f2`](https://github.com/Duke1616/ework-runner/commit/1aa79f2224dce012831ccd62ff3eacc3aa643c74) chore: 新增任务管理接口
- [`7c0da08`](https://github.com/Duke1616/ework-runner/commit/7c0da08e01cda095e34d742848e73dd9a0945c03) chore:  修改 corsHdl, 跨域配置
- [`6892ca8`](https://github.com/Duke1616/ework-runner/commit/6892ca8f70d74b4fb3436ee6a64a27a9e70e8e19) chore: 升级 ecmdb
- [`b961b81`](https://github.com/Duke1616/ework-runner/commit/b961b812301be3158638f66c9da9798ae346fd62) chore: web 层 固定 biz_id
- [`4259f39`](https://github.com/Duke1616/ework-runner/commit/4259f39c079dce315f6e86cbdcaf42e4c549907f) chore: 处理 clients 的逻辑
- [`aa67039`](https://github.com/Duke1616/ework-runner/commit/aa67039a916ba199df0c248dbb8b7cf665c5bdb2) chore: 新增 biz_id 的 grpc 处理
- [`0511eed`](https://github.com/Duke1616/ework-runner/commit/0511eed440d306ec312fe0babd9c0a25d7e982a9) fix: 调整断言
- [`76fa463`](https://github.com/Duke1616/ework-runner/commit/76fa463e7bda4cd8dde8448ff2d81523a21d9332) fix: 配置 mapstructure 解析填写错误
- [`ac09c47`](https://github.com/Duke1616/ework-runner/commit/ac09c47f05a48cb9a7f0725543ad327ee714efa2) chore: 任务管理，CRUD 开发
- [`21b54a8`](https://github.com/Duke1616/ework-runner/commit/21b54a8ee3205f433a36ac970f183f44799a801d) chore: 新增 set biz 方法
- [`56a0f15`](https://github.com/Duke1616/ework-runner/commit/56a0f15ba30c5eb4460621b18f4b665c88de752b) chore: task 新增 list 方法
- [`7b48f8d`](https://github.com/Duke1616/ework-runner/commit/7b48f8d1e56ac51402df12fc67fdc790b3e15a25) chore: 添加 debug.sh, 调试 task 任务
- [`2eda24e`](https://github.com/Duke1616/ework-runner/commit/2eda24e63c44a4f8249b3bed58e7c382357c256c) chore: 异常拦截
- [`0b4f338`](https://github.com/Duke1616/ework-runner/commit/0b4f3385fbef4e49d654804cd4284d2c54bc4c9e) fix: agent 注册的 registry 错误，粗暴解决
- [`5b4397f`](https://github.com/Duke1616/ework-runner/commit/5b4397fed494649c402ae40e8764b0bda2642604) fix: 修复 task result 消息未保存成功
- [`b6a57ea`](https://github.com/Duke1616/ework-runner/commit/b6a57ea4bb09ecc1aa55fbb0a0cc85d91ca26850) fix: ci 报错
- [`e515f87`](https://github.com/Duke1616/ework-runner/commit/e515f877bdd17add1d4b672e1d9ae88e27074bc8) chore: 修复 serviceName 强制依赖
- [`82a27a3`](https://github.com/Duke1616/ework-runner/commit/82a27a378c3e578895b3517e305b86540a341f94) chore: 去除 executor 启动 db 依赖
- [`56af2b4`](https://github.com/Duke1616/ework-runner/commit/56af2b4d5a69ebdabbde2971280d3d5f21a47ee7) chore: rpc 错误返回
- [`02ce194`](https://github.com/Duke1616/ework-runner/commit/02ce194f38888a2c9b6f525b22bafd0b8e274fa4) chore: 新增 retry 接口
- [`b054585`](https://github.com/Duke1616/ework-runner/commit/b0545856acae064c90d337d2df0cf185fea15c38) fix: 修复 schduelr 未进行 etcd 注册
- [`fc73d17`](https://github.com/Duke1616/ework-runner/commit/fc73d17307812c50a93a09836327b642ac811bc6) refactor: 优化代码，通过插件式加载
- [`270f37f`](https://github.com/Duke1616/ework-runner/commit/270f37fbf46dc0906adaef342aed510536f8fb5e) chore: wire 注入
- [`bc813b3`](https://github.com/Duke1616/ework-runner/commit/bc813b3629f8f6876fecde23132687de8b1b0bfb) fix: scheduler 模式单独运行，对 agent 模式存在依赖关系
- [`96f6396`](https://github.com/Duke1616/ework-runner/commit/96f6396672e1c008dec0bce866ce8bc84ab05da3) chore: 同步最新配置文件
- [`ac11f5a`](https://github.com/Duke1616/ework-runner/commit/ac11f5a8dda58ffb9b00fa1178fa7ac8156d54f3) chore: 调整获取 executor id 优先级
- [`e16055d`](https://github.com/Duke1616/ework-runner/commit/e16055dd7d6eb29d2d359ef55f3b2031c0008856) chore: 修改 exeutor 节点注册 id 规则
- [`69d51cb`](https://github.com/Duke1616/ework-runner/commit/69d51cb90d9026cd2f04654093ecc0874c981d29) fix: etcd registry 没有返回 instance 数据
- [`272adc1`](https://github.com/Duke1616/ework-runner/commit/272adc1a71ddfbd9eef8d5ba31f202cdeedf34c8) docs: 美观性优化
- [`a6eadd4`](https://github.com/Duke1616/ework-runner/commit/a6eadd4237f3fd301ab38a6a5704755cd3de5ac5) docs: 流程图展示效果
- [`9241e60`](https://github.com/Duke1616/ework-runner/commit/9241e6076d2fdde54d31df2fc6ff6fea35519faf) docs: 完善 readme 文档
- [`54bec5d`](https://github.com/Duke1616/ework-runner/commit/54bec5ded3f63e25cc339d8b5aae974e325e6df8) chore: 自定义 logger 失效
- [`cc0f06a`](https://github.com/Duke1616/ework-runner/commit/cc0f06ae0f572fc2aabf99a731bf980848dc7c76) refactor: 使用新版 SDK 进行登陆鉴权
- [`9b76624`](https://github.com/Duke1616/ework-runner/commit/9b76624b71e33cfa87250fef3e514489a29b4554) fix: mode scheduler、executor 启动启动状态下，agent 依旧会提前注册
- [`343cbf9`](https://github.com/Duke1616/ework-runner/commit/343cbf9c67b3a456e8befcf83572224f37a3c5c1) fix: 修复 python want_result 调用失败
- [`c1cb1ed`](https://github.com/Duke1616/ework-runner/commit/c1cb1ed0c6aa649040f9f5f38ae977c7e010aa73) fix: 修复 python want_result 调用失败
- [`274a184`](https://github.com/Duke1616/ework-runner/commit/274a184e6f6435ffc0964ed02ca6d2e521bf2de0) fix: 修复 agent web 注册 prefix 不一致，导致查询错误
- [`34c389a`](https://github.com/Duke1616/ework-runner/commit/34c389a8eb40437e8d596a789984ea7615d6698d) chore: 同步最新鉴权返回
- [`d398f85`](https://github.com/Duke1616/ework-runner/commit/d398f856a33d616f6a2480d3c60798366aa929d7) chore: 新增 setLogLevel 日志等级
- [`99bc94b`](https://github.com/Duke1616/ework-runner/commit/99bc94b3535c1f6d9bd382f1f64a1ce977482a3f) chore: check policy 日志信息调整
- [`0c2a9a0`](https://github.com/Duke1616/ework-runner/commit/0c2a9a039fc266de8c4e28ff31abaef2f3824d16) chore: 调整 web 权限
- [`5577557`](https://github.com/Duke1616/ework-runner/commit/557755742a008d7c4934b46b2e681924b68ff68a) chore: 修改 default 默认配置文件位置
