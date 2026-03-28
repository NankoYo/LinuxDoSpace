# Mail Stream Protocol

本文档定义 LinuxDoSpace 邮件 Token SDK 的当前公共协议约束。

## 目标

该协议用于让客户端通过一个 API Token 建立一条长连接，持续接收服务器转发的邮件事件。

## 传输层

- 方法：`GET`
- 路径：`/v1/token/email/stream`
- 鉴权：`Authorization: Bearer <token>`
- 协议：远端必须使用 `HTTPS`
- 本地调试例外：`localhost`、`127.0.0.1`、`::1` 与 `*.localhost` 可使用 `HTTP`
- 响应内容类型：`application/x-ndjson; charset=utf-8`

## 数据格式

服务端按行持续输出 JSON，每一行都是一个独立事件。

示例：

```json
{"type":"ready","token_public_id":"tok123","owner_username":"testuser"}
{"type":"heartbeat"}
{"type":"mail","original_envelope_from":"bounce@example.com","original_recipients":["alice@testuser.linuxdo.space"],"received_at":"2026-03-20T10:11:12Z","raw_message_base64":"RnJvbTogLi4u"}
{"type":"mail","original_envelope_from":"bounce@example.com","original_recipients":["alice@testuser-mail.linuxdo.space"],"received_at":"2026-03-20T10:11:13Z","raw_message_base64":"RnJvbTogLi4u"}
{"type":"mail","original_envelope_from":"bounce@example.com","original_recipients":["alice@testuser-mailfoo.linuxdo.space"],"received_at":"2026-03-20T10:11:14Z","raw_message_base64":"RnJvbTogLi4u"}
```

## 事件类型

### `ready`

表示连接已经建立成功，当前 SDK 应忽略该事件或仅用于内部握手。

`owner_username` 是必需字段。SDK 使用它把第一方枚举后缀
`linuxdo.space` 解析成当前 token 拥有者的真实邮件命名空间后缀
`<owner_username>-mail.linuxdo.space`。

这一点在当前版本仍然保持为黑箱语义。`Suffix.linuxdo_space` 是第一方语义后缀，
SDK 必须在内部把它映射为当前 token 拥有者在 `linuxdo.space` 下可接收邮件的
第一方命名空间集合。当前至少包括：

- `<owner_username>-mail.linuxdo.space`
- `<owner_username>-mail<suffix_fragment>.linuxdo.space`
- 为兼容旧事件输入，也可能出现 `<owner_username>.linuxdo.space`

示例：

```json
{
  "type": "ready",
  "token_public_id": "tok123",
  "owner_username": "testuser"
}
```

### `heartbeat`

表示连接保活，SDK 应忽略该事件，不向最终用户暴露。

示例：

```json
{
  "type": "heartbeat"
}
```

### `mail`

表示收到一封真实邮件。

字段：

- `original_envelope_from`
- `original_recipients`
- `received_at`
- `raw_message_base64`

示例：

```json
{
  "type": "mail",
  "original_envelope_from": "bounce@example.com",
  "original_recipients": [
    "alice@testuser.linuxdo.space"
  ],
  "received_at": "2026-03-20T10:11:12Z",
  "raw_message_base64": "RnJvbTogU2VuZGVyIDxzZW5kZXJAZXhhbXBsZS5jb20+DQpUbzogUmVjZWl2ZXIgPGFsaWNlQGxpbnV4ZG8uc3BhY2U+DQpTdWJqZWN0OiBUZXN0DQoNCkhlbGxv"
}
```

当用户把 API token 选为邮箱泛解析 `*@<owner_username>-mail.linuxdo.space`
的转发目标时，`original_recipients` 也可能出现新的显式邮箱命名空间，
例如：

```json
{
  "type": "mail",
  "original_envelope_from": "bounce@example.com",
  "original_recipients": [
    "alice@testuser-mail.linuxdo.space"
  ],
  "received_at": "2026-03-20T10:11:13Z",
  "raw_message_base64": "RnJvbTogLi4u"
}
```

