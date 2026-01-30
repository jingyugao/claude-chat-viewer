# Claude 使用指南

本指南旨在梳理 Claude 的核心能力与应用场景，帮助用户更高效地利用 Claude 进行开发、分析与自动化工作。

## 1. 核心定位 (Core Identity)

**Claude Code 的本质**
Claude 不仅仅是一个代码生成工具，更是一个**通用型 AI Agent 框架**。
> "Claude Code 已经远远超越了编程工具的范畴，我们在 Anthropic 内部将它用于深度研究、视频创作、笔记整理等无数非编程场景。"

---

## 2. 核心技能详解 (Core Skills Deep Dive)

### 2.1 技能 (Skill)
Claude 的“技能”是指其内置或通过扩展获得的操作能力。与简单的问答不同，技能允许 Claude **主动执行动作**，如读取文件、运行终端命令、搜索网络等。
*   **特性**: 具备原子性、可组合性。
*   **使用**: 用户无需手动调用底层 API，只需用自然语言描述目标（例如 "帮我重构这个文件"），Claude 会自动规划并调用相应的技能（Read File -> Replace -> Run Test）。

### 2.2 MCP (Model Context Protocol)
MCP 是连接 AI 模型与外部数据/工具的标准协议。通过 MCP，Claude 可以安全地连接到本地数据库、GitHub 仓库、Slack 或自定义的内部系统。

#### 实战案例：连接 Apache Doris 数据库
Apache Doris 兼容 MySQL 协议，因此我们可以通过配置通用的 **MySQL MCP Server** 让 Claude 具备查询和分析 Doris 数据的能力。

**配置步骤 (Config)**
通常在 Claude 的配置文件（如 `claude_desktop_config.json` 或项目级配置）中添加 server 定义：

```json
{
  "mcpServers": {
    "doris-db": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "mcp/mysql", 
        "--host", "127.0.0.1",
        "--port", "9030",
        "--user", "root",
        "--password", "your_password",
        "--database", "your_db"
      ]
    }
  }
}
```

**使用方法 (Usage)**
配置完成后，重启 Claude 客户端。你可以直接用自然语言下达复杂的分析指令，Claude 会自动生成并执行 SQL：
*   **用户**: "检查 `doris-db` 中的 `orders` 表，统计过去 30 天每个地区的销售总额，并按金额降序排列。"
*   **Claude**: (自动调用 MCP 工具 -> `show tables` -> `desc orders` -> 生成 SQL -> 执行查询 -> 返回分析结果)

### 2.3 Command (自定义命令)
Command 是用户为 Claude 定义的快捷指令，用于封装常用的上下文、Prompt 模板或特定的工作流。

**如何新建 Command**
在项目的 `CLAUDE.md` 或全局配置中定义。
*   **格式**: `命令名称: 对应的 Shell 命令或自然语言指令`

```markdown
# CLAUDE.md 示例
commands:
  - test: npm test
  - lint: npm run lint -- --fix
  - summary: 总结当前目录下所有 .md 文件的核心内容
  - deploy: ./scripts/deploy.sh staging
```

**如何使用 Command**
在对话框中直接输入命令名称，或者配合 `/` 使用（取决于具体客户端实现）：
*   "运行 `test` 命令"
*   "执行 `summary`"
Claude 会识别这些别名并执行对应的底层指令，节省重复输入 Prompt 的时间。

### 2.4 探索-规划-执行 (E-P-E)
**E-P-E (Explore-Plan-Execute)** 是 Claude 解决复杂、未知问题的标准思维范式。

1.  **探索 (Explore)**:
    *   **动作**: 浏览文件结构 (`ls -R`)、阅读相关文档 (`read_file`)、搜索代码库 (`grep`).
    *   **目的**: 建立对现状的认知，理解依赖关系，避免盲目修改。
2.  **规划 (Plan)**:
    *   **动作**: 提出分步实施计划，预测潜在风险。
    *   **输出**: "我将分三步修改：1. 创建接口... 2. 实现逻辑... 3. 更新测试..."
3.  **执行 (Execute)**:
    *   **动作**: 编写代码、运行命令。
    *   **原则**: 原子化提交，每一步修改后立即验证。

### 2.5 测试驱动开发 (TDD)
Claude 极其适合 TDD 模式，因为这能最大程度减少 AI 产生的幻觉和逻辑错误。

**工作流**:
1.  **Red (写失败测试)**:
    *   用户: "我们要添加一个‘计算折扣’的功能。请先写一个单元测试，覆盖正常折扣和异常输入的情况。"
    *   Claude: 创建 `test_discount.py`，运行测试，确认失败。
2.  **Green (让测试通过)**:
    *   用户: "现在实现 `calculate_discount` 函数，通过上述测试。"
    *   Claude: 编写最小可用代码。
3.  **Refactor (重构)**:
    *   用户: "代码通过了，现在优化一下变量命名，并提取常量。"
    *   Claude: 在保证测试通过的前提下优化代码结构。

---

## 3. 电脑自动化 (Computer Automation)
利用 Claude 进行桌面级操作与流程自动化。
*   **典型工具:** ClawdBot (开源版), Cowork (Claude 闭源版)
*   **主要场景:** 桌面自动化、浏览器操控、文件批量管理、跨应用工作流串联。

---

## 4. AI 数据分析 (Data Analysis)
Claude 在数据处理、SQL 生成及业务分析方面的深度应用。

### SQL 生成与优化
*   **自然语言转 SQL**: 支持多表 JOIN、窗口函数、CTE。
*   **性能优化**: 慢查询分析、索引优化建议、执行计划解读。

### 数据探索与统计分析
*   **质量检测**: 缺失值、异常值清洗。
*   **统计与聚类**: 分布分析、相关性热力图、K-Means 聚类。

### 业务指标与漏斗分析
*   **指标体系**: DAU/MAU、LTV、ARPU、留存率。
*   **漏斗优化**: 识别流失节点，生成 A/B 测试方案。

### 用户行为与画像
*   **行为分析**: 路径归因、Session 分析。
*   **用户分层**: RFM 模型、用户画像标签化。

---

## 5. AI 运维 (AIOps)
利用 Claude 进行系统诊断、故障排查与资源优化。

*   **性能诊断**: 火焰图分析、内存泄漏排查、锁争用检测。
*   **日志与故障**: 日志聚合分析、Coredump 解析、根因定位。
*   **Kubernetes**: Helm/Yaml 配置生成、Pod 故障排查、HPA 优化。
*   **监控**: PromQL 生成、Grafana 面板配置、告警阈值建议。

---

## 6. AI 爬虫与数据采集
*   **接口逆向**: 识别加密/签名算法、生成调用代码。
*   **页面解析**: 智能 DOM 结构识别、生成稳定 CSS 选择器。
*   **内容清洗**: 非结构化数据提取、模板填充与二次加工。