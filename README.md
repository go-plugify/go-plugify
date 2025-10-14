# go-plugify

go-plugify is a lightweight framework that leverages Golang plugins to enable hot-patching and remote execution of code.
With go-plugify, you can compile only the changed parts of your program locally, upload them, and execute them directly in the remote environment.

This approach helps you:

⚡ Reduce compile & deployment waiting time

🛠 Fix issues faster in production/test environments

🔌 Easily extend functionality via plugins

## ✨ Features

Hot Update via Plugin: Compile small pieces of code locally and load them remotely without restarting.

Remote Execution: Inject and run uploaded methods/functions in the target environment.

Faster Debug & Fix Cycles: Quickly validate bugfixes or new logic online.

Simple Integration: Drop into existing Go projects with minimal setup.

## 🚀 Getting Started
1. Install
go get github.com/chenhg5/go-plugify

2. Build a Plugin
package main

import "fmt"

// Exported symbol must be capitalized
func Patch() {
    fmt.Println("Hello from plugin!")
}


Compile the plugin:

go build -buildmode=plugin -o patch.so patch.go

3. Upload & Execute

Use go-plugify to upload the plugin file (patch.so) to the remote server and execute:

import "github.com/yourname/go-plugify"

func main() {
    client := gportal.NewClient("remote-address")
    err := client.UploadAndRun("patch.so", "Patch")
    if err != nil {
        panic(err)
    }
}

## 📌 Use Cases

Hotfix in production/test environments without full redeploy

Dynamic feature experiments by loading optional business logic

Development acceleration: reduce compilation time in large codebases

## ⚠️ Notes

Golang plugin support is Linux-only (official Go limitation).

The plugin must be compiled with the same Go version and architecture as the target environment.

## 📄 License

MIT License. See LICENSE
 for details.