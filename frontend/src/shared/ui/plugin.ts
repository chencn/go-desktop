import type { App, Component } from 'vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { CardAction, CardContent, CardDescription, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import AlertDialogCompat from './AlertDialog.vue'
import CardCompat from './Card.vue'
import CardTitleCompat from './CardTitle.vue'
import DialogCompat from './Dialog.vue'
import Field from './Field.vue'
import Label from './Label.vue'
import NativeSelect from './NativeSelect.vue'
import ProgressCompat from './Progress.vue'
import SwitchCompat from './Switch.vue'
import TooltipCompat from './Tooltip.vue'

// components 是全局 Ui* 注册白名单；业务页只能从这里拿项目 wrapper 或官方 primitive 别名。
const components: Record<string, Component> = {
  UiAlertDialog: AlertDialogCompat,
  UiBadge: Badge,
  UiButton: Button,
  UiCard: CardCompat,
  UiCardAction: CardAction,
  UiCardContent: CardContent,
  UiCardDescription: CardDescription,
  UiCardHeader: CardHeader,
  UiCardTitle: CardTitleCompat,
  UiDialog: DialogCompat,
  UiField: Field,
  UiInput: Input,
  UiLabel: Label,
  UiNativeSelect: NativeSelect,
  UiProgress: ProgressCompat,
  UiSwitch: SwitchCompat,
  UiTable: Table,
  UiTableBody: TableBody,
  UiTableCell: TableCell,
  UiTableHead: TableHead,
  UiTableHeader: TableHeader,
  UiTableRow: TableRow,
  UiTooltip: TooltipCompat,
}

export const uiPlugin = {
  install(app: App) {
    // 统一注册避免业务页重复 import primitive，也把 shared/ui wrapper 作为项目级行为边界。
    for (const [name, component] of Object.entries(components)) {
      app.component(name, component)
    }
  },
}
