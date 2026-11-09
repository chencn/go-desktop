# 代码注释规范

## 通用

- **结构**：第一段自然语言就是一句话契约（第一个空行前的内容进搜索与概览，务必一行）；细节与设计原因放后续段落或语言对应的标签。
- **内容**：写业务契约与设计原因（单位、状态转换、错误边界、动画时序参数的单位），不复述代码或签名；无内容的小节不留空；禁止用注释记载变更历史（那是 git 的事），禁止写外部参考项目来源。
- **引用**：跨项引用用语言文档工具认的形式（TS 用 `{@link Type}`，Go 用 `[Type]` / `[pkg.Type]`）；不要写「见 xx.vue 第 N 行」这类行号引用（行号会漂移）。
- 自动生成的文件（如 `components.d.ts`、wire/sqlc 产物）与上游生成代码不适用本规范。

## JS / TS / Vue

新增或修改的声明紧贴写 JSDoc `/** ... */`，TypeScript 遵循 TSDoc 标签；纯 JS 文件用 JSDoc 类型注解（`@param {Type}`）补齐类型信息。覆盖关键字：`function`（含 `async`）、`interface`、`type`、`enum`、`class`、`const`、`let`，带 `export` 的同样适用；组合式函数（`use*`）与未导出的辅助声明不豁免。按形态要求：

- **function / 箭头函数**：一句话契约 + `@param` 逐个 + `@returns`（`void` 也写，说明副作用与提前返回条件）；会 reject 的异步函数写 `@throws`。
- **interface / type 别名 / enum**：一句话契约；每个成员一律用前置 `/** 一句话 */`，禁止行尾 `//`。类型别名是联合类型时成员无法挂 JSDoc，用行尾 `//` 或在类型上方整段说明。
- **class**：一句话契约；字段与方法一律前置 `/** 一句话 */`，禁止行尾 `//`。
- **const / let 标量字面量**（string/number/boolean）：一句话契约即可，无标签要求。
- **const 对象 / 数组 / 配置表**：一句话契约，说明结构与取值含义。
- **store / 组合式函数返回对象**：`@returns` 逐字段说明。

- **TS 标签**：`@param` / `@returns` 按序逐个说明，`@returns` 写明 `null`/`undefined`/空数组等特殊值；异步失败写 `@throws` 并注明 reject 的类型及判定方式，同步异常同样 `@throws`；泛型约束无需专门标签，参数说明里带出即可；可见性、弃用等用 TSDoc 标签 `@deprecated` / `@experimental`，不造新标签。
- **JS 文件**：签名无类型时必须写 `@param {Type}` / `@returns {Type}`；能迁 TS 就迁，不为注释而注释。
- **Vue SFC**：
  - 组件契约（props、emits、slots、v-model）写进对应声明的 JSDoc：`defineProps` 上方注释整体职责，各 prop 靠 `withDefaults` + 接口字段的前置 `/** */` 说明单位与默认值语义；`defineEmits` 的每个事件在类型里命名并注明触发时机与载荷。
  - 组合式函数（`use*`）按普通函数规范写 JSDoc；返回值是对象时逐字段说明。
  - `<template>` 内只写「为什么」型 `<!-- -->` 注释（非直觉的分支、魔法值的含义），不复述结构；`<style>` 内不写注释，语义放进类名。
- **小节排序**：一句话契约 → 细节正文 → `@throws` / `@remarks` → `@example`（仅公共 API 且用法不自明时写）。

TS 示例：

```ts
/**
 * 获取第三方平台的接口调用凭据。
 *
 * @param ctx - 请求上下文；取消时中止刷新且不写缓存
 * @param user - 当前登录用户及租户信息
 * @returns 接口调用凭据，不含 Bearer 前缀
 * @throws {@link TokenExpiredError} 凭据已过期时 reject
 * @remarks 并发安全；同租户并发刷新合并为一次上游请求
 */
function getAccessToken(ctx: Ctx, user: Member): Promise<string>

/** 已缓存的第三方平台凭据。 */
interface Credential {
  /** 访问令牌，不含 Bearer 前缀 */
  token: string;
  /** 过期时刻（epoch 毫秒），提前 60 秒视为过期 */
  expiresAt: number;
}
```

Vue SFC 示例：

```vue
<script setup lang="ts">
/** 预约成功弹窗；仅展示结果，不承载业务校验。 */
interface Props {
  /** 医生展示名，格式「科室 · 姓名」 */
  title: string;
  /** 关闭后是否自动回滚已选号源，默认 false */
  rollback?: boolean;
}

const props = withDefaults(defineProps<Props>(), { rollback: false });

/** 用户确认后触发；载荷为号源 ID，由父级负责跳转 */
const emit = defineEmits<{ confirmed: [slotId: string] }>();
</script>

<template>
  <!-- rollback 只在弹窗关闭动画结束后执行，避免视觉跳变 -->
  <van-dialog v-if="props.rollback" ... />
</template>
```

## Go

新增或修改的声明紧贴写 godoc 行注释 `//`（Go 文档工具只认 `//`，不用 `/** */`）。覆盖关键字：`package`、`func`（含方法）、`type`（struct / interface / 别名）、`const`、`var`；导出与非导出声明同样适用，未导出的辅助函数不豁免。按形态要求：

- **package**：每个包至少一个文件写 `// Package xxx ...` 包注释，说明包的职责与边界；单文件包可附「文件职责」说明。
- **func / 方法**：注释以标识符名开头，一句话契约；随后说明参数语义（单位、零值含义、`ctx` 取消行为）、每个返回值的含义与特殊值（`nil`、空切片、零值、`bool` 为 `false` 的条件）；返回 `error` 的写明错误边界与判定方式（`errors.Is` / `errors.As` 的哨兵或类型）。
- **type（struct / interface）**：一句话契约；struct 每个字段带行尾 `//` 简注（单位、格式、零值语义），需要进悬停提示的用前置 `//`；interface 每个方法按 func 规范写。
- **const / var**：标量字面量一句话契约即可；`const` / `var` 分组（枚举、配置表）组注释说明整组含义，每个成员行尾 `//` 简注取值语义。
- **弃用**：单独一段以 `Deprecated: ` 开头（godoc 标准段落），注明替代方案。
- **结构**：一句话契约 → 参数/返回值/错误边界正文 → 设计原因；示例代码写进 `_test.go` 的 `Example` 函数，不塞进注释。
- **测试文件**：`Test*` 函数注释说明验证的行为契约与回归背景，不复述断言步骤。

Go 示例：

```go
// Package semver 提供项目内部使用的简化版本号解析和比较。
// 它只接受 v?N、v?N.N 或 v?N.N.N 三种数字版本，不支持 prerelease/build metadata。
package semver

// Parse 解析 v?N(.N){0,2} 格式版本，并把缺失段补 0。
// 任一段为空、包含非数字字符或超过三段时返回 false。
func Parse(value string) (Version, bool)

// ConfigItem 是通用配置项结构。
// value/default_value 都以字符串保存，业务层负责按 value_type 做类型转换和枚举校验。
type ConfigItem struct {
	Key       string // 稳定配置键，例如 startup.auto_launch
	ValueType string // 值类型：bool、int、string
	UpdatedAt string // 更新时间（RFC3339 UTC）
}

// DownloadAndVerify 下载更新资产并校验 SHA256，返回落盘路径。
// ctx 取消时中止下载并清理临时文件；progress 为 nil 时不上报进度。
// 校验不匹配返回 error 并删除已下载文件。
func (m *Manager) DownloadAndVerify(ctx context.Context, asset ReleaseAsset, progress ProgressFunc) (DownloadResult, error)
```
