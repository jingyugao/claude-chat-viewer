import asyncio
from os import system
from claude_agent_sdk import (
    query,
    ClaudeAgentOptions,
    ClaudeSDKClient,
    create_sdk_mcp_server,
)


system_prompt = """
你是一个k8s运维专家。
你精通Kubernetes集群的管理和故障排除，能够帮助用户解决各种与Kubernetes相关的问题，包括但不限于Pod管理、服务配置、网络设置和资源监控。
更新k8s后，请务必检查集群的健康状态，确保所有节点和Pod正常运行。并发送lark消息通知相关人员。
对于简单的代码bug，请你修复代码提交分支，并通知相关人员review。
"""


async def main():
    options = ClaudeAgentOptions(
        system_prompt="You are a helpful assistant.",
        mcp_servers={
            "grafana": {"type": "http", "url": "{grafana_mcp_host}"},
            "lark": {"type": "http", "url": "{lark_mcp_host}"},
        },
        tools=["Bash", "Skill"],
        allowed_tools=["Bash", "Skill"],
        model="haiku",
    )
    async with ClaudeSDKClient(system_prompt=system_prompt, options=options) as client:
        await client.query("分析一下刚才oom的服务，并给出优化建议。")
        async for message in client.receive_response():
            print(message)


asyncio.run(main())
