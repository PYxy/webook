# webook

# 注册
![img.png](option_png%2Fimg.png)
```Bash
mysql> select * from webook.users;
+----+----------------------+--------------------------------------------------------------+---------------+---------------+-----------+-----------+----------+
| id | email                | password                                                     | ctime         | utime         | nick_name | birth_day | describe |
+----+----------------------+--------------------------------------------------------------+---------------+---------------+-----------+-----------+----------+
|  2 | asxxxxxxxxxx@164.com | $2a$10$kCmkTGVNnH/wJeJP1l0T8eZba5JJNr2QE.hxlPnSLu8i3rBAnY6uK | 1691734608691 | 1691734608691 |           |           |          |
+----+----------------------+--------------------------------------------------------------+---------------+---------------+-----------+-----------+----------+

```

# 登录
![img_1.png](option_png%2Fimg_1.png)

# 修改个人信息
![img_2.png](option_png%2Fimg_2.png)
```Bash
mysql> select * from webook.users;
+----+----------------------+--------------------------------------------------------------+---------------+---------------+-----------+------------+----------+
| id | email                | password                                                     | ctime         | utime         | nick_name | birth_day  | describe |
+----+----------------------+--------------------------------------------------------------+---------------+---------------+-----------+------------+----------+
|  2 | asxxxxxxxxxx@164.com | $2a$10$kCmkTGVNnH/wJeJP1l0T8eZba5JJNr2QE.hxlPnSLu8i3rBAnY6uK | 1691734608691 | 1691734762641 | xcd       | 1992-01-01 | ccccc2   |
+----+----------------------+--------------------------------------------------------------+---------------+---------------+-----------+------------+----------+
```

# 获取个人信息
![img_3.png](option_png%2Fimg_3.png)


# webook k8s 部署


```Bash
#登录docker hub
root@master:~/webook# docker login
Login with your Docker ID to push and pull images from Docker Hub. If you don't have a Docker ID, head over to https://hub.docker.com to create one.
Username: ljy2022
Password: 
WARNING! Your password will be stored unencrypted in /root/.docker/config.json.
Configure a credential helper to remove this warning. See
https://docs.docker.com/engine/reference/commandline/login/#credentials-store

Login Succeeded

#镜像打包
root@master:~/webook# docker build -f dockerfile -t webook:0.2 .
root@master:~/webook# docker images
REPOSITORY                                            TAG           IMAGE ID       CREATED         SIZE
webook                                                0.2           4f955df859d9   3 minutes ago   12.5MB

#镜像上传
root@master:~/webook# docker tag 4f955df859d9 ljy2022/webook:0.2
root@master:~/webook# docker push ljy2022/webook:0.2
The push refers to repository [docker.io/ljy2022/webook]
523d0f493eb3: Pushed 
0.2: digest: sha256:49f2b1c4a1cfd3d64373bfb2b2faf921a0a483e6de46cd74499331e1ee3e08db size: 527


# on_k8s 文件夹在k8s 集群部署相应的服务
#helm 安装
curl https://baltocdn.com/helm/signing.asc | sudo apt-key add -
sudo apt-get install apt-transport-https --yes
echo "deb https://baltocdn.com/helm/stable/debian/ all main" | sudo tee /etc/apt/sources.list.d/helm-stable-debian.list
sudo apt-get update
sudo apt-get install helm

#安装ingress

root@master:~/webook/on_k8s# ll
total 52
drwxr-xr-x 3 root root 4096 Aug 18 06:43 ./
drwxr-xr-x 8 root root 4096 Aug 18 00:48 ../
-rw-r--r-- 1 root root 1281 Aug 17 07:18 mysql-deploy.yaml 
-rw-r--r-- 1 root root  208 Aug 17 07:35 mysql-pvc.yaml
-rw-r--r-- 1 root root  219 Aug 17 08:33 mysql-pv.yaml
-rw-r--r-- 1 root root  247 Aug 18 02:39 mysql-service.yaml
-rw-r--r-- 1 root root  594 Aug 17 08:25 redis-deploy.yaml
-rw-r--r-- 1 root root  247 Aug 18 02:57 redis-service.yaml
-rw-r--r-- 1 root root  595 Aug 18 03:08 webook-deploy.yaml
-rw-r--r-- 1 root root  609 Aug 18 05:55 webook-ingress.yaml
-rw-r--r-- 1 root root  276 Aug 18 01:49 webook-pod.yaml
-rw-r--r-- 1 root root  227 Aug 18 06:43 webook-service.yaml


#全部执行一篇
root@master:~/webook# kubectl create ns webook
NAME                   TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)          AGE
webook-mysql-service   NodePort    10.110.101.202   <none>        3308:30002/TCP   5h47m
webook-pod-service     ClusterIP   10.99.92.44      <none>        8081/TCP         128m
webook-redis-service   NodePort    10.108.251.69    <none>        6380:30003/TCP   5h47m

root@master:~/webook# kubectl get ingress -n webook
NAME                  CLASS   HOSTS             ADDRESS        PORTS   AGE
webook-live-ingress   nginx   ljy.ingress.com   10.108.21.42   80      79m

root@master:~/webook# kubectl get pod -n webook
NAME                              READY   STATUS    RESTARTS        AGE
webook-mysql-794c795b8b-qrwbg     1/1     Running   0               5h48m
webook-redis-7d88d5bc66-tqv67     1/1     Running   0               5h48m
webook-service-847cbf5b58-76fr4   1/1     Running   0               5h39m
webook-service-847cbf5b58-gjn8d   1/1     Running   0               5h39m

root@master:~/webook# kubectl get svc -n default
NAME                                 TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)          AGE
ingress-nginx-controller             ClusterIP   10.108.21.42   <none>        80/TCP,443/TCP   3h49m
ingress-nginx-controller-admission   ClusterIP   10.100.65.87   <none>        443/TCP          3h49m
kubernetes                           ClusterIP   10.96.0.1      <none>        443/TCP          40d


```

