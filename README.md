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
root@master:~/webook# helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
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


root@master:~/webook# kubectl get pod -n webook
NAME                              READY   STATUS    RESTARTS        AGE
webook-mysql-794c795b8b-qrwbg     1/1     Running   0               5h48m
webook-redis-7d88d5bc66-tqv67     1/1     Running   0               5h48m
webook-service-847cbf5b58-76fr4   1/1     Running   0               5h39m
webook-service-847cbf5b58-gjn8d   1/1     Running   0               5h39m

root@master:~/webook# kubectl get svc -n ingress-nginx
NAME                                 TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)                      AGE
ingress-nginx-controller             NodePort    10.101.121.95    <none>        80:32473/TCP,443:30044/TCP   34d
ingress-nginx-controller-admission   ClusterIP   10.102.220.156   <none>        443/TCP                      34d

```

#通过service的Nodeport 访问 webook 服务
![img_4.png](option_png%2Fimg_4.png)

![img_5.png](option_png%2Fimg_5.png)


#通过ingress 访问 weboolk 服务(本地需要修改hosts 文件)
![img_6.png](option_png%2Fimg_6.png)