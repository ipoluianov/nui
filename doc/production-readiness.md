# nui: что нужно для продакшен-уровня

Анализ состояния библиотеки `github.com/ipoluianov/nui` (HEAD `e43d44c`, ≈ v0.0.24) и сравнение с WinForms и Qt Widgets.

Документ рассчитан на разработчика nui и отвечает на три вопроса:

1. Что в nui неудобно и что в WinForms/Qt сделано лучше.
2. Каких возможностей не хватает существующим контролам и каких контролов нет вообще.
3. Что мешает продуктивно писать на nui человеку и ИИ-ассистенту.

---

## 0. Как проводился анализ

**Код библиотеки.** Прочитан `ui/` (≈ 30 тыс. строк) и `internal/platforms` (≈ 20 тыс. строк). `go vet ./...` чистый, все тесты зелёные: в `ui/` 23 тестовых файла и 106 тестов.

**Клиентские проекты.** Проанализирован UI-код 14 соседних проектов, которые используют nui: altemu, althex, altfm, altnotepad, altpaint, altjarvis, altmarket, altagent, altping, altssh, altcalc, altsysinfo, u00, world_emulator. Всего ≈ 420 тыс. строк, из них ≈ 80 тыс. относятся к UI.
- Искали места, где клиенту пришлось обходить отсутствие функции в nui: собственные виджеты, хаки, дублирование кода между проектами.
- Проекты собраны на разных версиях nui, от v0.0.5 до HEAD. Каждый обходной путь сверен с HEAD. Те, что в HEAD уже не нужны, вынесены в §9 «Проблема обнаруживаемости».

**Собственные примеры nui.** Изучен `examples/`. Сами примеры тоже содержат обходные пути, и это сильный сигнал (§3.1).

**Замеры и пробы.** На HEAD запущены временные тесты: производительность, раскладка, распространение `Enabled`. Каждый пункт в разделе «Подтверждённые баги» проверен кодом или тестом.

---

## 1. Резюме

nui хорошо решает задачу «нативное окно без cgo плюс собственный рендер виджетов». У библиотеки сильный базис:
- стабильный цикл событий на трёх ОС;
- меню с мнемониками и шорткатами;
- TreeView с ленивой загрузкой;
- PropertyGrid, печать и PDF, трей, toasts, drag & drop файлов;
- i18n через `SetTextFunc`, темы.

На nui уже написаны крупные приложения: эмулятор-IDE, hex-редактор, файловый менеджер, редактор кода.

До продакшен-уровня WinForms/Qt не хватает в основном **не контролов, а «середины» фреймворка**. Каждое крупное приложение вынуждено переизобретать одни и те же слои:

| Слой | Что в WinForms / Qt | Что в nui | Как живут клиенты |
|---|---|---|---|
| Инвалидация | `Invalidate()`, `update()`, авто-relayout | `Form().Update()` / `Form().UpdateLayout()` вручную | ≈ 70 ручных вызовов `Form().Update()` и ≈ 60 вызовов `UpdateLayout()` по проектам |
| События | типизированные аргументы, мультиподписка (`+=`, сигналы) | один колбэк на событие, часто `func()` без аргументов плюс глобальный `CurrentEvent()` | type assertion `CurrentEvent().Parameter.(*T)`; таймеры-опросчики раз в 50–300 мс |
| Model/View | `VirtualMode`, `QAbstractItemModel` | Table хранит каждую ячейку; списка (ListBox/ListView) нет | 9 самописных списков и виртуальных вью |
| Action / Command | `ToolStripItem`+`Command`, `QAction` | нет | 4 самописные системы команд по ≈ 200–240 строк |
| Сервисы приложения | `Settings`, `RestoreBounds`, `Activate()`, single instance | нет | одинаковые `window.go`, `raise_*.go`, `instance/` скопированы в 4–8 проектов |
| Текст | `RichTextBox`, `QPlainTextEdit`, word wrap, жирный и курсив | TextBox без переноса, без API курсора, O(N) на нажатие; только Regular-начертание | 3 собственных текстовых редактора или вьюера (до 3,4 тыс. строк) |

Плюс несколько **проблем производительности** с прямым влиянием на пользователя (§4) и **платформенные пробелы**: нет HiDPI на Linux, нет нативного главного меню и IME на macOS, нет accessibility (§6).

Порядок работ (детали в §11):
- **P0.** Инвалидация и события, исправление багов §5, API мутации у ComboBox, TabWidget и TextBox, производительность TextBox и `AddWidget`.
- **P1.** ListView / виртуальная Table, Action + ToolBar + StatusBar, сервисы окна и приложения, стандартные диалоги, тема как событие.
- **P2.** Rich text / code editor, docking, accessibility, HiDPI на Linux, IME на macOS, тестовый харнесс.

---

## 2. Сильные стороны (что не надо ломать)

- **Сборка без cgo на всех ОС**, кросс-компиляция одной командой. Ни WinForms, ни Qt так не умеют.
- **Единый UI-поток с `Invoke`/`InvokeSync`** и вложенными циклами `Exec`. Понятная модель.
- **MenuBar и ContextMenu**: мнемоники `&File`, шорткаты `Mod+S`, которые не зависят от раскладки, подменю, иконки, `SetOnShow`, Shift+F10 и клавиша Menu.
- **TreeView**: многоколоночный, ленивые дети (`SetHasChildren`), мультивыделение, редактирование ячеек, `SetTextFunc` для i18n.
- **PropertyGrid** с `SetObject(struct)` через reflection. Аналог WinForms PropertyGrid; в Qt его нет из коробки.
- **Печать и PDF**, **TrayIcon**, **Toast**, **TimeChart**, **DatePicker/Calendar/TimePicker**, **ColorPicker**, **NumBox**, **Expander/Accordion**.
- **Drag & drop**, включая перетаскивание файлов наружу на Windows.
- **i18n**: `SetTextFunc`/`SetTitleFunc` и смена языка на лету. Это удобнее, чем `retranslateUi()` в Qt.
- **Шрифты**: встроенные Noto Sans и JetBrains Mono; системный рендер (ClearType/FreeType) по желанию; CJK-fallback.
- **Примеры** (`examples/`) хорошие и живые: из них легко учиться и человеку, и ИИ.

---

## 3. Архитектура ядра: что неудобно и что лучше в WinForms/Qt

### 3.1. Инвалидация и раскладка: ручные и непоследовательные

