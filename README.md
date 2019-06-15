# NatProxy

NatProxy是一个方便、快捷的内网穿透工具，借助natproxy，你可以远程访问在家里的电脑。例如，通过NatProxy你可以做到：

- 远程访问在家里的笔记本电脑
- 将本地开发的网页发给朋友看
- 将局域网内的其他服务共享给别人
- 等等

## 如何使用NatProxy

### 下载

首先，我们需要下载NatProxy客户端，在 [这个页面]() 点击最新版本下载，下载完成之后，把它放到你想要的目录，如果是Linux/macOS用户，
记得添加可执行权限：

```bash
$ sudo chmod +x ./natproxy
```

为了方便在命令行里执行，你还可以把它添加到 `/usr/local/bin/` 下面：

```bash
$ sudo mv ./natproxy /usr/local/bin/
```


