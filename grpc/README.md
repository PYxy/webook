
```bash
etcd 启动
docker run -d --name ai-etcd --network=host --restart always \
 -v $PWD/etcd.conf.yml:/opt/bitnami/etcd/conf/etcd.conf.yml \
 -v $PWD/data:/opt/binami/etcd/data \
 -e ETCD_ADVERTISE_CLIENT_URLS=http://{你服务器IP}:2379 \
 -e ETCD_LISTEN_CLIENT_URLS=http://0.0.0.0:2379 \
 -e ALLOW_NONE_AUTHENTICATION=yes \
 -e ETCD_CONFIG_FILE=/opt/bitnami/etcd/conf/etcd.conf.yml \
 -e ETCD_DATA_DIR=/opt/binami/etcd/data \
 bitnami/etcd:3.5.9


 docker run -d --name etcd --network=host --restart always \
 -e ETCD_ADVERTISE_CLIENT_URLS=http://120.132.118.90:2379 \
 -e ETCD_LISTEN_CLIENT_URLS=http://0.0.0.0:2379 \
 -e ALLOW_NONE_AUTHENTICATION=yes \
 bitnami/etcd:3.5.9


# docker exec -it etcd /bin/bash
启动服务端 grpc_etcd_test.go

$ etcdctl get --prefix service
service/user/127.0.0.1:8090
{"Op":0,"Addr":"127.0.0.1:8090","Metadata":{"time":"2023-12-29 16:30:31.6171779 +0800 +08 m=+40.277483101","weight":200}}

consoul 启动
docker run \
    -d \
    -p 8500:8500 \
    -p 8600:8600/udp \
    --name=badger \
    consul:1.15 agent -server -ui -node=server-1 -bootstrap-expect=1 -client=0.0.0.0
```