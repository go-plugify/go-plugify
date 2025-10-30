<a name="readme-top"></a>

<h1 align="center">
  <a href="https://github.com/go-plugify/go-plugify">
    <picture>
        <source media="(prefers-color-scheme: dark)" srcset="https://github.com/go-plugify/go-plugify/blob/main/docs/images/logos/logo-dark.png?raw=true">
        <source media="(prefers-color-scheme: light)" srcset="https://github.com/go-plugify/go-plugify/blob/main/docs/images/logos/logo-light.png?raw=true">
        <img alt="go-plugify logo" src="https://github.com/go-plugify/go-plugify/blob/main/docs/images/logos/logo-light.png?raw=true" width="351">
    </picture>
  </a>
</h1>

<div align="center">
  一个优雅易用的golang插件框架
  <br />
  <a href="https://github.com/go-plugify/go-plugify/issues/new?assignees=&labels=bug&template=bug_report_zh.md">提BUG</a>
  ·
  <a href="https://github.com/go-plugify/go-plugify/issues/new?assignees=&labels=enhancement&template=proposal_zh.md">提建议</a>
  .
  <a href="https://github.com/go-plugify/go-plugify/issues/new?assignees=&labels=question&template=question_zh.md">问问题</a>
</div>

<br />

</div>

<details open="open">
<summary>目录</summary>

- [介绍](#介绍)
  - [特性](#特性)
- [快速上手](#快速上手)
  - [依赖](#依赖)
  - [上手](#上手)
  - [例子](#例子)
- [文档](#文档)
- [证书](#证书)

</details>

---

## 介绍

golang毫无疑问是近年来很成功且广泛应用的语言，他的诸多优点使得其被广泛用于网页以及手机应用的后端程序中。go-plugify是基于golang的插件框架，其本身利用了golang的原生plugin能力。
使用 go-plugify 可以帮助您快速实现很多很棒的特性，解决很多开发上的问题，您不需要再为一个小修补而编译部署整个程序。您可以把功能修复验证的时间从小时分钟级缩减为秒级。而这只是其诸多能解决的问题之一。

注意，目前 go-plugify 仍在迭代开发中，不建议用在生产环境。

### 特性

- 通过插件热更新：在本地编译小片段代码，并在远程加载，无需重启。

- 远程执行：在目标环境中注入并运行上传的方法或函数。

- 更快的调试与修复周期：可在线快速验证修复或新逻辑。

- 简单集成：可轻松接入现有的 Go 项目，几乎无需额外配置。

## 快速上手

### 依赖

- 服务端程序需要cgo的支持，编译时加上：CGO_ENABLED=true
- 程序需要在linux或mac上运行

### 上手

#### 1. 安装命令行工具

```
go install github.com/go-plugify/plugcli
```

#### 2. 新建脚手架

```
plugcli create myplugin
```

#### 3. 编写自己的插件

##### 3.1 客户端代码

打开`plugin.go`文件，编写：
```go
...
func (p Plugin) Run(args any) {
	ctx := args.(HttpContext)
	p.Logger().Info("Plugin %s is running, ctx %+v", p.Name, ctx)
	p.Component("ginengine").(HttpRouter).ReplaceHandler("GET", "/", func(ctx context.Context) {
		ctx.(HttpContext).JSON(200, map[string]string{"message": "Hello from plugin !!!"})
	})
	cal := p.Component("calculator").(Calculator)
	ctx.JSON(200, map[string]any{
		"message":      "Plugin executed successfully",
		"load pkg":     pkg.SayHello(),
		"1 + 5 * 5 = ": cal.Add(1, cal.Mul(5, 5)),
	})
}
...
```

##### 3.2 服务端

服务端是挂载端点，以接收插件的加载运行请求。可以根据自身的web框架来接入，如：

```go
...

import (
	...
	ginadapter "github.com/go-plugify/webadapters/gin"
	...
)

func main() {
	router := setupRouter()
	router.Run(":8080")
}

func setupRouter() *gin.Engine {
	r := gin.Default()
	ginrouters := ginadapter.NewHttpRouter(r)
	plugManager := goplugify.Init("default",
		goplugify.ComponentWithName("ginengine", ginrouters),
		goplugify.ComponentWithName("calculator", &Caclulator{}),
	)
	plugManager.RegisterRoutes(ginrouters, "/api/v1")
	return r
}
...
```

#### 4. 运行

服务端编译记得加上：`CGO_ENABLED=true`。

客户端运行，可以进入项目文件夹后执行：`make run`，但记得修改 `Makefile` 中的服务端地址。

### 例子

查看：https://github.com/go-plugify/example

<img alt="example" src="https://github.com/go-plugify/example/blob/main/example.gif?raw=true" width="651">

## 证书

本项目使用 **MIT license**.

了解更多，请访问： [LICENSE](LICENSE)

<p align="right">(<a href="#readme-top">back to top</a>)</p>
