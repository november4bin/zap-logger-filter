# zap-logger-filter

zap-logger-filter 是一个基于  Zap 日志库的扩展，提供了强大的敏感数据过滤功能，能够在日志记录过程中自动检测和掩码敏感信息，保护用户隐私和安全数据。

## 功能特性

- **敏感数据自动过滤**：可配置敏感字段列表，自动检测并掩码敏感信息
- **大小写不敏感匹配**：对字段名的匹配不区分大小写
- **嵌套结构支持**：支持处理嵌套的 JSON 对象和数组中的敏感数据
- **多种日志输出**：支持控制台和文件输出
- **灵活的配置选项**：可配置日志级别、文件轮转策略等
- **多目标日志**：支持创建多个不同配置的日志记录器
- **高性能设计**：基于高性能的 Zap 日志库构建

## 安装

使用 Go 模块安装：

```bash
go get -u github.com/november4bin/zap-logger-filter
```

## 导入

```go
import (
    "github.com/november4bin/zap-logger-filter"
    "go.uber.org/zap"
)
```

## 使用方法

### 基本配置和初始化

```go
package main

import (
    "github.com/november4bin/zap-logger-filter"
)

func main() {
    // 定义配置
    configs := []zaploggerfilter.Config{
        {
            Type:            zaploggerfilter.Console,  // 控制台输出
            Name:            "console",               // 日志名称
            Level:           "info",                  // 日志级别
            SensitiveFilter: true,                     // 启用敏感数据过滤
            SensitiveFields: []string{"password", "token", "card", "secret"}, // 敏感字段列表
        },
        {
            Type:            zaploggerfilter.File,     // 文件输出
            Name:            "file",                  // 日志名称
            Level:           "debug",                 // 日志级别
            SensitiveFilter: true,                     // 启用敏感数据过滤
            SensitiveFields: []string{"password", "token"}, // 敏感字段列表
            Path:            "./logs/app.log",        // 日志文件路径
            MaxSize:         100,                      // 单个文件最大尺寸（MB）
            MaxAge:          7,                        // 最大保留天数
            MaxBackups:      5,                        // 最多保留文件数
            Compress:        true,                     // 是否压缩
        },
    }

    // 初始化日志
    zaploggerfilter.Init(configs)
    defer zaploggerfilter.Sync() // 确保日志被刷新

    // 使用全局日志记录器
    zaploggerfilter.L.Info("应用启动", zap.String("password", "secret123"))
    // 输出将显示: {"time":"2023-10-01T12:00:00Z","level":"info","logger":"","caller":"main.go:25","msg":"应用启动","password":"***"}

    // 使用指定名称的日志记录器
    zaploggerfilter.InfoTo("file", "用户登录", zap.String("token", "abc123xyz"))
}
```

### 添加新的日志记录器

```go
// 在运行时添加新的日志记录器
zaploggerfilter.AddTagetLogger(zaploggerfilter.Config{
    Type:   zaploggerfilter.Console,
    Name:   "newlogger",
    Level:  "warn",
})

// 使用新添加的日志记录器
zaploggerfilter.WarnTo("newlogger", "警告消息")
```

### 不同级别的日志记录

```go
// 全局日志记录器的不同级别
zaploggerfilter.L.Debug("调试信息")
zaploggerfilter.L.Info("一般信息")
zaploggerfilter.L.Warn("警告信息")
zaploggerfilter.L.Error("错误信息")
zaploggerfilter.L.Fatal("致命错误")
zaploggerfilter.L.Panic(" panic 信息")

// 指定目标的不同级别
zaploggerfilter.DebugTo("console", "调试信息")
zaploggerfilter.InfoTo("console", "一般信息")
zaploggerfilter.WarnTo("console", "警告信息")
zaploggerfilter.ErrorTo("console", "错误信息")
```

## 配置说明

`Config` 结构体包含以下字段：

- **Type**: 日志输出类型（Console 或 File）
- **Name**: 日志记录器名称，用于后续引用
- **Level**: 日志级别（debug, info, warn, error, panic, fatal）
- **SensitiveFilter**: 是否启用敏感数据过滤
- **SensitiveFields**: 需要过滤的敏感字段列表
- **Path**: 日志文件路径（仅对 File 类型有效）
- **MaxSize**: 单个日志文件最大尺寸（MB）（仅对 File 类型有效）
- **MaxAge**: 日志文件最大保留天数（仅对 File 类型有效）
- **MaxBackups**: 最多保留的日志文件数（仅对 File 类型有效）
- **Compress**: 是否压缩旧日志文件（仅对 File 类型有效）

## 自定义掩码字符串

可以自定义用于替换敏感数据的掩码字符串：

```go
zaploggerfilter.Mask = "[REDACTED]"
```

## 嵌套数据处理

敏感数据过滤器能够自动处理嵌套的 JSON 结构：

```go
userData := map[string]interface{}{
    "name": "John Doe",
    "credentials": map[string]interface{}{
        "password": "secret123",
        "token": "abc123",
    },
    "address": "123 Main St",
}

zaploggerfilter.L.Info("用户数据", zap.Any("user", userData))
// credentials.password 和 credentials.token 将被自动掩码
```

## 数组处理

敏感数据过滤器也能处理数组中的敏感信息：

```go
users := []map[string]interface{}{
    {"username": "user1", "password": "pass1"},
    {"username": "user2", "password": "pass2"},
}

zaploggerfilter.L.Info("用户列表", zap.Any("users", users))
// 所有 password 字段将被掩码
```

## 贡献指南

欢迎贡献代码！请遵循以下步骤：

1. Fork 本仓库
2. 创建您的特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交您的更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开一个 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详情请查看 [LICENSE](LICENSE) 文件