这段和 semiMessages 的同步逻辑是“按 runId 锚定 assistant 消息，再按 activityMessageId 锚定 content item 做增量更新”：

事件筛选和主键解析
在 ACTIVITY_SNAPSHOT/ACTIVITY_DELTA 里取 runId + activityType + activityMessageId，缺一就丢弃。

进入 updateRunMessage(runId, updater)
updateRunMessage 会在 semiMessages 中找到（或创建）该 runId 的 assistant 消息，然后把更新后的消息写回状态。

先用索引缓存命中，再回退扫描
activityMessageMapRef 缓存了 activityMessageId -> {runId,index}。
先尝试用缓存 index 直接定位；如果 index 失效（越界/ID不匹配），再 findIndex 扫描 content。

合并内容（snapshot/delta）
拿到 previousItem?.content 后，通过 mapActivityContent 合并新事件：

SNAPSHOT：覆盖或浅合并
DELTA：增量合并（a2ui-surface 走 patch->operations 逻辑）

写回 content 并刷新缓存索引
存在则替换该 item，不存在则 push；然后更新 activityMessageMapRef 的最新 index，最后返回新 message 触发 semiMessages 更新。

简化理解：semiMessages 是真状态，activityMessageMapRef 只是“加速定位”的索引缓存。