# Unsupported host boundaries

Antigravity CLI remains unsupported because its process-only credential and Skill execution binding are not verified. Rakazo, OpenMausBot and Cursor are not registered Skill hosts. Do not substitute a supported host ID.

Switch's `deepseek-bot` is a separate community client, not `deepseek-harness`
or `dsh-desktop`. It currently has no registered Skills host, installation
target or credential adapter. Do not pass either Harness host ID or read its
connection as a substitute. Switch configuration alone does not enable Skills.

Its per-Bot model selection must be bound to the current execution session
before an adapter can be enabled. Skill discovery, effective connection and
native result delivery still need independent implementation and acceptance;
a saved default or successful chat does not establish those capabilities.
