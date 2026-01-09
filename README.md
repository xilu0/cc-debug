# cc-debug

Claude API 调试代理服务，用于拦截和记录 Claude 客户端请求。

## 功能

- 拦截所有 Claude API 请求
- 记录请求头、请求体到控制台或 JSON 文件
- 支持流式响应（SSE）
- 返回固定模拟响应，无需真实 API Key

## 安装

```bash
go build -o cc-debug ./cmd/server
```

## 使用

```bash
# 输出到控制台
./cc-debug -port 8080 -output console

# 保存到 JSON 文件
./cc-debug -port 8080 -output json -dir logs
```

将 Claude 客户端的 API 地址指向 `http://localhost:8080` 即可。
