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
- [QuickStart](#Quick Start)
  - [Dependencies](#Dependencies)
  - [GettingStarted](#Quick Start)
  - [Examples](#Examples)
- [License](#License)

</details>

---

## Introduction

Golang is undoubtedly one of the most successful and widely used programming languages in recent years. Its many advantages make it a popular choice for backend development in both web and mobile applications.

go-plugify is a plugin framework built on top of Golang, leveraging its native plugin capabilities.

With go-plugify, you can easily implement powerful features and solve many common development problems. For instance, instead of recompiling and redeploying an entire program for a minor patch, you can apply, test, and verify changes within seconds. And that’s just one of the problems it helps you solve.

### Features

- Hot Update via Plugin: Compile small pieces of code locally and load them remotely without restarting.

- Remote Execution: Inject and run uploaded methods/functions in the target environment.

- Faster Debug & Fix Cycles: Quickly validate bugfixes or new logic online.

- Simple Integration: Drop into existing Go projects with minimal setup.

## Quick Start

### Dependencies

- The server-side program requires cgo support — compile with: CGO_ENABLED=true

- Must run on Linux or macOS

### Getting Started

#### 1. Install the CLI tool

```
go install github.com/go-plugify/plugcli
```

#### 2. Create a new plugin scaffold

```
plugcli create myplugin
```

#### 3. Write your plugin

##### 3.1 Client-side Code

Open the `plugin.go` file and write your plugin logic:
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

##### 3.2 Server-side Code

The server provides endpoints to receive and load plugin requests.
You can integrate it with your own web framework, for example:

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

#### 4. Run

When compiling the server, remember to include: `CGO_ENABLED=true`

For the client, navigate into the project folder and run:

### Examples

See：https://github.com/go-plugify/example

## License

This project is licensed under the **MIT license**.

See [LICENSE](LICENSE) for more information.

<p align="right">(<a href="#readme-top">back to top</a>)</p>