#通过service的Nodeport 访问 webook 服务
![img_4.png](option_png%2Fimg_4.png)

![img_5.png](option_png%2Fimg_5.png)


#通过ingress 访问 weboolk 服务
```bash
[root@HZ1LY_Mysql_Manage ~]# curl -s -XPOST -H "Host: ljy.ingress.com" -H'Content-Type: application/json' -d '{"email":"asxxxxxxxxxx@163.com","password":"xxxx@1xxxxxxxxx"}' http://154.221.17.220/users/loginJWT -v -k
* About to connect() to 154.221.17.220 port 80 (#0)
*   Trying 154.221.17.220...
* Connected to 154.221.17.220 (154.221.17.220) port 80 (#0)
> POST /users/loginJWT HTTP/1.1
> User-Agent: curl/7.29.0
> Accept: */*
> Host: ljy.ingress.com
> Content-Type: application/json
> Content-Length: 61
> 
* upload completely sent off: 61 out of 61 bytes
< HTTP/1.1 200 OK
< Date: Sat, 19 Aug 2023 06:28:39 GMT
< Content-Type: text/plain; charset=utf-8
< Content-Length: 12
< Connection: keep-alive
< X-Jwt-Token: eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTI0MjY1NzksIlVpZCI6MSwiVXNlckFnZW50IjoiY3VybC83LjI5LjAifQ.8WoY03hPEwCK6gW5h4xXrpLIHbUILovfKVkZrceAqLs7Rpo6ElwkNQBqfHSrnGxpgd03LCh1FRWv-5eVuSuJww
< 
* Connection #0 to host 154.221.17.220 left intact
登录成功
```

wire
```bash
#wire 安装
go install github.com/google/wire/cmd/wire
go get github.com/google/wire/cmd/wire

wire.go 文件路径下

wire


项目启动命令: go run .\main.go .\wire_gen.go
```


mock
```bash
#linux
root@master:~/webook# mockgen -source=./internal/service/user.go  -package=svcmocks -destination=./internal/service/mocks/user_mock.go
root@master:~/webook# mockgen -source=./internal/service/code.go  -package=svcmocks -destination=./internal/service/mocks/code_mock.go
#window 
PS F:\git_push\webook\internal\web> mockgen -source=F:\git_push\webook\internal\service\user.go  -package=svcmocks -destination=F:\git_push\webook\internal\service\mocks\user_mock.go
PS F:\git_push\webook\internal\web> mockgen -source=F:\git_push\webook\internal\service\code.go  -package=svcmocks -destination=F:\git_push\webook\internal\service\mocks\code_mock.go


 
PS F:\git_push\webook\internal\web> mockgen -source=F:\git_push\webook\internal\repository\user.go  -package=svcmocks -destination=F:\git_push\webook\internal\repository\mocks\user_mock.go
PS F:\git_push\webook\internal\web> mockgen -source=F:\git_push\webook\internal\repository\code.go  -package=svcmocks -destination=F:\git_push\webook\internal\repository\mocks\code_mock.go

PS F:\git_push\webook\internal\web> mockgen -source=F:\git_push\webook\internal\repository\dao\type.go  -package=daomocks -destination=F:\git_push\webook\internal\repository\dao\mocks\user_mock.go
PS F:\git_push\webook\internal\web> mockgen -source=F:\git_push\webook\internal\repository\cache\type.go  -package=cachemocks -destination=F:\git_push\webook\internal\repository\cache\mocks\user_mock.go


#根据Cmdable interface  
PS F:\git_push\webook\internal\web> mockgen -source=D:\GOPro\pkg\mod\github.com\redis\go-redis\v9@v9.1.0\commands.go  -package=redismocks -destination=F:\git_push\webook\internal\repository\cache\redis\mocks\redis_mock.go

```

