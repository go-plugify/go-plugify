<a name="readme-top"></a>

<a href="https://github.com/go-plugify/go-plugify/blob/main/README_CN.md">[中文介绍]</a>

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
  An elegant and easy-to-use Golang plugin framework
  <br />
  <a href="https://github.com/go-plugify/go-plugify/issues/new?assignees=&labels=bug&template=bug_report.md">Report a Bug</a>
  ·
  <a href="https://github.com/go-plugify/go-plugify/issues/new?assignees=&labels=enhancement&template=proposal.md">Request a Feature</a>
  .
  <a href="https://github.com/go-plugify/go-plugify/issues/new?assignees=&labels=question&template=question.md">Ask a Question</a>
</div>

<br />

</div>

<details open="open">
<summary>Table of contents</summary>

- [Introduction](#Introduction)
  - [Features](#Features)
- [Quick Start](#quick-start)
  - [Dependencies](#Dependencies)
  - [Getting Started](#getting-started)
  - [Examples](#Examples)
- [License](#License)

</details>

---

## Introduction

**go-plugify** is a plugin system framework based on Golang. It helps you quickly add plugin capabilities to your Golang applications.

With **go-plugify**, you can easily build a plugin system that enables rapid feature iteration and problem solving. You no longer need to recompile and redeploy your entire program for small patches. You can reduce the time for bug fixing and verification from hours or minutes down to just seconds.
It also allows you to build a pluggable plugin ecosystem, making it easy to explore and develop plugins that suit your needs.

> ⚠️ **Note:** go-plugify is still under active development and iteration. It is **not recommended** for production use yet.

### Features

* **Hot plugin updates**: Compile small code snippets locally and load them remotely — no restart required.
* **Remote execution**: Inject and run uploaded methods or functions in the target environment.
* **Faster debugging and patch cycles**: Quickly verify fixes or new logic online.
* **Easy integration**: Seamlessly integrate with existing Go projects — no complex setup needed.

---

## Getting Started

### Quick Start

#### 1. Install the CLI tool

```bash
go install github.com/go-plugify/plugcli
```

#### 2. Create a new scaffold

```bash
plugcli create myplugin
```

#### 3. Write your own plugin

The client supports both the `yaegi` mode and the native Golang `plugin` mode.
Using **yaegi** is generally recommended for simplicity, while the **native plugin** mode provides broader compatibility and extensibility.

If your logic is not too complex and doesn’t rely heavily on advanced Golang system packages, **yaegi** is a good choice.

Below is an example using **yaegi**.

---

##### 3.1 Client Code

Open `main.go` and write:

```go
package main

import (
	"context"

	// The plugify package represents dependencies exposed by the host program.
	// Locally, you can use `replace` in go.mod to map it to the actual package.
	"plugify/plugify"
)

// The following three functions must be implemented: Run, Methods, Destroy.

// Run is executed after the plugin is loaded — it’s the main entry point.
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

// Methods defines functions that can be called by the host.
func Methods() map[string]func(any) any {
	return map[string]func(any) any{
		"hello": func(input any) any {
			plugify.Logger.Info("Hello from the 'hello' method!")
			return "Hello, World!"
		},
	}
}

// Destroy is called when the plugin is unloaded, for cleanup and resource release.
func Destroy(input map[string]any) error {
	plugify.Logger.Info("Example plugin is being destroyed")
	return nil
}
```

---

##### 3.2 Server Side

On the server side, you need to initialize the system and mount API routes to handle plugin load and execution requests.
You can integrate it into your existing web framework. For example, using Gin:

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

	// Initialize plugin manager and attach dependencies
	plugManager := goplugify.InitPluginManagers("default",
		goplugify.ComponentWithName("ginengine", ginRouter),
		goplugify.ComponentWithName("bookService", bookService),
		goplugify.ComponentWithName("allKindBook", new(service.AllKindBook)),
	)

	registerCoreRoutes(r, plugManager, bookService)

	// Register service routes
	goplugify.InitHTTPServer(plugManager).RegisterRoutes(ginRouter, "/api/v1")

	return r
}
...
```

#### 4. Run

For the server (if using native Golang plugin mode), remember to compile with:

```bash
CGO_ENABLED=true
```

For the client, you can enter the project directory and run:

```bash
make init
```

For more command options, run:

```bash
make help
```

---

### Examples

For more detailed example projects, please visit:
👉 [https://github.com/go-plugify/example](https://github.com/go-plugify/example)

<img alt="example" src="https://github.com/go-plugify/example/blob/main/example.gif?raw=true" width="651">

## License

This project is licensed under the **MIT license**.

See [LICENSE](LICENSE) for more information.

<p align="right">(<a href="#readme-top">back to top</a>)</p>
