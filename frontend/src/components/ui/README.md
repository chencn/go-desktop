# shadcn-vue 组件目录

本目录只存放 shadcn-vue CLI 生成或覆盖的源码组件。

规则：

- 只能通过 `shadcn-vue add ... --overwrite` 或等价 CLI 流程更新。
- 禁止放业务组件、业务 class、全局注册文件或项目兼容 wrapper。
- 禁止手写修改现有组件实现；需要项目级适配时放到 `frontend/src/shared/ui`。
- 页面私有控件放到对应 `frontend/src/features/**` 目录。
- `frontend/tsconfig.json` 排除 `src/components/ui/**/*.vue`，避免框架生成 Vue 文件被当作业务源码全量检查；实际打包链路仍由前端构建验证。