test
```bash

#测试当前测试文件中 能匹配 TestUserHandler_LoginJWT 字符串的函数(正则) -v 详细信息
go test -run="TestUserHandler_LoginJWT" -v

#当前测试文件的所有测试
go test -v

```


修改Jwt 登录设置成长短token+JWT
```bash
测试流程:
1.注册
http://127.0.0.1:8091/users/signup post 请求
{"email":"asxxxxxxxxxx@163.com","confirmPassword":"xxxx@1xxxxxxxxx","password":"xxxx@1xxxxxxxxx"} json格式

2.登录
http://127.0.0.1:8091/users/loginJWT post 请求
{"email":"asxxxxxxxxxx@163.com","password":"xxxx@1xxxxxxxxx"} json格式
查看 resp Headers
X-Jwt-Token      eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTUxMDkwMTUsIlVpZCI6MiwiU3NpZCI6IjVhZTlmZDlhLWY1MDEtNDMyZC05OTA3LTZiMDY2ZTc3ZDA2MCIsIlVzZXJBZ2VudCI6IlBvc3RtYW5SdW50aW1lLzcuMjYuOCJ9.BY8cL8iac8pRvAjAl-v-aonlZyPJU0QTjc5AnAE8E9c6guK7SX8tb7eOnytU5Lb2_G3xBMccJ8m7J9tld7KVeg
X-Refresh-Token  eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTU3MTIwMTUsIlVpZCI6MiwiU3NpZCI6IjVhZTlmZDlhLWY1MDEtNDMyZC05OTA3LTZiMDY2ZTc3ZDA2MCJ9.BVts4bsP-2DZEgGGhXpCOAKVHCQyE6FZI1vG17gyvXrRXNaBD5KJPy1U_EaXNH4UJLDM8Sv7VlUTX9V2J36XNQ

3.更新access_token
http://127.0.0.1:8091/users/refresh_token post 请求
req Headers 带参数   
              X-Refresh-Token
Authorization bearer eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTU3MTIzMTksIlVpZCI6MiwiU3NpZCI6IjY0Nzc0MTlhLTQ4MzItNGU5MC1hN2ZhLWI5Y2MwNzZjYTkxMiJ9.jGVEMTGd9Nyl2E9zjzcGo7vXX0vxDCxQUBXpqn0O5442rNaUd0Jmw_GLZCryUTpjt81VJjqVG4ETnWdIYNFcHA

重新获取搞 access_token
eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTUxMDkzMjksIlVpZCI6MiwiU3NpZCI6IjY0Nzc0MTlhLTQ4MzItNGU5MC1hN2ZhLWI5Y2MwNzZjYTkxMiIsIlVzZXJBZ2VudCI6IlBvc3RtYW5SdW50aW1lLzcuMjYuOCJ9.wkIbLPqnvbJ1OlK8jOwWB5hLUq6RWdgmZxnxXXVi1cYt7gSzz_Y8o-WEYdQ2ILp1pW5eem3YPibNC24BOki_ew

4.退出
http://127.0.0.1:8091/users/logoutJWT post 请求
req Headers 带参数  这个是access_token (如果带access_token  “提示未登录”，应该使用refresh token 请求 refresh_token接口)
Authorization bearer eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTUxMDkzMjksIlVpZCI6MiwiU3NpZCI6IjY0Nzc0MTlhLTQ4MzItNGU5MC1hN2ZhLWI5Y2MwNzZjYTkxMiIsIlVzZXJBZ2VudCI6IlBvc3RtYW5SdW50aW1lLzcuMjYuOCJ9.wkIbLPqnvbJ1OlK8jOwWB5hLUq6RWdgmZxnxXXVi1cYt7gSzz_Y8o-WEYdQ2ILp1pW5eem3YPibNC24BOki_ew


5.使用原来的refresh token 去更新access token


```