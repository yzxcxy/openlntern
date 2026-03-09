# 核心数据结构

```go
type FrontMatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type Skill struct {
	FrontMatter
	Content       string
	BaseDirectory string
}
```

Backend，用于获取skill

```go
type Backend interface {
	List(ctx context.Context) ([]FrontMatter, error)
	Get(ctx context.Context, name string) (Skill, error)
}
```

# 中间件注入的内容

```go
	return adk.AgentMiddleware{
		AdditionalInstruction: buildSystemPrompt(name, config.UseChinese),
		AdditionalTools:       []tool.BaseTool{&skillTool{b: config.Backend, toolName: name, useChinese: config.UseChinese}},
	}, nil
```

注入了一个工具和一个提示词，这里的name实际上就是skill工具的name，默认为skill,可以通过该skill去加载工具。

系统提示词如下：

```txt
# Skill 系统

**如何使用 Skill（技能）（渐进式展示）：**

Skill 遵循**渐进式展示**模式 - 你可以在上方看到 Skill 的名称和描述，但只在需要时才阅读完整说明：

1. **识别 Skill 适用场景**：检查用户的任务是否匹配某个 Skill 的描述
2. **阅读 Skill 的完整说明**：使用 '{tool_name}' 工具加载 Skill
3. **遵循 Skill 说明操作**：工具结果包含逐步工作流程、最佳实践和示例
4. **访问支持文件**：Skill 可能包含辅助脚本、配置或参考文档 - 使用绝对路径访问

**何时使用 Skill：**
- 用户请求匹配某个 Skill 的领域（例如"研究 X" -> web-research Skill）
- 你需要专业知识或结构化工作流程
- 某个 Skill 为复杂任务提供了经过验证的模式

**执行 Skill 脚本：**
Skill 可能包含 Python 脚本或其他可执行文件。始终使用绝对路径。

**示例工作流程：**

用户："你能研究一下量子计算的最新发展吗？"

1. 检查可用 Skill -> 发现 "web-research" Skill
2. 调用 '{tool_name}' 工具读取完整的 Skill 说明
3. 遵循 Skill 的研究工作流程（搜索 -> 整理 -> 综合）
4. 使用绝对路径运行任何辅助脚本

记住：Skill 让你更加强大和稳定。如有疑问，请检查是否存在适用于该任务的 Skill！
```

# 注入的工具🔧

工具结构体

```go
type skillTool struct {
	b          Backend
	toolName   string
	useChinese bool
}
```

1. 初始阶段: 只显示技能名称列表（通过工具描述）
2. 按需加载: Agent调用 skill: "skill-name" 时才加载完整内容 （实际上就是调用这个工具）
3. 效率优化: 避免一次性加载所有技能内容

调用这个skill工具，就会获得代表这个skill的完整结构体，包含绝对路径，通过这个绝对路径可以使用其他工具去访问这个skill的其他内容。