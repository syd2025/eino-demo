1. 常见问题

- 遇到网络问题：

# Windows PowerShell

`$env:HTTP_PROXY="http://127.0.0.1:7890"`
`$env:HTTPS_PROXY="http://127.0.0.1:7890"`

# Linux/MacOS

`export HTTP_PROXY="http://127.0.0.1:7890"`
`export HTTPS_PROXY="http://127.0.0.1:7890"`

- 模块下载慢

`go env -w GOPROXY=https://goproxy.cn,direct`

