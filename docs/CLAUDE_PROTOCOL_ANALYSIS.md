# Claude API 多轮对话结构与协议分析

本文档基于对 Claude Code CLI 工具的网络抓包分析，描述了 Claude API 的多轮对话结构、上下文管理机制以及客户端如何处理复杂任务流。

## 1. 核心机制：无状态 API (Stateless)

Anthropic 的 `/v1/messages` 接口是**无状态**的。这意味着模型本身不记忆之前的对话。

*   **请求结构**：每一次请求（Request）必须包含该次对话所需的**全部历史消息**。
*   **累积模式**：
    *   Turn 1: 发送 `[User A]` -> 返回 `[Asst B]`
    *   Turn 2: 发送 `[User A, Asst B, User C]` -> 返回 `[Asst D]`
    *   Turn 3: 发送 `[User A, Asst B, User C, Asst D, User E]` ...

## 2. 实体定义

在分析抓包数据时，我们识别出以下几个关键实体：

### 2.1 Trace (快照/轨迹)
一次独立的 HTTP 请求 (`POST /v1/messages`)。
*   包含完整的 `request_body`（当时看来所有的历史上下文）和 `response_body`（模型的即时回答）。
*   在 Viewer 中，每一个 Trace 代表对话在某一时刻的“快照”。

### 2.2 Session (会话)
由客户端（CLI）维护的一个生命周期。
*   **标识符**：位于 `request_body.metadata.user_id` 字段中，通常以 `..._session_<UUID>` 结尾。
*   **特点**：同一个 Session ID 下的所有请求属于同一个用户交互进程，但**不一定**共享相同的消息历史（Context）。

### 2.3 Branch (上下文分支)
这是一组拥有共同“消息历史前缀”的 Trace 集合。
*   **主分支 (Main Branch)**：通常是用户与 AI 进行的主要对话流，消息历史不断累积增长。
*   **侧任务分支 (Side Task Branch)**：CLI 工具为了特定目的发起的独立请求，通常不包含主对话的历史。

---

## 3. 复杂行为分析

### 3.1 分支与上下文重置 (Branching & Reset)
我们观察到 Claude CLI 会频繁地“重置”上下文。

*   **现象**：在长时间的对话（消息数 > 5）后，突然出现一个只有 1 条消息的请求。
*   **原因**：
    1.  **侧任务**：CLI 在后台启动了一个独立的 Agent 来执行特定任务（如搜索、总结）。
    2.  **上下文溢出保护**：当 `token` 达到上限，CLI 会触发 `compact` 操作。

### 3.2 提示缓存 (Prompt Caching)
为了降低成本和延迟，Claude 使用 Prompt Caching。

*   **特征**：请求中的消息对象会包含 `cache_control` 字段。
*   **影响**：在对比消息历史时必须忽略此字段。

### 3.3 思考过程 (Thinking Blocks)
Claude 3.7+ 模型引入了显式的思考过程。

*   **结构**：`content: [{ "type": "thinking", ... }, { "type": "text", ... }]`
*   **建议**：进行历史匹配时忽略此块。

---

## 4. 最后一次 Invoke 的内部结构 (Request Payload)

当一个 Turn 包含工具调用时，最后一次 Invoke 是最关键的。它将之前所有的中间步骤打包发回给模型。