当客户端主动注册了动态 mail suffix 过滤列表后，`original_recipients`
还可能出现：

```json
{
  "type": "mail",
  "original_envelope_from": "bounce@example.com",
  "original_recipients": [
    "alice@testuser-mailfoo.linuxdo.space"
  ],
  "received_at": "2026-03-20T10:11:14Z",
  "raw_message_base64": "RnJvbTogLi4u"
}
```

## 动态邮箱后缀控制面

为了让服务端只接受当前活跃客户端真正需要的动态邮箱域，协议还定义了一条
辅助控制接口：

- 方法：`PUT`
- 路径：`/v1/token/email/filters`
- 鉴权：`Authorization: Bearer <token>`
- 请求体：`{"suffixes":["", "foo", "bar"]}`

说明：

- `suffixes` 不是完整域名，而是固定 `-mail` 后面的可选后缀片段
- `""` 表示 `<owner_username>-mail.linuxdo.space`
- `"foo"` 表示 `<owner_username>-mailfoo.linuxdo.space`
- 服务端会把这些片段和 `ready.owner_username` 结合，维护当前 token
  活跃的动态邮箱域过滤列表
- 当某个动态 `-mail<suffix>` 域没有被当前任何活跃 token stream 注册时，
  服务端应尽早拒绝，而不是继续执行更重的数据库路由流程

## SDK 公共行为

### 上游连接

- `Client` 创建时立即尝试建立连接
- 若初始连接失败，应直接报错
- 若连接在后续中断，SDK 可在后台重连
- 若鉴权失败，应停止并抛出认证错误

### 完整监听

- `client.listen(...)` 是完整接收接口
- 它应接收当前 Token 收到的每一封邮件

### 本地绑定

- 绑定基于邮箱地址的 local-part 与 suffix
- 第一方枚举后缀 `linuxdo.space` 是语义后缀，不是字面父域名
- SDK 必须在内部把它匹配到当前 token 拥有者在 `linuxdo.space` 下的第一方命名空间
- 当前 SDK 至少应自动兼容：
  - `<owner_username>-mail.linuxdo.space`
  - `<owner_username>-mail<suffix_fragment>.linuxdo.space`
- 对支持动态后缀 helper 的 SDK：
  - `Suffix.linuxdo_space` 表示 `<owner_username>-mail.linuxdo.space`
  - `Suffix.linuxdo_space.with_suffix("foo")` 表示 `<owner_username>-mailfoo.linuxdo.space`
- 用户代码不应被迫为了新邮件命名空间改写原有 `Suffix.linuxdo_space` 用法
- `prefix` 与 `pattern` 二选一
- `pattern` 使用全匹配，不是搜索匹配
- 同一 suffix 下的所有绑定共享一条创建顺序链

### 匹配顺序

- 精确绑定和正则绑定没有独立优先级
- 只按创建顺序匹配
- 某条绑定命中后：
  - 必定收到消息
  - 若 `allow_overlap=false`，则立即停止
  - 若 `allow_overlap=true`，则继续向后匹配

### 队列语义

- 绑定在 `bind(...)` 时立即注册
- mailbox 队列在 `listen()` 开始时才激活
- `listen()` 之前到达的消息不会回填
- 同一个 mailbox 同时只允许一个活动监听器

## 邮件解析

SDK 应至少产出这些公共字段：

- `address`
- `sender`
- `recipients`
- `received_at`
- `subject`
- `message_id`
- `date`
- `from_header`
- `to_header`
- `cc_header`
- `reply_to_header`
- `from_addresses`
- `to_addresses`
- `cc_addresses`
- `reply_to_addresses`
- `text`
- `html`
- `headers`
- `raw`
- `raw_bytes` 或等价二进制字段

说明：

- 各语言 SDK 的 MIME 深解析能力可以不同
- 但以上字段至少要有稳定的公开访问方式
- 当某些字段无法从当前邮件中解析时，应返回空值而不是伪造数据
