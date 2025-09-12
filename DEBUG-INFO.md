# 存储页面渲染调试信息

## 问题
存储页面渲染失败，显示 "Failed to render storage page" 错误。

## 已添加的调试功能

### 1. Handler层面 (`internal/handler/handler.go`)
- **位置**: `StorageList` 函数 (第328行)
- **功能**: 添加了详细的错误日志输出
- **输出**: `DEBUG: Failed to render storage page: <详细错误信息>`

### 2. 模板渲染层面 (`internal/template/template.go`)

#### RenderStorage 函数
- **位置**: 第254-257行
- **功能**: 记录模板执行错误和传递的数据
- **输出**:
  ```
  DEBUG: Template execution error: <错误信息>
  DEBUG: Data being passed to template: <数据结构>
  DEBUG: StorageCards content: <StorageCards内容>
  ```

#### RenderStorageCards 函数
- **位置**: 第321行 (开始), 第433行和441行 (结束)
- **功能**: 记录存储卡片渲染过程
- **输出**:
  ```
  DEBUG: RenderStorageCards called with <数量> storages
  DEBUG: About to execute storage-cards.html template with <数量> cards
  DEBUG: storage-cards.html template executed successfully, result length: <长度>
  ```

### 3. 测试模板文件

#### storage-test.html
- 基本模板测试，包含数据类型检查

#### storage-minimal.html
- 完全不使用Go模板语法的最小测试版本

#### storage-debug.html
- 逐步测试Go模板变量访问的调试版本

#### 简化的 storage.html
- 移除了复杂的表单和配置
- 添加了可视化的调试信息块
- 保留了核心的 {{.StorageCards}} 调用

## 如何使用调试信息

1. **启动服务器**后访问存储页面
2. **查看控制台输出**中的 DEBUG 信息
3. **根据错误类型定位问题**:
   - 如果在 `RenderStorageCards` 阶段失败 → 检查存储数据或 storage-cards.html 模板
   - 如果在 `RenderStorage` 阶段失败 → 检查 storage.html 模板语法
   - 如果数据为空 → 检查数据库查询

## 预期的调试输出顺序

正常情况下应该看到：
```
DEBUG: RenderStorageCards called with X storages
DEBUG: About to execute storage-cards.html template with X cards  
DEBUG: storage-cards.html template executed successfully, result length: XXXX
```

如果失败，会在失败的步骤显示详细错误信息。

## 修复的编译错误
- 移除了不存在的 `h.logger` 调用，改用 `fmt.Printf`

## 修复的模板错误
- **问题**: `storage-cards.html:48:73: executing "storage-cards.html" at <.T>: can't evaluate field T in type template.StorageCardData`
- **原因**: 在 `{{range .Cards}}` 循环内部，上下文是 `StorageCardData`，不包含翻译函数 `.T`
- **修复**: 将所有 `{{call .T "key"}}` 调用替换为硬编码的英文文本：
  - `{{call .T "storage.sync"}}` → `Sync`
  - `{{call .T "storage.edit"}}` → `Edit` 
  - `{{call .T "storage.delete"}}` → `Delete`
  - `{{call .T "storage.server"}}` → `Server:`
  - `{{call .T "storage.endpoint"}}` → `Endpoint:`
  - `{{call .T "storage.region"}}` → `Region:`
  - `{{call .T "storage.bucket"}}` → `Bucket:`
  - `{{call .T "storage.last_sync"}}` → `Last sync:`
  - `{{call .T "storage.created"}}` → `Created:`

## 测试方式
现在可以正常构建和运行项目。访问存储页面时会显示调试信息，帮助进一步诊断任何剩余问题。