```text
================================================================================
                           LLM INVOKE REQUEST (HTTP POST)
================================================================================

+------------------------------------------------------------------------------+
|  [SYSTEM PROMPT] (身份与能力定义)                                              |
+------------------------------------------------------------------------------+
| "You are Claude Code, an expert software engineer..."                        |
|                                                                              |
| +---------------------+   +---------------------+   +---------------------+  |
| | Core Tools          |   | MCP Tools           |   | Output Format       |  |
| | (Bash, Read, Edit)  |   | (get_db_tables...)  |   | (XML/JSON rules)    |  |
| +---------------------+   +---------------------+   +---------------------+  |
+------------------------------------------------------------------------------+
       |
       v
+------------------------------------------------------------------------------+
|  [DYNAMIC CONTEXT] (动态注入的项目信息)                                         |
+------------------------------------------------------------------------------+
| > CLAUDE.md Content:                                                         |
|   "这是一个测试项目。请不要查看项目中的任何文件。"                                  |
|                                                                              |
| > Recent Files:                                                              |
|   ["src/server.py", "web/index.html"]                                        |
+------------------------------------------------------------------------------+
       |
       v
+------------------------------------------------------------------------------+
|  [MESSAGE HISTORY] (累积的对话状态 - State)                                    |
+------------------------------------------------------------------------------+
| 1. [USER]                                                                    |
|    "使用doris mcp工具。看一下omni_data库是否存在ods_user_u表。"                  |
|                                                                              |
| 2. [ASSISTANT] (Previous Output)                                             |
|    Thinking: "我需要查询库中的表列表..."                                        |
|    Tool Call: mcp__doris__get_db_table_list(db="omni_data")                  |
|                                                                              |
| 3. [USER] (Tool Execution Result)                                            |
|    Tool Result: ["ods_user_u", "table_b", "table_c"]                         |
+------------------------------------------------------------------------------+
       |
       v
================================================================================
                          MODEL GENERATION (RESPONSE)
================================================================================
| [ASSISTANT]                                                                  |
| "经查询，omni_data 库中存在 ods_user_u 表。"                                   |
+------------------------------------------------------------------------------+
```

## 5. Dynamic Context 与 Prompt Caching

### 5.1 动态内容的代价
`Dynamic Context`（如 `CLAUDE.md` 内容、最近修改的文件列表、当前终端输出）是**实时更新**的。
每一次 Invoke，客户端都会重新读取这些信息并填入 Prompt。

### 5.2 缓存失效机制
Claude 的 Prompt Caching 基于 **前缀匹配 (Prefix Matching)**。
如果 Prompt 的结构是：
`[Static System] + [Dynamic Context] + [Conversation History]`

一旦中间的 `[Dynamic Context]` 发生变化（例如你修改了一个文件），**其后所有的 `[Conversation History]` 缓存都会失效**，导致 API 调用成本剧增。

### 5.3 结构优化策略 (实测结构)
根据对 `mcp.json` 的实测分析，Claude Code CLI 采用了如下结构：

1.  **System Prompt (纯静态)**：仅包含身份定义和核心工具。这一部分被永久缓存。
2.  **User Message 1 (混合体)**：
    *   **Dynamic Context**: `CLAUDE.md` 内容、Skill 定义。
    *   **Command History**: 最近的终端输出 (`<local-command-stdout>`)。
    *   **Actual Query**: 用户真正的问题。
3.  **Subsequent Turns**: 后续的问答交替。

```text
+-------------------------------------------------------------------+
|  SYSTEM PROMPT (STATIC) - [Cached]                                 |
|  - "You are Claude Code..."                                       |
+-------------------------------------------------------------------+
              |
              v
+-------------------------------------------------------------------+
|  USER MESSAGE 1 (THE "BIG" CONTEXT)                               |
|  - <system-reminder> CLAUDE.md Content... </system-reminder>      |
|  - <local-command-stdout> ... </local-command-stdout>             |
|  - "User Question: How do I fix this bug?"                        |
+-------------------------------------------------------------------+
              |
              v
+-------------------------------------------------------------------+
|  ASSISTANT RESPONSE 1                                             |
|  - Tool Call / Text                                               |
+-------------------------------------------------------------------+
```

**影响**：
由于 Dynamic Context 被捆绑在第一条消息中，如果 `CLAUDE.md` 在对话中途发生变化，或者终端历史过长需要轮替，整个 Conversation History 的头部就会发生变化，从而导致**整个会话的缓存失效**。

## 6. 术语表

1.  **Session (会话)**: 用户打开终端到关闭终端的全过程。在元数据中由 `session_id` 标识。
2.  **Turn (轮次)**: 用户发起一次交互，到 AI 给出最终回答并停止工作的过程。一个 Turn 可能包含多次 **Invoke**。
3.  **Invoke (调用)**: 一次 HTTP 请求/响应周期。这是无状态的快照。
4.  **Tool Call / Resp**: 作为 **Message 数组**中的项存在。`tool_use` (assistant) -> `tool_result` (user)。
5.  **CLAUDE.md**: 项目级规范，由客户端在发起 Invoke 之前读取并注入到上下文。