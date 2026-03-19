# LinuxDoSpace SDKs

本目录存放 LinuxDoSpace 邮件实时流的多语言 SDK。

除当前父仓库中的 `sdk/README.md` 与 `sdk/spec/` 外，其余每个语言目录都已经拆成独立 Git 仓库，并通过 submodule 挂回父仓库。

当前目标不是做一堆只会发 HTTP 请求的壳，而是统一实现这些核心语义：

- 一个 `Client` 只维护一条到 `/v1/token/email/stream` 的真实上游连接
- 客户端在本地解析完整邮件事件，再进行本地路由
- 精确绑定和正则绑定共享同一条有序匹配链
- `allow_overlap=false` 命中后立即停止
- `allow_overlap=true` 命中后继续向后分发
- 未开始 `listen()` 的 mailbox 不会积压历史消息
- 服务端只知道 Token，不知道本地注册了哪些 mailbox 规则

当前 SDK 列表：

- `python/`
- `rust/`
- `go/`
- `nodejs/`
- `c/`
- `cpp/`
- `kotlin/`
- `java/`
- `csharp/`
- `swift/`
- `dart/`
- `lua/`
- `zig/`

协议说明见 [spec/MAIL_STREAM_PROTOCOL.md](/G:/ClaudeProjects/LinuxDoSpace/LinuxDoSpace/sdk/spec/MAIL_STREAM_PROTOCOL.md)。

## 状态矩阵

| 语言 | 目录 | 当前状态 | 本机验证情况 |
| --- | --- | --- | --- |
| Python | `sdk/python` | 独立子仓库，功能最完整 | 子仓库已存在，当前 PyPI 安装命令是 `pip install linuxdospace` |
| Rust | `sdk/rust` | 完整 SDK | `cargo test` 通过 |
| Go | `sdk/go` | 完整 SDK | `go test ./...` 通过 |
| Node.js (TypeScript) | `sdk/nodejs` | 完整 SDK | `npm run build` 通过 |
| Java | `sdk/java` | 完整 SDK | `javac --release 21 ...` 通过 |
| Kotlin | `sdk/kotlin` | 完整 SDK | 当前环境缺少 `kotlinc`，未本地验证 |
| C# | `sdk/csharp` | 完整 SDK | 当前环境缺少 .NET SDK，未本地验证 |
| Swift | `sdk/swift` | 完整 SDK | 当前环境缺少 Swift toolchain，未本地验证 |
| Dart | `sdk/dart` | 完整 SDK | 当前环境缺少 Dart SDK，未本地验证 |
| Lua | `sdk/lua` | 完整 SDK | 当前环境缺少 Lua 运行时，未本地验证 |
| Zig | `sdk/zig` | 完整 SDK | 当前环境缺少 Zig toolchain，未本地验证 |
| C | `sdk/c` | 传输无关核心 SDK | `gcc` 编译与示例运行通过 |
| C++ | `sdk/cpp` | 传输无关核心 SDK | `g++` 编译与示例运行通过 |

## 说明

- Python SDK 是独立 Git 子仓库，父仓库只跟踪 submodule 指针。
- 其他语言 SDK 也都已经拆成各自独立 Git 仓库，父仓库只保存 gitlink 与 `.gitmodules` 配置。
- C 与 C++ 当前实现的是“传输无关核心”，通过 `ingestNdjsonLine` 接入上层 HTTPS 流。
- 其余语言都按照 `MAIL_STREAM_PROTOCOL` 提供真实的 SDK 工程骨架与公开 API。
