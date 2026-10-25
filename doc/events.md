# События в nui: сигналы

Виджет сообщает о том, что с ним происходит, через **сигналы**. Это поля с именами `On…` и типом `ui.Signal[T]`. К одному сигналу можно подключить сколько угодно обработчиков, и каждый получает данные события аргументом:

```go
table.OnCellDoubleClick.Connect(func(e *ui.TableCellEvent) { open(e.Row) })

conn := search.OnTextChanged.Connect(func(text string) { filter(text) })
conn.Disconnect() // когда обработчик больше не нужен
```

Старые сеттеры `SetOn…` продолжают работать как раньше: каждый хранит один обработчик, и новый вызов заменяет предыдущий. В новом коде лучше использовать сигналы.

## Три правила

1. **Сигналы ввода приходят до обработки самим виджетом.** Их данные — указатель на событие. Обработчик, который поставил `Handled = true`, отменяет обработку по умолчанию: например, буква не будет напечатана, а таблица не откроет редактор ячейки.
2. **`On…Changed` срабатывает на любое изменение значения**: и от пользователя, и из кода (`SetText`, `SetChecked`, `SetSelectedIndex`…). Если значение не изменилось, сигнала нет.
3. **`OnActivated` (и `OnTextEdited` у TextBox) срабатывает только на действие пользователя.** Поэтому обработчик, который сам выставляет значение, не вызывает сам себя, и флаги вроде `updating`/`filling` не нужны.

Все сигналы срабатывают в потоке UI. Обработчики вызываются в порядке подключения. Обработчик, подключённый во время срабатывания, вызывается со следующего раза. Отключённый во время срабатывания больше не вызывается.

## Сигналы любого виджета

| Сигнал | Данные | Когда |
|---|---|---|
| `OnMouseDown`, `OnMouseUp`, `OnMouseDoubleClick` | `*MouseEvent{Button, X, Y, Mods, Handled}` | кнопка мыши нажата, отпущена или дважды нажата. Отпускание приходит тому виджету, на котором кнопку нажали, где бы ни была мышь (для любой кнопки) |
| `OnMouseMove` | `*MouseEvent` | мышь над виджетом, или где угодно, пока нажата кнопка, нажатая на нём |
| `OnWheel` | `*WheelEvent{DX, DY, X, Y, Mods, Handled}` | колесо над виджетом, с позицией мыши и модификаторами: для «Ctrl + колесо = масштаб вокруг курсора» |
| `OnKeyDown`, `OnKeyUp` | `*KeyEvent{Key, Mods, Handled}` | клавиша на виджете с фокусом, раньше сочетаний клавиш формы и собственной обработки виджета |
| `OnChar` | `*CharEvent{Char, Mods, Handled}` | введённый символ |
| `OnClick` | `Widgeter` (кто) | левая кнопка нажата и отпущена на виджете; кнопка нажимается ещё клавишами Space и Enter и методом `Push` |
| `OnResize` | `Size{Width, Height}` | раскладка дала виджету новый размер, вместо опроса ширины таймером |
| `OnFocusChanged` | `bool` | виджет получил (`true`) или потерял фокус |

Координаты в событиях мыши — координаты содержимого виджета с учётом его прокрутки, как в его собственных обработчиках.

## Сигналы конкретных виджетов

| Виджет | Любое изменение | Действие пользователя | Прочее |
|---|---|---|---|
| TextBox | `OnTextChanged(string)` | `OnTextEdited(string)` | `OnSubmit(string)` — Enter в однострочном поле (пока подключён обработчик, Enter не доходит до кнопки по умолчанию формы); `OnEditingFinished(string)` — Enter или уход с поля после правки |
| ComboBox | `OnSelectionChanged(int)` | `OnActivated(int)` — выбор пользователя, даже того же пункта | — |
| Checkbox, ToggleSwitch | `OnCheckedChanged(bool)` | `OnActivated(bool)` | — |
| RadioButton | `OnCheckedChanged(bool)` — в том числе когда отметку сняла другая кнопка группы | `OnActivated(bool)` | — |
| NumBox, Slider | `OnValueChanged(float64)` | `OnActivated(float64)` | — |
| Table | `OnSelectionChanged(TableCell)`, `OnCellChanged(TableCell)` | `OnCellEdited(TableCell)` | `OnCellDoubleClick(*TableCellEvent)` (`Handled` отменяет редактирование), `OnColumnClick(int)` |
| TreeView | `OnSelectionChanged(*TreeNode)` | `OnNodeActivated(*TreeNode)`, `OnCellEdited(TreeCell)` | `OnExpanding(*TreeNode)` — до раскрытия, здесь можно подгрузить детей; `OnCollapsed(*TreeNode)`; `OnColumnClick(int)` |

Сигнал приложения: `ui.OnThemeChanged(bool dark)` — тема применена. Виджеты перекрашиваются сами, а обработчик перекрашивает то, что приложение раскрасило само, например иконки.

## Вместо `CurrentEvent()`

`ui.CurrentEvent()`, `PushEvent`, `PopEvent`, `TextBox.SetOnTextBoxKeyDown` и `Table.SetOnCellMouseDblClick` помечены `Deprecated`.

| Было | Стало |
|---|---|
| `tb.SetOnTextBoxKeyDown(func() { ev := ui.CurrentEvent().Parameter.(*ui.EventTextboxKeyDown); if ev.Key == ui.KeyEnter { …; ev.Processed = true } })` | `tb.OnKeyDown.Connect(func(e *ui.KeyEvent) { if e.Key == ui.KeyEnter { …; e.Handled = true } })` или просто `tb.OnSubmit.Connect(func(text string) { … })` |
| `table.SetOnCellMouseDblClick(func() { ev, _ := ui.CurrentEvent().Parameter.(*ui.EventTableCellMouseDblClick); open(ev.Row) })` | `table.OnCellDoubleClick.Connect(func(e *ui.TableCellEvent) { open(e.Row) })` |
| опрос выделения чужой таблицы таймером, потому что `SetOnSelectionChanged` занят владельцем | `table.OnSelectionChanged.Connect(…)` — обработчиков сколько угодно |
| свой список `themeListeners` | `ui.OnThemeChanged.Connect(…)` |
| опрос ширины виджета таймером | `w.OnResize.Connect(…)` |
| позиция мыши, сохранённая в `OnMouseMove`, и `Form().KeyModifiers()` в обработчике колеса | `w.OnWheel.Connect(func(e *ui.WheelEvent) { zoomAt(e.X, e.Y, e.Mods.Ctrl) })` |

## Свой виджет

Виджет, встроивший `ui.Widget`, получает все сигналы базового виджета без дополнительной работы. Свои сигналы объявляются полями и вызываются через `Emit`:

```go
type ColorWell struct {
	ui.Widget
	OnColorChanged ui.Signal[color.RGBA] // любое изменение
	OnActivated    ui.Signal[color.RGBA] // выбор пользователя
}
```
