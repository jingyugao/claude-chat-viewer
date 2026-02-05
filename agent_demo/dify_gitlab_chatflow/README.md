# Dify Chatflow: omni-data weekly commits

This directory contains a Dify Chatflow DSL file that uses the MCP Agent strategy to fetch last week's GitLab commits for the `omni-data` project and supports chat-based Q&A.

## Files
- `agent_demo/dify_gitlab_chatflow/omni-data-weekly-commits-chatflow.yml`

## Import
1. In Dify, create a new app and choose **Chatflow** (advanced chat).
2. Import the YAML file above.
3. Set the environment variables in the app settings:
   - `GITLAB_PROJECT` (project path like `group/omni-data` or numeric ID)
   - `GITLAB_REF` (optional branch or tag)
4. In the Agent node, configure MCP servers (the `config_json` field) to point at your GitLab MCP server, or set it in the MCP Agent plugin UI.

The chatflow computes the previous calendar week in UTC (Monday 00:00:00 through Sunday 23:59:59.999999) and uses MCP tools to list commits in that range.
