@echo off
REM 创建证书目录
mkdir certs 2>nul

REM 生成CA私钥
openssl genrsa -out certs/ca.key 2048

REM 生成CA证书
openssl req -new -x509 -days 3650 ^
    -key certs/ca.key ^
    -out certs/ca.crt ^
    -subj "/C=CN/ST=Beijing/L=Beijing/O=WechatBackup/OU=Dev/CN=Wechat Backup CA"

echo 证书生成完成！
pause