#!/bin/bash

# 创建证书目录
mkdir -p certs

# 生成CA私钥
openssl genrsa -out certs/ca.key 2048

# 生成CA证书
openssl req -new -x509 -days 3650 \
    -key certs/ca.key \
    -out certs/ca.crt \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=WechatBackup/OU=Dev/CN=Wechat Backup CA"

# 设置权限
chmod 400 certs/ca.key
chmod 444 certs/ca.crt 