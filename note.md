# 1.缓存淘汰策略

1.FIFO

&返回地址

*解引用指针类型的值

# 2.

缓存雪崩：缓存在同一时间失效，造成瞬间DB请求量大、压力剧增，引起雪崩

缓存击穿：一个存在的key在缓存过期的一刻，同时又有大量请求，这些请求会击穿到DB，造成瞬间DB请求量大，压力剧增

缓存穿透：请求查询一个不存在的数据

# 3当前问题

- [X]  回调数据库迁移到mysql
- [X]  定期处理过期缓存
- [X]  虽然已经有了节点管理，但是无法动态添加节点

在分布式系统中，**服务发现（Service Discovery）**通常指的是**客户端动态地找到提供某服务的存活实例（节点）的过程**。它的核心目标是让客户端知道当前有哪些健康的、可以响应的服务节点可用，而不是仅仅获取静态配置。

sudo service mysqld start

sudo systemctl stop etcd

goreman -f Procfile start

etcdctl get "GroupCache" --prefix

mysql -u root -p

# 4 note

键值对是怎么去除的？
