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

**go-plugify** 是基于golang的插件系统框架，他可以帮您将您的golang应用快速拥有插件能力。
使用 **go-plugify** 可以帮助您利用插件系统，快速实现很多很棒的特性，解决很多开发上的问题，您不需要再为一个小修补而编译部署整个程序。您可以把功能修复验证的时间从小时分钟级缩减为秒级。您还可以拥有可插拔的脚本插件生态，探索适合您的插件。

注意，目前 **go-plugify** 仍在迭代开发中，不建议用在生产环境。

### 特性

- 通过插件热更新：在本地编译小片段代码，并在远程加载，无需重启。

- 远程执行：在目标环境中注入并运行上传的方法或函数。

- 更快的调试与修复周期：可在线快速验证修复或新逻辑。

- 简单集成：可轻松接入现有的 Go 项目，无需额外复杂配置。

## 快速上手

### 上手

#### 1. 安装命令行工具

```
go install github.com/go-plugify/plugcli
```

#### 2. 新建脚手架

```
plugcli -l zh create myplugin
```

#### 3. 编写自己的插件

客户端支持 `yaegi` 与 原生golang plugin的模式，更推荐使用 `yaegi`，但原生的golang plugin会有更大的扩展与支持，如果您要编写的逻辑不是特别复杂不会用上很多比较复杂golang系统包函数的话，那么使用 `yaegi` 是合适的。

下面以 `yaegi` 为例子。

##### 3.1 客户端代码

打开`main.go`文件，编写：

```go
package main

import (
	"context"

	// plugify 包是宿主程序挂载的依赖包，在本地可以通过 replace 关联到对应的实际包
	"plugify/plugify"
)

// 必须实现下面三个函数，Run，Methods，Destroy

// Run 函数在加载后运行，是脚本的主要逻辑
func Run(input map[string]any) (any, error) {
	plugify.Logger.Info("Example plugin is running")
	plugify.Ginengine.ReplaceHandler("GET", "/", func(ctx context.Context) {
		plugify.Ginengine.NewHTTPContext(ctx).JSON(200, map[string]string{"message": "Hello from example plugin 2 !!!"})
	})
	plugify.BookService.AddBook(plugify.ServiceBook{ID: 1, Title: "The Great Gatsby", Author: "F. Scott Fitzgerald"})
	plugify.BookService.AddBook(plugify.ServiceBook{ID: 2, Title: "Pride and Prejudice", Author: "Jane Austen"})
	plugify.BookService.DeleteBook(1)
	plugify.Logger.Info("Books in the service: %+v", plugify.BookService.ListBooks())
	plugify.Logger.Info("Example plugin finished execution")
	return map[string]any{
		"message": "Plugin executed successfully",
		"books":   plugify.BookService.ListBooks(),
		"fictionBook": plugify.ServiceFictionBook{Book: plugify.ServiceBook{
			ID:     1,
			Title:  "Dune",
			Author: "Frank Herbert",
		}},
	}, nil
}

// Methods 函数返回内部的可供宿主函数调用的方法
func Methods() map[string]func(any) any {
	return map[string]func(any) any{
		"hello": func(input any) any {
			plugify.Logger.Info("Hello from the 'hello' method!")
			return "Hello, World!"
		},
	}
}

// Destroy 将会在 unload 时被调用，用于卸载后回收资源
func Destroy(input map[string]any) error {
	plugify.Logger.Info("Example plugin is being destroyed")
	return nil
}
```

##### 3.2 服务端

服务端需要初始化，挂载路由接口，以接收插件的加载运行请求。可以根据自身的web框架来接入，如：

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

	ginRouter := ginadapter.NewHttpRouter(r)

	bookService := service.NewBookService()

	// 初始化插件管理器，挂载相应的依赖组件
	plugManager := goplugify.InitPluginManagers("default",
		goplugify.ComponentWithName("ginengine", ginRouter),
		goplugify.ComponentWithName("bookService", bookService),
		goplugify.ComponentWithName("allKindBook", new(service.AllKindBook)),
	)

	registerCoreRoutes(r, plugManager, bookService)

	// 挂载服务路由
	goplugify.InitHTTPServer(plugManager).RegisterRoutes(ginRouter, "/api/v1")

	return r
}
...
```

#### 4. 运行

服务端，如果是原生golang plugin模式，编译记得加上：`CGO_ENABLED=true`。

客户端运行，可以进入项目文件夹后执行：`make init`。更多命令信息，执行：`make help`。

### 例子

更详细的例子说明，查看：https://github.com/go-plugify/example

<img alt="example" src="https://github.com/go-plugify/example/blob/main/example.gif?raw=true" width="651">

## 开发路线

- [ ] 支持插件常驻与持久化安装
- [ ] 接入服务发现能力，支持插件广播安装与运行
- [ ] 增加Hook能力，增加web框架全局中间件Hook节点，支持自定义增加Hook节点
- [ ] 完善客户端前端html控制台，支持插件管理与服务器节点管理等
- [ ] 支持插件自定义ui与接口

## 社区

添加微信：mongorz，备注：go-plugify 加入微信交流群。

## 证书

本项目使用 **MIT license**.

了解更多，请访问： [LICENSE](LICENSE)

<p align="right">(<a href="#readme-top">back to top</a>)</p>
