ubuntu环境部署步骤：

1. 安装etcd,启动
apt install etcd
修改etcd配置
/etc/default/etcd
// 开通外部连接
ETCD_LISTEN_CLIENT_URLS="http://0.0.0.0:2379"
ETCD_ADVERTISE_CLIENT_URLS="http://0.0.0.0:2379"
重启服务
systemctl restart etcd

2. 安装redis,启动
apt install redis
修改配置
/etc/redis/redis.conf
// 需要外部访问
将protected-mode改为no
将bind 127.0.0.1注释掉
重启服务
systemctl restart redis

3. 安装nats-server,启动
下载nats release包后执行即可

4. 执行scipts目录build_linux.sh编译
5. 修改配置参数，主要是连接的etcd,nats,redis等地址
6. 启动服务器：执行scipts目录allStart.sh运行
7. 结束服务器：执行scipts目录allStop.sh运行