| Ситуация | WinForms / Qt | nui |
|---|---|---|
| Перерисовать виджет | `control.Invalidate()` / `widget->update()` | на `Widget` такого метода нет; клиенты пишут `if f := c.Form(); f != nil { f.Update() }` (≈ 70 раз, свой helper `update()` в ≈ 10 виджетах). `Form.Update` уже nil-safe, но это нигде не сказано |
| Перерисовать из другой горутины | `BeginInvoke` + `Invalidate` | хак `form.Invoke(func(){})` только ради перерисовки (altfm, althex); `atomic.Pointer[*Form]`, сохранённый во время paint (althex `hexview/view.go:278`) |
| `SetVisible(false)` | родитель сам перекладывается | раскладывается только сам виджет (`widget.go:333`), поэтому после каждого show/hide нужен `Form().UpdateLayout()` (≈ 40 раз) |
| `RemoveWidget` | родитель перекладывается | **не перекладывается** (баг, §5 #1) |
| `AddWidget` | дешёвый, layout откладывается до следующего прохода | **каждый** вызов запускает полную раскладку всей формы, дважды (`widget.go:507-510`). Итог O(N²): 1000 меток добавляются 3,3 с (§4) |
| `Label.SetText` | инвалидирует только себя | запускает `form.UpdateLayout()` всей формы, даже если текст не изменился (`label.go:63`). altcalc и altpaint написали свой `statusCell`, «потому что Label перекладывает окно на каждое изменение» |
| `SetMinWidth` / `SetMaxWidth` | `updateGeometry()` | ничего не инвалидирует |
| Пакетное обновление | `SuspendLayout/ResumeLayout`, `setUpdatesEnabled(false)` | есть `LayoutingBlockPush/Pop` и `UpdateBlockPush/Pop`, но о них не знают; у Table нет `BeginUpdate/EndUpdate` |

**Что сделать.**
- Ввести флаги dirty-layout и dirty-paint на виджете и один проход раскладки и отрисовки перед кадром, как в Qt (`LayoutRequest` + `UpdateRequest`, склеиваемые в очереди).
- Все сеттеры только помечают dirty. `AddWidget`, `RemoveWidget`, `SetVisible`, `SetText`, `SetMin*` инвалидируют родителя.
- Публичные `Widget.Update()` и `Widget.UpdateLayout()`, а также потокобезопасный `Widget.RequestRepaint()`.

### 3.2. Модель событий

1. **Один колбэк на событие.** `SetOnSelectionChanged(f)` перезаписывает предыдущий. В Qt `connect` и в C# `+=` разрешают нескольких подписчиков. Следствие: altping и altagent опрашивают выделение чужой таблицы таймером раз в 50 мс, потому что слот уже занят владельцем таблицы.
2. **Аргументы через глобальный стек.** У `Table.SetOnCellMouseDblClick(func())` и `TextBox.SetOnTextBoxKeyDown(func())` колбэки без параметров, данные приходят через `ui.CurrentEvent().Parameter.(*ui.EventTableCellMouseDblClick)`. Этот приём встречается в 6+ проектах. Он не типобезопасен, и ИИ-ассистенту его не угадать.
3. **Нет событий, которые есть везде:**
   - `OnResize` / `OnLayout` у виджета. Клиенты опрашивают ширину таймером раз в 100–150 мс (altjarvis `chatview.go:110`, altssh `filepanel.go:110`) или проверяют размер прямо в `paint` (altssh terminal).
   - `OnThemeChanged`. В 7 проектах свой список `themeListeners []func()`.
   - Изменение модификаторов, когда клавиша нажата одна. altfm опрашивает `KeyModifiers()` таймером.
   - `OnMoved`, `OnStateChanged` (maximized/minimized) у формы.
   - `TextBox.OnEnter`/`OnSubmit`, `OnEditingFinished`.
4. **Неполные данные в событиях:**
   - Колесо мыши `func(dx, dy int) bool` приходит без позиции и модификаторов. Для зума «Ctrl+колесо вокруг курсора» клиенты хранят позицию мыши сами и читают `Form().KeyModifiers()` (altemu, altpaint, altnotepad).
   - **Отпускание правой и средней кнопки не доходит до виджета**, и захвата мыши для этих кнопок нет (баг, §5 #3).
5. **Программные сеттеры ведут себя непоследовательно.** `ComboBox.SetSelectedIndex` колбэк не вызывает, а `Checkbox.SetChecked` и `NumBox.SetValue` вызывают. Поэтому почти в каждом проекте есть флаги `filling`, `updating`, `selecting`. В Qt есть `blockSignals` и раздельные `activated`/`currentIndexChanged`, в WinForms пользовательские события отделены от программных (`SelectionChangeCommitted`).

**Что сделать.**
- Типизированные события с мультиподпиской, например `table.OnCellDoubleClick.Add(func(e *CellEvent))`. Оставить `SetOn…` как сахар.
- Единое правило: `*Changed` срабатывает на любое изменение, `*Activated`/`*Committed` только на пользовательское.
- Убрать `CurrentEvent()` из публичного API.
- Добавить `WheelEvent{DX, DY, X, Y, Mods}` и `OnResize`.

### 3.3. Строковые свойства (`SetProp`/`GetProp`)

Состояние многих виджетов хранится в `map[string]any`, и типы приводятся через reflection:
- текст TextBox: `props["text"]`;
- `enabled`, `align` (хранится строкой `"left"`);
- `multiline`, `padding`, `spacing`.

Последствия:
- **Скорость.** Каждый `Text()` идёт через `fmt.Sprint` из map. Каждый `SetProp` делает поиск по глобальному реестру `form.WidgetById(c.Id())` под `RWMutex` (`widget.go:1099`), чтобы найти «внешний» тип для `ProcessPropChange`.
- **Нет проверки типов.** Опечатка в ключе тихо не работает. Клиенты красят выделение через `t.SetProp("background_selected_cell", hex)` в 3 проектах, и это нигде не документировано.
- **Побочные эффекты.** У многострочного TextBox **любое** изменение свойства включает `SetXExpandable(true)`/`SetYExpandable(true)` и затирает настройки пользователя (баг, §5 #5).

В WinForms и Qt свойства типизированы и обнаруживаются через IDE (Qt Designer, PropertyGrid). Это заметно и для ИИ: строковый ключ невозможно найти автодополнением.

**Что сделать.** Перевести все состояния на типизированные поля и методы. Строковые свойства оставить только для XML-лейаута, через таблицу «ключ → сеттер», с ошибкой на неизвестный ключ.

### 3.4. Собственные виджеты: embedding вместо наследования

В Go нет виртуальных методов. nui обходит это так:
- в интерфейсе `Widgeter` есть **неэкспортированные** методы (`setId`, `attachToForm`, `applyThemeMetrics`, `updateLayout`, `nextFocus`…), поэтому реализовать его можно только встраиванием `ui.Widget`;
- «внешний» объект находится через глобальный реестр `allwidgets[id]`.

Проблемы:
- **Неясно, что можно переопределить.** `MinWidth()` у собственного типа сработает, только если родитель вызовет его через интерфейс. Методы внутри `Widget`, которые вызывают `c.Something()`, позовут базовую версию. Об этом нет ни слова в документации. В коде есть комментарии о том, как авторы сами наступали на эти грабли (`textbox.go`: «the interface value stored as focusedWidget can carry the promoted *Widget type»).
- **Нет готовой «пустой холст-виджет» точки входа.** Клиенты пишут `&ui.Widget{}` + `InitWidget()` + `SetOnPaint` (althex `pages.go:383`: «A plain widget: a Space keeps the size of SetSize only»). В Qt для этого есть `QWidget::paintEvent`, в WinForms — `UserControl` / `Control.OnPaint`.
- `AddWidget` молча ничего не делает, если виджет уже где-то зарегистрирован (`widget.go:500`). Перенос виджета к другому родителю без явного `RemoveWidget` тихо не работает.

**Что сделать.**
- Документ «Как написать свой виджет» со списком переопределяемых методов: `ProcessPaint`, `MinWidth/MinHeight`, `ProcessKeyDown`, `SetSize`…
- Конструктор `ui.NewCanvasWidget(paint func(*Canvas))` или `NewPaintBox()`.
- `AddWidget` должен паниковать или возвращать ошибку на повторное добавление, либо корректно перепривязывать виджет.

### 3.5. Раскладка (layout)

Всё строится на одном grid-лейауте. У него есть ограничения, которых нет в `TableLayoutPanel` и `QGridLayout`:

| Возможность | WinForms / Qt | nui |
|---|---|---|
| Объединение ячеек (`RowSpan`/`ColumnSpan`) | есть | нет (span есть только у заголовка Table) |
| Веса растяжения (`stretch`, `SizeType.Percent`) | есть | только `bool expandable`, ширина делится поровну; остаток целочисленного деления теряется (`widget.go:2340`) |
| Выравнивание в ячейке (`Anchor`, `Qt::AlignRight`) | есть | нет: виджет растягивается на ячейку или стоит в левом верхнем углу; `SetAnchors` — пустая заглушка (тело закомментировано, `widget.go:1380`) |
| Отступы у виджета (`Margin`) | есть | только `padding`/`spacing` у контейнера |
| HBox / VBox / Flow / Form layouts | `FlowLayoutPanel`, `QHBoxLayout`, `QFormLayout` | нет; ряд строится через `Panel` + ручной счётчик `col++` + `HSpacer` |
| Два виджета в одной ячейке | ошибка или наложение по правилам | второй **молча не раскладывается** и остаётся 300×180 в точке (0,0) (§5 #2) |
| Размер по содержимому (`AutoSize`, `adjustSize()`) | есть | у диалогов нет: ≈ 120 жёстко заданных `Form().SetSize(w,h)` по проектам, иногда «формулой» `SetSize(360, 110+len(names)*36)` |
| Единицы, зависящие от шрифта | DPI-aware layout | пиксели; Checkbox и RadioButton жёстко ставят `SetMinWidth(150)` и не подстраиваются под текст (`checkbox.go:16`) |

Статистика по клиентам: `SetPanelPadding(0)` ≈ 150 раз, `NewHSpacer()` ≈ 135 раз, ≈ 250 жёстких пиксельных размеров. Почти весь этот шум исчез бы с `HBox`/`VBox` (`Add`, `AddStretch`, `AddSpacing`), `FormLayout.AddRow(label, w)` и `DialogButtonBox`.

### 3.6. Потоки и асинхронность

Модель «один UI-поток плюс `Invoke`» сама по себе правильная. Не хватает сахара вокруг неё:
- **`ui.Invoke` не перерисовывает форму, а `form.Invoke` перерисовывает.** Клиенты используют оба и путаются, отсюда ручные `Update()` после `Invoke`.
- **Склейка частых обновлений.** Схема `atomic.Bool` + `CompareAndSwap` + `Invoke` написана вручную в altemu, althex и altssh. Троттлинг прогресса («не чаще 100 мс») повторён 5 раз только в althex.
- **Фоновая задача с отменой.** ≈ 27 горутин в altfm живут по шаблону `serial++; go …; Invoke(func(){ if serial != c.serial {return} … })`. Ничего похожего на `BackgroundWorker` или `QFuture`/`QtConcurrent` нет.
- **Таймеры нельзя остановить.** `AddTimer` ничего не возвращает (`widget.go:481`): ни Stop, ни single-shot, ни debounce. Отсюда ≈ 60 постоянных таймеров-опросчиков в клиентах. Кроме того, таймеры обходят всё дерево виджетов каждые 10 мс.
- **Модальность ведёт себя по-разному на разных ОС.** `ShowModal` на Windows и Linux возвращается сразу, на macOS блокирует (`internal/platforms/window.go:27`). Код вида «показал диалог и прочитал результат» работает на одной ОС и ломается на другой. В WinForms `ShowDialog()` всегда возвращает `DialogResult`. В Qt есть и `exec()`, и `open()`.

**Что сделать.**
- `ui.Invoke` всегда перерисовывает затронутые формы.
- `ui.InvokeCoalesced(key, f)`.
- `ui.RunAsync(ctx, work func(ctx, progress func(float64)) (T, error), done func(T, error))` с отменой при закрытии формы-владельца.
- `Timer` как объект: `ui.NewTimer(interval, f)` с методами `Start/Stop/SingleShot`.
- Единая семантика `ShowModal` (всегда асинхронная) плюс отдельный блокирующий `Exec()` с вложенным циклом, который возвращает результат.

### 3.7. Отрисовка: Canvas и текст

- **Мало примитивов.** Есть линии, прямоугольники, скруглённые прямоугольники, треугольник, картинка и текст. Нет:
  - окружности и эллипса, дуги;
  - polyline/path, заливки полигона (`fillPolygon` приватный);
  - толщины и пунктира;
  - градиентов, трансформаций (rotate/scale).

  Клиенты растеризуют «статусные точки» попиксельно (`FillRect(…,1,1)`, 3 проекта), а иконки рисуют через `fogleman/gg` в обход Canvas. Даже примеры nui рисуют индикатор сортировки и глифы калькулятора через `gg`.
- **`FillRoundedRect` медленный.** На каждый вызов создаётся `gg.Context` на всё окно с маской-клипом (`canvas.go:1033`). altcalc переписал его сам.
- **`DrawImage`:**
  - всегда использует `draw.Over`, нет быстрого `Src`-пути;
  - нет публичной отрисовки с масштабом;
  - игнорирует `img.Bounds().Min` (баг, §5 #6).

  altpaint и world_emulator пишут прямо в `cnv.RGBA()`, вычисляя смещение через `TranslatedX` и `ClipX`. althex для этого подгоняет координаты с допуском ±2 px, потому что clip хранится в физических пикселях, а translate в логических.
- **Текст:**
  - **Только начертание Regular.** Встроены только `NotoSans-Regular` и `JetBrainsMono-Regular`, API для bold и italic нет. altnotepad имитирует жирный (`emboldened()`).
  - `DrawText` переносит строки только по `"\r\n"`, простой `"\n"` не переносит (`canvas.go:1182`). У клиентов 32 места с `"\r\n"`.
  - Нет word wrap и elide в `DrawText` и Label. Приватные `truncateTextToWidth` и `wrapTextLines` переписаны клиентами 4 и 2 раза.
  - Нет shaping (HarfBuzz) и BiDi. Арабский, иврит и деванагари, лигатуры, составные графемы и эмодзи с ZWJ не поддерживаются. Курсор в TextBox ходит по рунам, а не по графемам.
  - Fallback шрифтов автоматический только для CJK. Для символов (`√`, `⌫`) и эмодзи его нет: калькулятор из примеров рисует `√` линиями, altjarvis перебирает 8 системных emoji-шрифтов вручную. Цветных эмодзи (COLR/CBDT) нет.
  - `RegisterFont` не читает `.ttc`.
- **Полная перерисовка окна** на любой `Update`, без dirty-регионов. Пустая форма 2560×1440 рисуется за 1,5 мс, но любой тяжёлый контрол, например большой TextBox, перерисовывается целиком на каждое мигание курсора (§4).

---

## 4. Производительность (замеры на HEAD)

Синтетические тесты гоняли на внутренних функциях отрисовки, без окна, на этой машине.

| Сценарий | Результат | Оценка |
|---|---|---|
| Table: заполнить 100 000 × 5 ячеек через `SetCellText2` | 122 мс, **+94 МБ** кучи (≈ 190 байт на ячейку, `map[int]*tableRow` → `map[int]*tableCell`) | память не годится для больших данных; нужен виртуальный режим |
| Table: отрисовать форму с этой таблицей | 4 мс на кадр | хорошо: видимые строки отсекаются |
| TextBox, 500 строк: одно нажатие клавиши | **39 мс** | на грани заметности |
| TextBox, 2 000 строк: нажатие / кадр | **157 мс / 127 мс** | заметные лаги; мигание курсора (250 мс) тоже вызывает кадр |
| TextBox, 20 000 строк: `SetText` / нажатие / кадр | **13 с / 5,8 с / 3,8 с** | непригодно |
| `Label.SetText` при 300 метках на форме | 0,78 мс на вызов (полная раскладка формы) | 50 меток статуса × 10 Гц ≈ 40% CPU |
| 1000 × `AddWidget` в открытую форму | **3,3 с** | O(N²): каждое добавление раскладывает всю форму |
| Пустая форма 2560×1440 | 1,5 мс на кадр | хорошо |

Причины тормозов TextBox (`textbox.go`):
- текст хранится одной строкой в `props`;
- `Lines()` на каждом вызове заново делает `strings.Split` всего текста;
- `updateInnerSize` измеряет **все** строки;
- `Draw` рисует **все** строки без отсечения по viewport;
- `AssemblyText` склеивает строки через `+=`, это O(N²);
- undo хранит до 500 полных копий текста.

**Что сделать.**
- TextBox: хранить `[]line` или piece table, отсекать по viewport, кэшировать ширины строк, хранить undo как дельты.
- Table: добавить виртуальный режим (§7.1), а данные ячеек хранить срезами, не картами.
- `AddWidget`: отложенная раскладка (§3.1).

---

## 5. Подтверждённые баги

| # | Баг | Где | Как проверено |
|---|---|---|---|
| 1 | ✅ исправлено. `RemoveWidget` не перекладывает родителя: `return` стоит до `c.updateLayout` | `ui/widget.go:541-554` | проба: после удаления левого из двух TextBox правый остался на `x=200` |
| 2 | ✅ исправлено. Два виджета в одной ячейке грида: второй не раскладывается, остаётся 300×180 в (0,0), предупреждения нет | `gridCells()` → `map[gridCell]Widgeter` | проба |
| 3 | Отпускание правой и средней кнопки мыши не доходит до `onMouseUp` виджета, захвата мыши для этих кнопок нет | `form.go:911` (`mouseLeftButtonPressedWidget` ставится только при нажатой левой), `widget.go:1680` | по коду; клиент altpaint `canvasinput.go:28`: «nui tells the release of the left button only» |
| 4 | `SetEnabled(false)` у контейнера не отключает детей: кнопка внутри отключённой Panel нажимается | `Widget.Enabled()` смотрит только на себя | проба: `clicked=1` |
| 5 | Многострочный TextBox на **любое** изменение свойства включает `SetXExpandable(true)`/`SetYExpandable(true)` | `textbox.go:235-256` | по коду; клиент altjarvis `chats_page.go:90` |
| 6 | `Canvas.DrawImage` игнорирует `img.Bounds().Min` (исходная точка `{xOffset,yOffset}`, ширина берётся из `Max.X`), поэтому sub-image рисуется неверно | `canvas.go:1091-1131` | по коду |
| 7 | `Canvas.DrawText` переносит строки только по `"\r\n"` | `canvas.go:1182` | по коду |
| 8 | ✅ исправлено. `Label.SetText` раскладывает всю форму даже при неизменном тексте | `label.go:63` | по коду и замеру |
| 9 | `SetAnchors` ничего не делает (тело закомментировано), но метод публичный | `widget.go:1380` | по коду |
| 10 | `Table.CopySelectionToClipboard` копирует только текущую ячейку, даже при мультивыделении | `table.go:2228` | по коду |
| 11 | Глобальный обработчик клавиш получает нажатия при открытом меню или попапе | `form.go:~1130` | клиенты altping, altcalc проверяют `TopPopupWidget() != nil` |
| 12 | Документация описывает несуществующие API: `ui.ShowEnterStringDialog`, `ui.ShowAboutDialog` (по `git log -S` их никогда не было), `ui.UpdateMainForm()` (в `theme.md` и `timer.md`) | `doc/ui/content/*.md` | grep |
| 13 | Нестабильная семантика `ShowModal`: блокирует только на macOS | `internal/platforms/window.go:27` | по коду и комментарию |
| 14 | Высота Label не зависит от размера шрифта: всегда `ThemeControlHeight()`, поэтому крупный текст обрезается снизу | `label.go:130` | вживую: Label 40 px обрезан по нижним выносным элементам |
| 15 | Вопрос «Сохранить изменения?» в примере notepad: Esc закрывает окно без сохранения. В nui есть только `ShowQuestionMessageBoxYesNo`, и Esc в нём — кнопка «Нет», то есть «не сохранять». Нужен вопрос «Да/Нет/Отмена», где Esc — «Отмена» | `ui/messagebox.go:50`, `examples/notepad/document.go:45` | тест `examples/notepad/notepad_ui_test.go` |

---

## 6. Платформенные пробелы

| Тема | Состояние | Сравнение |
|---|---|---|
| **HiDPI на Linux** | `scale` в Linux-бэкенде не вычисляется вообще (только Windows per-monitor и Retina на macOS). На 4K-мониторе интерфейс будет мелким; `ApplyBaseFontSize` увеличивает только шрифт, а не иконки, отступы и рамки | Qt: `QT_SCALE_FACTOR`, Xft.dpi, `GDK_SCALE` |
| **Wayland** | только через XWayland | Qt: нативный Wayland |
| **Главное меню macOS** | MenuBar рисуется внутри окна; `NSMenu` используется только для трея. Нет системного меню приложения (Cmd+Q, Hide, Services, About), что для macOS — нарушение HIG | Qt: `QMenuBar` автоматически становится нативным |
| **IME** | Linux: XIM есть (`ime_linux.go`). Windows: только `WM_CHAR`, окно композиции не позиционируется у курсора. macOS: ввод через `characters` в `keyDown:`, без `NSTextInputClient`, поэтому китайский и японский IME и, вероятно, dead keys не работают (стоит проверить) | Qt/WinForms: полноценный IME |
| **Accessibility** | отсутствует полностью: нет UIA, AT-SPI, NSAccessibility. Скринридеры видят пустое окно | WinForms: `AccessibleObject`; Qt: `QAccessible`. Для госсектора и корпоративных продаж это часто обязательное требование |
| **Файловые диалоги на Linux** | нужен внешний `zenity` или `kdialog`, иначе `ErrNoFileDialog`. Клиенты пишут fallback; диалог открывается под окном с always-on-top | Qt: свой диалог плюс портал XDG |
| **Drag-out файлов** | только Windows; на Linux и macOS перетаскивание остаётся в окне | — |
| **Буфер обмена** | только текст. altpaint вызывает `xclip`/`wl-copy` для картинок | Qt `QMimeData`: картинки, HTML, URL, свои типы |
| **Тёмная тема ОС** | не определяется и не отслеживается | Qt 6.5+: `QStyleHints::colorScheme` |
| **Экраны** | нет публичного API экранов и рабочих областей (`ScreenWorkArea` только внутренний). world_emulator: «nui does not tell the screen size» | `Screen.AllScreens`, `QScreen` |
| **Проверка окружения** | при отсутствии `$DISPLAY`/libX11 нет понятной ошибки; u00 сам делает `dlopen` | — |
| **Нативный вид** | все контролы рисуются самостоятельно, нативного стиля ОС нет | Qt: стили платформ; WinForms: нативные контролы. Это осознанный выбор, но его стоит явно задекларировать |

---

## 7. Чего не хватает существующим контролам

### 7.1. Table

Самая используемая и самая «больная» часть: 18 таблиц только в althex.

| Не хватает | Как в WinForms / Qt | Как сейчас живут клиенты |
|---|---|---|
| **Виртуальный режим / модель** (`RowCount()` + `Cell(row,col)` + `CellStyle`) | `DataGridView.VirtualMode`, `QAbstractTableModel` | заливка каждой ячейки; altfm на каждую строку делает ≈ 15 вызовов, а затем таймером раз в 100 мс проходит все строки; altagent переписывает всю таблицу раз в 200 мс |
| `InsertRow` / `RemoveRow` / `MoveRow` | есть | только `ClearRows` + полная перезаливка, выделение восстанавливается вручную |
| Встроенная сортировка и индикатор в заголовке | `SortMode`, `setSortingEnabled` | треугольник рисуется через `gg` и `SetColumnImage` или приклеивается к тексту ▲▼ (примеры nui, altping) |
| Данные строки (`RowData`) | `Row.Tag`, `Qt::UserRole` | `SetCellData2(row, 0, …)` как хранилище |
| Стили строк: фон, шрифт, жирность | `DefaultCellStyle`, `BackgroundRole` | есть только цвет текста ячейки |
| Колонка только для чтения (`SetColumnReadOnly`) | `Column.ReadOnly` | 3 вызова `SetCellEditTrigger…(i, col, false)` на каждую строку (althex) |
| Редакторы ячеек: checkbox, combo, число, дата | `DataGridViewCheckBoxColumn`, `QStyledItemDelegate::createEditor` | только TextBox или `AddWidgetOnTable` |
| Контекст для custom draw: rect, selected, hovered, focused | `CellPainting` + `e.State` | `SetCellOnDraw(func(*Canvas))` без контекста, всё захватывается замыканием; closures пересоздаются каждый тик |
| `SetColumnNameFunc` (i18n) | — | у TreeView есть, у Table нет; отсюда 15 методов `applyLanguage()` |
| Placeholder для пустой таблицы | `QTableView` + overlay | отдельная панель, которую подменяют вместо таблицы (3 проекта) |
| Растяжение любой колонки или по весам | `AutoSizeColumnsMode.Fill` + `FillWeight`, `QHeaderView::Stretch` | только `SetStretchLastColumn`; altssh растягивает первую колонку таймером |
| Скрытие и перестановка колонок, сохранение их состояния | `Column.Visible`, `DisplayIndex`; `QHeaderView::saveState` | нет; ширины сохраняют вручную в каждом проекте |
| Закреплённые колонки и строки | `Frozen` | нет |
| Переменная высота строк, перенос текста в ячейке | `AutoSizeRowsMode`, `WrapMode` | одна высота на всю таблицу |
| Копирование выделения в TSV | стандарт | только одна ячейка (§5 #10) |
| Стиль выделения, не перекрашивающий цветной текст | `SelectionForeColor` | `SetProp("background_selected_cell", …)`: недокументированное строковое свойство |
| Именование | — | `SetCellText2`, `GetCellData2`, `SetCurrentCell2`, `ScrollToCell2`: суффикс `2` — наследие старого API; `SetOnSelectionChanged(func(x, y int))` с порядком (x,y), а остальной API работает в (row,col) |

### 7.2. TextBox

| Не хватает | Комментарий |
|---|---|
| **API курсора и выделения** для многострочного текста: `SetCursor(line, col)`, `Cursor()`, `SetSelection(from, to)` в рунах или байтах, `ReplaceSelection(s)`, `InsertText(s)` | **Пример notepad из самой nui** эмулирует нажатия стрелок, чтобы поставить курсор, и `KeyChar` по символу, чтобы вставить текст (`examples/notepad/editing.go:12`: «The text box has no API to put the cursor or the selection…») |
| `Cut()`, `Copy()`, `Paste()` | примеры вызывают `text.KeyDown(KeyX, Ctrl)` (`notepad.go:229`) |
| Программная правка, которую можно отменить | `SetText` очищает историю отмены, а другого способа заменить текст нет. В notepad «Upper Case» без выделения нельзя отменить по Ctrl+Z (найдено тестом `examples/notepad/notepad_ui_test.go`) |
| Word wrap | altjarvis написал ChatView на 1100 строк (`chatview.go:14`: «The TextBox of the library does not wrap») |
| Производительность | §4 |
| `MaxLength`, маска ввода, валидатор | `MaskedTextBox`, `QValidator`, `inputMask` |
| `OnEnter`/`OnSubmit`, `OnEditingFinished`, `OnCursorChanged` | сейчас только через `CurrentEvent()` |
| Разделение на `LineEdit` и `PlainTextEdit` | один класс с флагом `multiline`, который меняет политику размеров (§5 #5) |
| Подсветка фрагментов (style spans), поиск, read-only с выделением | см. §8.3 |
| Tab-символ и отступы, `SetTabWidth` | — |
| Кнопки внутри поля: очистка, показ пароля | `QLineEdit::setClearButtonEnabled` |
| Контекстное меню Cut/Copy/Paste/Select All по умолчанию | есть во всех тулкитах |

### 7.3. ComboBox

- **Нельзя очистить или удалить элементы**: нет `Clear`, `RemoveItem`, `InsertItem`, `SetItems`. Это самый частый обходной путь: пример converter держит набор ComboBox и подменяет их («a ComboBox can't drop its items»), althex и altjarvis пересоздают ComboBox и заново навешивают обработчики.
- Нет `ItemData(i)`, `FindData(v)`, `SelectByData`. althex ищет элемент, выбирая по очереди каждый (`disk.go:932`).
- Нет `SetItemTextFunc` (i18n).
- `SetOnSelectedIndexChanged(func())` вызывается без аргументов.
- Нет иконок у элементов, разделителей и групп.
- `combobox.go` содержит два метода `AddItem` с разными сигнатурами: у ComboBox и у ComboBoxPopup.

### 7.4. TabWidget

API: `AddPage`, `SetPageNameFunc`, `SetItems`. Нет:
- `CurrentIndex`/`SetCurrentIndex`, `OnCurrentChanged` (altjarvis: «The library's TabWidget cannot switch pages from the code»);
- `RemovePage`, `InsertPage`, `SetPageVisible`, `SetPageEnabled`;
- закрываемых вкладок, перестановки перетаскиванием, иконок и тултипов вкладок;
- прокрутки и меню при переполнении;
- Ctrl+Tab и Ctrl+PgUp/PgDn;
- позиции вкладок (снизу, слева);
- режима «только панель вкладок» (QTabBar).

Итог: **6 самописных tab-bar** (altemu, althex, altnotepad, altpaint, altssh, altfm). TabWidget nui в крупных проектах не используется.

### 7.5. Остальные

| Контрол | Чего не хватает |
|---|---|
| **Label** | word wrap; elide (`…` слева, по центру, справа); выделяемый текст; вертикальное выравнивание; `SetForegroundRole("muted")`; buddy/мнемоника для поля; простая разметка (жирный, ссылка) |
| **Button** | иконка рядом с текстом (сейчас Button только с текстом, ButtonImage и ToolButton только с картинкой, поэтому altfm написал свой `drive_button`); выпадающее меню (`QToolButton::MenuButtonPopup`, SplitButton); стиль «default» и «flat»; `SetAutoRepeat`; авто-ширина |
| **Checkbox / RadioButton** | ширина по тексту вместо `SetMinWidth(150)`; явные группы `RadioGroup` / `QButtonGroup` с `OnChanged(index)` |
| **ToolButton** | текст под или рядом с иконкой; режим checkable-группы; выпадающее меню; привязка к Action |
| **ContextMenu / MenuBar** | **checkable и radio-пункты** (4 проекта имитируют галочку иконкой в `SetOnShow`); несколько шорткатов на пункт; `ShowBelow(widget)` (`dropDownFrom` приватный); привязка к Action; нативное меню на macOS |
| **Splitter** | сворачивание панели с запоминанием размера (altnotepad хранит теневые размеры 4 сплиттеров); `SaveState/RestoreState`; больше двух панелей |
| **ScrollArea** | `EnsureVisible(widget)`; публичный `ScrollBar` как отдельный виджет (3 проекта рисуют свой); логический диапазон прокрутки больше `int` пикселей (hex-вьюер, лог) |
| **ImageBox** | зум, панорамирование, fit/actual size (altfm и altpaint написали свои) |
| **ProgressBar** | неопределённый режим (marquee), формат текста |
| **NumBox / Slider** | раздельные события «пользователь изменил» и «значение изменилось»; `SetSuffix`/`SetPrefix` |
| **TreeView** | чекбоксы у узлов (`QTreeWidgetItem::setCheckState`); перестановка узлов drag & drop; фильтр/поиск; переиспользование модели из Table |
| **Chart** | разрывы в линии (пример plotter: «the chart can't show gaps»); столбчатые диаграммы (althex рисует гистограмму сам) |
| **MessageBox** | произвольные кнопки, `Yes/No/Cancel`, «Save / Don't Save / Cancel» (4 самописных диалога); иконка уровня (info, warning, error, question); «Не показывать снова»; подробный текст под спойлером |
| **Widget (общее)** | `OnResize`; тултип с динамическим текстом по координате `SetTooltipProvider(func(x, y) string)` (altemu рисует свой); `MapToScreen`/`MapToGlobal` (altfm считает через `Form().Position()` + `RectClientAreaOnWindow()`); `SetFixedSize`/`SetFixedWidth` вместо пары Min + Max + `Expandable(false)`; `InsertWidget(index)`/`ReplaceWidget`; распространение `Enabled` на детей (§5 #4) |

---

## 8. Каких контролов не хватает

Отсортировано по фактическому спросу в клиентских проектах.

### P0–P1: нужны почти каждому приложению

| Контрол | Аналог | Доказательство спроса |
|---|---|---|
| **ListView / ListBox** с моделью: режимы list, details, icons; виртуальные строки; owner draw с контекстом; чекбоксы | `ListView` + `VirtualMode`, `QListView` + модель | ≈ 12 самописных реализаций: textView в altemu (используется 9 раз), листинг althex («only the lines on the screen are made»), gridview в altfm, слои и история в altpaint, gallery в altagent, history в altcalc, checked list в althex; плюс одноколоночные Table вместо ListBox |
| **StatusBar** с секциями фиксированной и растягиваемой ширины, кликабельными секциями и прогрессом | `StatusStrip`, `QStatusBar` | свой в **11** проектах |
| **ToolBar** с группами, разделителями, overflow, привязкой к Action и обновлением enabled-состояния | `ToolStrip`, `QToolBar` | свой в 8 проектах; 3 проекта обновляют enabled таймером раз в 200 мс |
| **Action / Command**: текст и text func, иконка, несколько шорткатов, checkable/checked, enabled func, triggered; общий для меню, тулбара, шорткатов и палитры команд | `QAction`, `ICommand` | 4 самописные системы по ≈ 200–240 строк (altemu, althex, altagent, altnotepad) плюс switch в `OnGlobalKeyDown` у остальных |
| **Полноценный TabWidget / TabBar** | `TabControl`, `QTabWidget`/`QTabBar` | 6 самописных реализаций |
| **StackedWidget** | `QStackedWidget` | ручная подмена через `RemoveAllWidgets`/`AddWidget` (althex, altfm) |
| **HBox / VBox / FormLayout / DialogButtonBox** | `FlowLayoutPanel`, `QHBoxLayout`, `QFormLayout`, `QDialogButtonBox` | ≈ 150 `SetPanelPadding(0)` и ≈ 135 `HSpacer`; одинаковый ряд кнопок OK/Cancel в ≈ 50 диалогах |
| **Стандартные диалоги**: InputBox (строка, число, пароль, выбор), MessageBox с кастомными кнопками, About, Progress dialog с отменой | `InputBox`, `QInputDialog`, `QMessageBox`, `QProgressDialog` | InputDialog написан ≥ 6 раз, AboutDialog 4 раза, SaveChanges 4 раза; ProgressDialog как статус-бар со ссылкой «Cancel» |

### P1–P2: нужны приложениям «посерьёзнее»

| Контрол | Аналог | Доказательство спроса |
|---|---|---|
| **PlainTextEdit / CodeEdit**: большие файлы, style spans, номера строк и gutter, поиск, word wrap | `QPlainTextEdit` + `QSyntaxHighlighter`, Scintilla | altnotepad editor (≈ 3,4 тыс. строк), textview в altfm (1,1 тыс.), code view altemu, однострочный редактор с подсветкой в altcalc |
| **RichText / Markdown view**: только чтение, перенос, выделение, ссылки | `RichTextBox`, `QTextBrowser` | ChatView в altjarvis (≈ 1,1 тыс. строк) |
| **LogView**: append-only, follow tail, уровни и цвета, лимит строк | — (обычно `QPlainTextEdit::appendPlainText`) | altemu (UART, Problems), u00 (Table как лог) |
| **Zoom/Pan canvas** (graphics view): трансформация, rubber band, Ctrl+колесо | `QGraphicsView` | схемы и платы altemu, canvas altpaint, image view altfm |
| **ScrollBar** как отдельный виджет | `ScrollBar`, `QScrollBar` | 4 собственных скроллбара |
| **FileEdit / PathPicker**: поле, кнопка «…», диалог | — | altemu, altfm |
| **SpinBox с суффиксом, DoubleSpinBox** | `NumericUpDown`, `QSpinBox` | частично покрывается NumBox |
| **SearchBox** с иконкой, очисткой и историей | `QLineEdit` + actions | EditableComboBox + костыли |
| **Docking / MDI** | WeifenLuo DockPanel, `QDockWidget` | altnotepad: 4 вложенных Splitter с теневыми размерами; altemu: «bottom dock» |
| **HexView** | — (специализированный) | althex, altemu; можно оставить клиентам, но нужны быстрый моноширинный текст и int64-скролл |
| **Terminal view** | QTermWidget | altssh (840 строк); специфично, но нужен быстрый текст |
| **WebView** | `WebView2`, `QWebEngineView` | не встречен; вне философии «без зависимостей» |

---

## 9. Сервисы уровня приложения

Здесь самое явное дублирование кода между проектами.

| Сервис | WinForms / Qt | Копий в клиентах |
|---|---|---|
| **Сохранение и восстановление геометрии окна**: normal bounds при maximized, проверка видимости на экране | `RestoreBounds`, `saveGeometry/restoreGeometry` | **≥ 12 проектов** со своим `config/window.go` (altjarvis не сохраняет ничего); обходят «размер maximized окна равен размеру экрана», «позиция 0,0 значит неизвестна», «Maximize только после Show через Invoke» |
| **Состояние Splitter и колонок Table** | `QSplitter::saveState`, `QHeaderView::saveState` | ≥ 6 проектов |
| **Settings-хранилище** + `ConfigDir(appName)` | `Properties.Settings`, `QSettings` | `ConfigDirectory()` реализован в 7 проектах |
| **`Form.Activate()` / `BringToFront()` / `IsMinimized()`** | `Activate()`, `raise()`+`activateWindow()` | файлы `raise_linux/windows/darwin.go` побайтово совпадают в 7 проектах (Linux: Hide+Show+Maximize; Windows: прямые вызовы user32 через `SystemHandle()`) |
| **Single instance с пересылкой аргументов** | `WindowsFormsApplicationBase.IsSingleInstance`, QtSingleApplication | пакет `instance/` в 8 проектах |
| **Сворачивание в трей** (поверх TrayIcon) | — | 3 почти идентичных `tray.go` |
| **Тема как событие** + семантические роли цветов (`muted`, `link`, `accent`) + `ThemedColor{Dark, Light}` | `SystemColors`, `QPalette` + `changeEvent(PaletteChange)` | тип `themeColors{dark,light}.get()` в 8 проектах; `themeListeners`/`linkLabels`/`themedIcons` (≈ 35 мест); есть латентные баги: метки не перекрашиваются при смене темы (althex `welcome.go:49`), `ThemeChanged()` в altjarvis никто не вызывает |
| **Тонирование монохромных иконок** + стандартный набор иконок + HiDPI-иконки (SVG или multi-res) | `QIcon` (modes и states), `QStyle::standardIcon` | `tintIcon` побайтово совпадает в 5+ проектах; у примеров nui свой пакет `icons` на `gg` |
| **OpenURL / Reveal in folder** | `Process.Start`, `QDesktopServices` | `ui.OpenURL` есть, но переписан в 8 проектах (§10); «Show in folder» нет |
| **Clipboard для изображений и MIME** | `Clipboard.SetImage`, `QMimeData` | altpaint через `xclip`/`wl-copy` |
| **Встроенные переводы кнопок** (OK, Cancel, Yes, No) | — | только ru и zh; altping, altsysinfo и altnotepad регистрируют ещё 9 языков вручную |

---

## 10. Обнаруживаемость, документация и пригодность для ИИ

### 10.1. Возможности есть, но о них не знают

Клиенты переписали то, что в nui уже есть:

| Есть в nui | Переписано в клиентах |
|---|---|
| `ui.Link` (с v0.0.8) | `newLinkLabel` в 8 проектах; `ui.Link` используется в одном месте. Возможно, ему не хватает перекраски по теме или фиксированного цвета: стоит выяснить |
| `ui.OpenURL` | 8 проектов |
| `ui.MixColors` | 3 проекта |
| `ui.ParseShortcut`, `Form.AddShortcut`, `ContextMenuItem.SetShortcut`, `Mod+` | 4 собственных парсера шорткатов (нужны также несколько клавиш и enabled-гейтинг, см. Action) |
| `Form.RequestClose` | комментарии «Closing from code does not call OnClose» в 3 проектах |
| `Form.ShowMaximized` | `Show()` + `Invoke(Maximize)` в 4 проектах |
| `ThemeLineHeight()` | `MeasureText(…, "Йg")` в 3 местах |
| `ContextMenu.RemoveAllItems` | пулы скрытых пунктов меню (althex, altnotepad) |
| палитра `Link`/`Error`/`Warning`/`Success`/`Highlight` | свои `colorLink`, `colorAccent` |
| nil-safe `Form.Update` | ≈ 70 проверок `if f := c.Form(); f != nil` |

Причина: документация в `doc/ui/content` очень короткая (≈ 1900 строк на 52 файла; у TabWidget 12 строк, у ScrollArea 13), а у части файлов нет godoc (`panel.go`, `frame.go`, `scrollarea.go`, `progressbar.go`, `imagebox.go`, `chart.go`, `messagebox.go`, `timer.go`…).

### 10.2. Что мешает ИИ-ассистенту (и новому человеку)

1. **Документация расходится с кодом.** Документированы несуществующие функции (§5 #12). Для ИИ это худший случай: он уверенно сгенерирует код, который не компилируется.
2. **Неоднородность API.** Одно и то же действие называется по-разному:
   - имена с суффиксом `2` и без него;
   - порядок `(x, y)` против `(row, col)`;
   - `SetOnStateChanged(func())` у Checkbox, но `func(btn, checked)` у RadioButton;
   - `SetOnSelectedIndexChanged` и `SetOnSelectionChanged`;
   - `SetOnChanged` и `SetOnValueChanged` у NumBox одновременно;
   - `ProgressBar.SetMinValue` и `Slider.SetMin`;
   - `NewCheckbox` и `NewToggleSwitch`, но `SetIsPassword`.

   ИИ угадывает имена по аналогии, и у него не получается.
3. **Скрытая семантика**, которую не вывести из сигнатуры:
   - какие сеттеры вызывают колбэк;
   - какие требуют `UpdateLayout`;
   - что `ShowModal` не блокирует (кроме macOS);
   - что `Close()` не вызывает `OnClose`;
   - что `ui.Invoke` не перерисовывает, а `form.Invoke` перерисовывает.
4. **Строковые свойства и `CurrentEvent()`.** Их нельзя найти через типы и автодополнение.
5. **Нет справочника API.** Нужен сгенерированный `API.md`, `pkg.go.dev` с примерами (`Example…` функции) и `llms.txt` с «правилами игры» (поток, инвалидация, модальность, свой виджет). ИИ отлично работает с короткими «контрактами»; один файл на 300 строк с перечнем правил сэкономил бы больше всего ошибок.
6. **Нет тестового харнесса.** altfm лезет во внутренности через `//go:linkname` (`processKeyDown`, `paintScaled`, `openForms`) и `reflect` (поля `closed`, `title`). Публичный пакет `uitest` (впрыск ввода, рендер в картинку, поиск виджета, ожидание idle) позволил бы и людям, и ИИ проверять UI автоматически. Для ИИ-агентов это особенно ценно: они смогут сами убедиться, что код работает.
7. **XML-лейаут (`SetLayout`):**
   - поддерживает ≈ 15 тегов (нет numbox, slider, treeview, splitter…);
   - ошибки печатает через `fmt.Printf`;
   - неизвестный тег молча игнорирует;
   - обработчик с неверной сигнатурой даёт панику (`method.Interface().(func())`);
   - id нельзя связать с полями структуры (altfm делает `FindWidgetByName` 131 раз, в том числе в циклах).

   Либо доделать (валидация, `Bind(&struct)`), либо пометить экспериментальным.

---

## 11. Сравнительная матрица

✅ есть, 🟡 частично, ❌ нет

| Возможность | WinForms | Qt Widgets | nui |
|---|---|---|---|
| Сборка без C-тулчейна, один бинарник | ❌ (.NET) | ❌ | ✅ |
| Grid layout | ✅ | ✅ | ✅ |
| Span, stretch-веса, выравнивание в ячейке | ✅ | ✅ | ❌ |
| Box / Flow / Form layouts | ✅ | ✅ | ❌ |
| Авто-инвалидация layout и paint | ✅ | ✅ | 🟡 |
| Типизированные события, мультиподписка | ✅ | ✅ | ❌ |
| Action / Command | 🟡 | ✅ | ❌ |
| MenuBar, мнемоники, шорткаты | ✅ | ✅ | ✅ |
| Checkable-пункты меню | ✅ | ✅ | ❌ |
| Нативное меню macOS | — | ✅ | ❌ |
| ToolBar / StatusBar | ✅ | ✅ | ❌ |
| Table с моделью / виртуальным режимом | ✅ | ✅ | ❌ |
| ListBox / ListView | ✅ | ✅ | ❌ |
| TreeView | ✅ | ✅ | ✅ |
| TabWidget (полный API) | ✅ | ✅ | 🟡 |
| TextBox: API курсора, wrap, большие тексты | ✅ | ✅ | ❌ |
| Rich text / code editor | ✅ | ✅ | ❌ |
| PropertyGrid | ✅ | ❌ | ✅ |
| Стандартные диалоги (input, message с кнопками) | ✅ | ✅ | 🟡 |
| Файловые диалоги | ✅ | ✅ | 🟡 (Linux: zenity/kdialog) |
| Печать и PDF | ✅ | ✅ | ✅ (PDF растровый) |
| Трей, уведомления | ✅ | ✅ | ✅ |
| Drag & drop | ✅ | ✅ | 🟡 (наружу только Windows) |
| Буфер обмена (MIME, картинки) | ✅ | ✅ | 🟡 (только текст) |
| HiDPI | ✅ | ✅ | 🟡 (нет на Linux) |
| Bold/italic, shaping, BiDi | ✅ | ✅ | ❌ |
| IME | ✅ | ✅ | 🟡 |
| Accessibility | ✅ | ✅ | ❌ |
| Сохранение геометрии, settings | ✅ | ✅ | ❌ |
| Single instance | ✅ | 🟡 | ❌ |
| Тестовый харнесс UI | 🟡 | ✅ (QTest) | ❌ |
| i18n с переключением на лету | 🟡 | 🟡 | ✅ |

---

## 12. Дорожная карта

### P0. Фундамент: без него остальное строить дорого

1. **Инвалидация.** Флаги dirty-layout и dirty-paint, один проход перед кадром. Публичные `Widget.Update()` и `Widget.UpdateLayout()`, потокобезопасный `RequestRepaint()`. Все мутации (`AddWidget`, `RemoveWidget`, `SetVisible`, `SetText`, `SetMin*`) инвалидируют родителя. Исправить §5 #1, #2, #8. Ожидаемый эффект: убрать ≈ 130 ручных вызовов у клиентов и O(N²) в `AddWidget`.

   **Сделано (2026-10-25):**
   - раскладка отложенная: `Form.UpdateLayout()` ставит флаг, форма раскладывается один раз — перед кадром, перед вводом или при первом запросе геометрии (`X`, `Width`, `SetSize`…);
   - `AddWidget`, `RemoveWidget`, `SetVisible` и `SetMin*`/`SetMax*` с новым значением раскладывают форму сами;
   - `Label.SetText`: тот же текст ничего не делает, текст той же ширины только перерисовывает;
   - добавлены `Widget.Update()`, `Widget.UpdateLayout()`, `Widget.RequestRepaint()` (из любой горутины, запросы склеиваются);
   - `ui.Invoke` и `ui.InvokeSync` перерисовывают формы;
   - несколько виджетов в одной ячейке лежат друг на друге, как в `QGridLayout`; щелчок получает верхний.

   Исправлены §5 #1, #2, #8. `AddWidget` × 1000 — 3,26 с → 10 мс (см. [benchmarks.md](benchmarks.md)). Не сделано: перерисовка только изменившейся части окна (dirty-регионы); окно по-прежнему перерисовывается целиком.
2. **События.**
   - Типизированные события с мультиподпиской; убрать `CurrentEvent()` из публичного API.
   - `OnResize` у виджета.
   - Колесо мыши с позицией и модификаторами.
   - mouse-up и захват для всех кнопок (§5 #3).
   - Единое правило «программный сеттер колбэк не вызывает» (или `Changed` и `Activated`).
3. **Мутационные API базовых контролов.**
   - ComboBox: `Clear`, `RemoveItem`, `InsertItem`, `SetItems`, `ItemData`, `FindData`.
   - TabWidget: `CurrentIndex`, `SetCurrentIndex`, `OnCurrentChanged`, `RemovePage`, `InsertPage`, `SetPageVisible`.
   - TextBox: курсор, выделение, `InsertText`, `Cut`, `Copy`, `Paste`, `OnEnter`.
   - Checkable-пункты меню.
4. **Производительность TextBox** (§4) и хранение ячеек Table.
5. **Баги §5**: `Enabled` у детей, `DrawImage` bounds, `"\n"` в `DrawText`, TextBox expandable, `SetAnchors`, копирование выделения в Table.
6. **Таймер как объект** (`Stop`, `SingleShot`) и единая семантика `ShowModal`.
7. **Документация**: убрать несуществующие API; написать короткий `doc/rules.md` (или `llms.txt`) с контрактами: поток, инвалидация, модальность, жизненный цикл, свой виджет, какие сеттеры вызывают события.

### P1. Слой приложения и недостающие контролы

8. **Layouts**: `HBox`, `VBox`, `FormLayout`, `DialogButtonBox`; span и stretch-веса в grid; авторазмер диалогов по содержимому; ширина Checkbox и RadioButton по тексту.
9. **ListView и виртуальная Table** на общей модели (`ItemModel`: `Count`, `Text`, `Icon`, `Style`, `Data` + уведомления), делегат отрисовки с контекстом, сортировка, placeholder.
10. **Action + ToolBar + StatusBar**; пункты меню и кнопки строятся из Action; enabled-состояние пересчитывается при открытии меню и на idle.
11. **Сервисы окна и приложения**: `SaveState`/`RestoreState` (Form, Splitter, Table header), `Activate()`, `IsMinimized()`, `NormalBounds()`, API экранов, `Settings(app)`, `SingleInstance`.
12. **Тема**: `OnThemeChanged`, семантические роли цветов у виджетов (`SetForegroundRole`), `ThemedColor`, автотонирование монохромных иконок, отслеживание тёмной темы ОС.
13. **Стандартные диалоги**: InputBox (строка, число, пароль, выбор), MessageBox с кастомными кнопками и иконками, ProgressDialog, About.
14. **Async-хелперы**: `RunAsync` с отменой, `InvokeCoalesced`; `ui.Invoke` перерисовывает.
15. **Canvas**: эллипс, дуга, path, полигон, толщина и пунктир, быстрый `FillRoundedRect`, `DrawImage` в режиме `Src` и с масштабом, публичные `ElideText`/`WrapText`, `DeviceRect()`.
16. **Шрифты**: bold и italic (встроить Noto Sans Bold и Italic, JetBrains Mono Bold), fallback для символов и эмодзи, `.ttc`.

### P2. Паритет с Qt

17. PlainTextEdit/CodeEdit со style spans и gutter, RichText/Markdown viewer, LogView.
18. Zoom/pan canvas, отдельный ScrollBar, логический скролл (int64).
19. Docking.
20. Платформы: HiDPI на Linux (Xft.dpi, `GDK_SCALE`), нативное меню macOS, `NSTextInputClient` (IME), позиционирование IME-окна на Windows, drag-out файлов на Linux и macOS, буфер обмена с MIME и картинками, свой файловый диалог как fallback на Linux.
21. Accessibility (UIA, AT-SPI, NSAccessibility), хотя бы для базовых контролов.
22. Shaping и BiDi (go-text/typesetting).
23. Публичный пакет `uitest`.
24. XML-лейаут: валидация, все виджеты, `Bind(&struct)`; либо пометить как экспериментальный.

---

## Приложение A. Индикаторы дублирования в клиентах

Цифры приблизительные, по grep в 14 проектах.

| Паттерн | Количество | Что заменит |
|---|---|---|
| `if f := c.Form(); f != nil { f.Update() }` и свои `update()` | ≈ 70 | `Widget.Update()` |
| `Form().UpdateLayout()` вручную | ≈ 60 | авто-инвалидация |
| `SetPanelPadding(0)` | ≈ 150 | HBox/VBox, padding по умолчанию у вложенных панелей |
| `NewHSpacer()` | ≈ 135 | `AddStretch()`, DialogButtonBox |
| Жёсткие пиксельные размеры (`SetMin/Max*`, `SetSize`) | ≈ 250 | layouts по содержимому |
| `OnDialogShow = func(){ SetTitle; SetSize; MoveToCenterOfParent; SetAccept/CancelButton }` | ≈ 50 диалогов | декларативный DialogContent + auto-size |
| `AddTimer` для опроса состояния | ≈ 60 | события, Action, модели |
| `CurrentEvent().Parameter.(*T)` | ≈ 10 | типизированные события |
| `themeColors{dark, light}` / `themeListeners` | 8 проектов / ≈ 35 мест | роли цветов + `OnThemeChanged` |
| `config/window.go` (геометрия) | ≥ 12 проектов | `Form.SaveState` |
| `raise_*.go` | 7 проектов × 3 ОС | `Form.Activate()` |
| `instance/` | 8 проектов | `ui.SingleInstance` |
| `newLinkLabel` / `OpenURL` | 8 / 8 | документация `ui.Link`, `ui.OpenURL` |
| Самописный tab bar | 6 | TabWidget/TabBar |
| Самописный status bar | 11 | StatusBar |
| Самописный toolbar | 8 | ToolBar + Action |
| Самописный список / виртуальный вью | ≈ 12 | ListView + модель |
| Самописный InputDialog / SaveChanges / About | ≥ 6 / 4 / 4 | стандартные диалоги |
| Своя система команд | 4 | Action |
| Свой скроллбар | 4 | ScrollBar |
| Elide / wrap текста | 4 / 2 | публичные `ElideText` / `WrapText`, Label wrap |

## Приложение B. Самые показательные цитаты из кода

- `nui/examples/notepad/editing.go:12`: «The text box has no API to put the cursor or the selection at a place of a multiline text, so these helpers move it with the keys, as the user would.»
- `nui/examples/converter/converter.go:53`: «a ComboBox can't drop its items, so the lists of the chosen category are put into the holders»
- `nui/examples/calculator/glyphs.go:12`: «The UI font has no "√" and "⌫", so these keys draw their symbols.»
- `nui/examples/plotter/plotter.go:219`: «the chart can't show gaps»
- `altjarvis/forms/mainform.go:15`: «The library's TabWidget cannot switch pages from the code»
- `altjarvis/forms/chatview.go:14`: «The TextBox of the library does not wrap, so the messages are drawn here.»
- `altjarvis/forms/chatview.go:110`: «The width is known after the layout: it is checked here, not in the drawing» (`AddTimer(100, …)`)
- `altpaint/forms/canvasinput.go:28`: «nui tells the release of the left button only»
- `altpaint/forms/clipboard_linux.go:11`: «nui talks to X11 without cgo and has no image clipboard»
- `altcalc/forms/draw.go:9`: «Canvas.FillRoundedRect makes a mask of the whole window for each call, too slow…»
- `altcalc/forms/statusbar.go:83`: «a label would lay the window out again on every change»
- `world_emulator/forms/view3d.go:180`: «Canvas.DrawImage blends pixel by pixel and costs more than rendering the scene»
- `world_emulator/forms/editor.go:104`: «nui does not tell the screen size»
- `altping/forms/raise_linux.go:9`: «Show does nothing for a shown window, so it is hidden first … The window manager forgets the maximized state of a hidden window»
- `althex/forms/dialogs.go:131`: «Waiting for it here would hang: the windows may share one thread (they do on Linux)»
