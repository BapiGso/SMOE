# 自定义

## webdav
 - /webdav/* 挂载 usr/，Basic Auth（usr/config.yaml 的 name 和 bcrypt 密码）

## comment notify
 - 新评论邮件提醒（Resend），邮件里带签名的审核链接

## validator
 - 校验器，用于校验前端发过来的req body

## pow
 - 接口防爆破的POW算法
 - [死磕共识算法|pow算法](https://juejin.cn/post/6933764793833226253)
 - Scrypt是内存依赖型的POW算法。第一个使用Scrypt算法的数字货币是Tenebrix，而后该算法被莱特币使用。

## logger
 - 将slog库封装为自定义的logger

## jwt
 - 暂时没用上的