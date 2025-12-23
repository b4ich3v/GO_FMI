# Cloud Logging (CLS) Agent

Read-only agent for querying SAP Cloud Logging.

## What it does
- Builds a prompt from `CLSParams` and calls the Kibana log tool via LangChain.
- Supports filters: time window, message substring, exact match, app name, DC, level, summary mode.
- Updates token usage and records a concise history entry for the UI.

## Components
- `cls_agent.py`: orchestrates prompt → tool call → formatting; uses `CLSValidator`, `CLSTokenHandler`, `CLSToolRunner`, `CLSHistoryRecorder`.
- `cls_prompt_builder.py`: builds the tool prompt embedding `CLSParams.to_dict()`.
- `validation/`:
  - `CLSValidator`: validates agent lifecycle args (`query`, `thread_id`, config keys).
  - `ClsParamsValidator`: validates params (minutes in range, optional strings, booleans).
- `cls_history_recorder.py`: appends a short summary + answer into shared memory.

## Flow
1) UI/form builds `CLSParams` (via `CLSParamsFormAdapter`) and wraps it in `AgentRequest(TypeOfAgent.CLS_AGENT, params)`.
2) `CLSAgent.run` validates args (`CLSValidator`), builds prompt, runs tool with `CLSToolRunner`.
3) `CLSTokenHandler` performs pre-check and post-run token updates.
4) `CLSHistoryRecorder` writes a summary into the shared memory.

## Validation & limits
- `minutes_ago` must be 1..1440.
- `message`, `app_name`, `dc_name`, `level` are `str` or `None`.
- `exact`, `summary_mode` are booleans.
- Agent args: `thread_id`, `agent_params_key`/`config` are required where applicable.

## Usage hints
- Typical fields: `minutes_ago`, `message`, `exact`, `app_name`, `dc_name`, `level`, `summary_mode`.
- Called by `AgentManager` when `AgentRequest(TypeOfAgent.CLS_AGENT, cls_params)` is present.
- Purely read-only; no mutations.
