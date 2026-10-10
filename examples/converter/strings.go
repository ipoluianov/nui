package converter

import (
	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/i18n"
)

// Strings are all the texts of the converter, a value per language.
type Strings struct {
	Title       string
	Category    string
	Conversion  string
	Swap        string
	Precision   string
	PrecisionTT string
	Thousands   string
	ThousandsTT string
	CommonTitle string
	CommonTT    string
	ColumnUnit  string
	ColumnValue string
	Search      string
	SearchHint  string
	SearchTT    string
	Language    string
	CopyValue   string
	ConvertTo   string

	// How numbers are written
	DecimalSeparator   string
	ThousandsSeparator string

	Categories CategoryStrings
	Units      UnitStrings
}

type CategoryStrings struct {
	Length      string
	Area        string
	Volume      string
	Mass        string
	Temperature string
	Speed       string
	Data        string
}

// UnitName is the name of a unit and its symbol, which some languages
// write in their own script.
type UnitName struct {
	Name   string
	Symbol string
}

type UnitStrings struct {
	Millimetre, Centimetre, Metre, Kilometre, Inch, Foot, Yard, Mile, NauticalMile            UnitName
	SquareCentimetre, SquareMetre, Hectare, SquareKilometre, SquareFoot, Acre, SquareMile     UnitName
	Millilitre, Litre, CubicMetre, FluidOunce, Cup, GallonUS, GallonUK                        UnitName
	Milligram, Gram, Kilogram, Tonne, Ounce, Pound, Stone                                     UnitName
	Celsius, Fahrenheit, Kelvin                                                               UnitName
	MetrePerSecond, KilometrePerHour, MilePerHour, Knot, FootPerSecond                        UnitName
	Bit, Byte, Kilobyte, Megabyte, Gigabyte, Terabyte, Kibibyte, Mebibyte, Gibibyte, Tebibyte UnitName
}

var en = Strings{
	Title:              "Unit Converter",
	Category:           "Category",
	Conversion:         "Conversion",
	Swap:               "Swap the units",
	Precision:          "Precision",
	PrecisionTT:        "Digits after the decimal point",
	Thousands:          "Thousands separators",
	ThousandsTT:        "Group the digits of the result: 1,000,000",
	CommonTitle:        "Common values",
	CommonTT:           "The value in every unit of the category; right click to copy",
	ColumnUnit:         "Unit",
	ColumnValue:        "Value",
	Search:             "Quick search",
	SearchHint:         "Type a unit: \"kilo\", \"gallon\"...",
	SearchTT:           "Find a unit in any category",
	Language:           "Language",
	CopyValue:          "Copy value",
	ConvertTo:          "Convert to this unit",
	DecimalSeparator:   ".",
	ThousandsSeparator: ",",
	Categories: CategoryStrings{
		Length: "Length", Area: "Area", Volume: "Volume", Mass: "Mass",
		Temperature: "Temperature", Speed: "Speed", Data: "Data size",
	},
	Units: UnitStrings{
		Millimetre:   UnitName{"Millimetre", "mm"},
		Centimetre:   UnitName{"Centimetre", "cm"},
		Metre:        UnitName{"Metre", "m"},
		Kilometre:    UnitName{"Kilometre", "km"},
		Inch:         UnitName{"Inch", "in"},
		Foot:         UnitName{"Foot", "ft"},
		Yard:         UnitName{"Yard", "yd"},
		Mile:         UnitName{"Mile", "mi"},
		NauticalMile: UnitName{"Nautical mile", "nmi"},

		SquareCentimetre: UnitName{"Square centimetre", "cm²"},
		SquareMetre:      UnitName{"Square metre", "m²"},
		Hectare:          UnitName{"Hectare", "ha"},
		SquareKilometre:  UnitName{"Square kilometre", "km²"},
		SquareFoot:       UnitName{"Square foot", "ft²"},
		Acre:             UnitName{"Acre", "ac"},
		SquareMile:       UnitName{"Square mile", "mi²"},

		Millilitre: UnitName{"Millilitre", "ml"},
		Litre:      UnitName{"Litre", "l"},
		CubicMetre: UnitName{"Cubic metre", "m³"},
		FluidOunce: UnitName{"Fluid ounce (US)", "fl oz"},
		Cup:        UnitName{"Cup (US)", "cup"},
		GallonUS:   UnitName{"Gallon (US)", "gal"},
		GallonUK:   UnitName{"Gallon (UK)", "imp gal"},

		Milligram: UnitName{"Milligram", "mg"},
		Gram:      UnitName{"Gram", "g"},
		Kilogram:  UnitName{"Kilogram", "kg"},
		Tonne:     UnitName{"Tonne", "t"},
		Ounce:     UnitName{"Ounce", "oz"},
		Pound:     UnitName{"Pound", "lb"},
		Stone:     UnitName{"Stone", "st"},

		Celsius:    UnitName{"Celsius", "°C"},
		Fahrenheit: UnitName{"Fahrenheit", "°F"},
		Kelvin:     UnitName{"Kelvin", "K"},

		MetrePerSecond:   UnitName{"Metre per second", "m/s"},
		KilometrePerHour: UnitName{"Kilometre per hour", "km/h"},
		MilePerHour:      UnitName{"Mile per hour", "mph"},
		Knot:             UnitName{"Knot", "kn"},
		FootPerSecond:    UnitName{"Foot per second", "ft/s"},

		Bit:      UnitName{"Bit", "bit"},
		Byte:     UnitName{"Byte", "B"},
		Kilobyte: UnitName{"Kilobyte", "kB"},
		Megabyte: UnitName{"Megabyte", "MB"},
		Gigabyte: UnitName{"Gigabyte", "GB"},
		Terabyte: UnitName{"Terabyte", "TB"},
		Kibibyte: UnitName{"Kibibyte", "KiB"},
		Mebibyte: UnitName{"Mebibyte", "MiB"},
		Gibibyte: UnitName{"Gibibyte", "GiB"},
		Tebibyte: UnitName{"Tebibyte", "TiB"},
	},
}

var ru = Strings{
	Title:              "Конвертер величин",
	Category:           "Величина",
	Conversion:         "Перевод",
	Swap:               "Поменять единицы местами",
	Precision:          "Точность",
	PrecisionTT:        "Знаков после запятой",
	Thousands:          "Разделять разряды",
	ThousandsTT:        "Группировать цифры результата: 1 000 000",
	CommonTitle:        "Во всех единицах",
	CommonTT:           "Значение во всех единицах величины; правый щелчок - копировать",
	ColumnUnit:         "Единица",
	ColumnValue:        "Значение",
	Search:             "Быстрый поиск",
	SearchHint:         "Введите единицу: «кило», «галлон»...",
	SearchTT:           "Найти единицу в любой величине",
	Language:           "Язык",
	CopyValue:          "Копировать значение",
	ConvertTo:          "Переводить в эту единицу",
	DecimalSeparator:   ",",
	ThousandsSeparator: " ", // a no-break space
	Categories: CategoryStrings{
		Length: "Длина", Area: "Площадь", Volume: "Объём", Mass: "Масса",
		Temperature: "Температура", Speed: "Скорость", Data: "Объём данных",
	},
	Units: UnitStrings{
		Millimetre:   UnitName{"Миллиметр", "мм"},
		Centimetre:   UnitName{"Сантиметр", "см"},
		Metre:        UnitName{"Метр", "м"},
		Kilometre:    UnitName{"Километр", "км"},
		Inch:         UnitName{"Дюйм", "дюйм"},
		Foot:         UnitName{"Фут", "фут"},
		Yard:         UnitName{"Ярд", "ярд"},
		Mile:         UnitName{"Миля", "миля"},
		NauticalMile: UnitName{"Морская миля", "мор. миля"},

		SquareCentimetre: UnitName{"Квадратный сантиметр", "см²"},
		SquareMetre:      UnitName{"Квадратный метр", "м²"},
		Hectare:          UnitName{"Гектар", "га"},
		SquareKilometre:  UnitName{"Квадратный километр", "км²"},
		SquareFoot:       UnitName{"Квадратный фут", "фут²"},
		Acre:             UnitName{"Акр", "акр"},
		SquareMile:       UnitName{"Квадратная миля", "миля²"},

		Millilitre: UnitName{"Миллилитр", "мл"},
		Litre:      UnitName{"Литр", "л"},
		CubicMetre: UnitName{"Кубический метр", "м³"},
		FluidOunce: UnitName{"Жидкая унция (США)", "жид. унц."},
		Cup:        UnitName{"Чашка (США)", "чашка"},
		GallonUS:   UnitName{"Галлон (США)", "гал"},
		GallonUK:   UnitName{"Галлон (брит.)", "брит. гал"},

		Milligram: UnitName{"Миллиграмм", "мг"},
		Gram:      UnitName{"Грамм", "г"},
		Kilogram:  UnitName{"Килограмм", "кг"},
		Tonne:     UnitName{"Тонна", "т"},
		Ounce:     UnitName{"Унция", "унц."},
		Pound:     UnitName{"Фунт", "фунт"},
		Stone:     UnitName{"Стоун", "стоун"},

		Celsius:    UnitName{"Градус Цельсия", "°C"},
		Fahrenheit: UnitName{"Градус Фаренгейта", "°F"},
		Kelvin:     UnitName{"Кельвин", "К"},

		MetrePerSecond:   UnitName{"Метр в секунду", "м/с"},
		KilometrePerHour: UnitName{"Километр в час", "км/ч"},
		MilePerHour:      UnitName{"Миля в час", "миль/ч"},
		Knot:             UnitName{"Узел", "уз"},
		FootPerSecond:    UnitName{"Фут в секунду", "фут/с"},

		Bit:      UnitName{"Бит", "бит"},
		Byte:     UnitName{"Байт", "Б"},
		Kilobyte: UnitName{"Килобайт", "кБ"},
		Megabyte: UnitName{"Мегабайт", "МБ"},
		Gigabyte: UnitName{"Гигабайт", "ГБ"},
		Terabyte: UnitName{"Терабайт", "ТБ"},
		Kibibyte: UnitName{"Кибибайт", "КиБ"},
		Mebibyte: UnitName{"Мебибайт", "МиБ"},
		Gibibyte: UnitName{"Гибибайт", "ГиБ"},
		Tebibyte: UnitName{"Тебибайт", "ТиБ"},
	},
}

var zh = Strings{
	Title:              "单位换算器",
	Category:           "类别",
	Conversion:         "换算",
	Swap:               "交换单位",
	Precision:          "精度",
	PrecisionTT:        "小数点后的位数",
	Thousands:          "千位分隔符",
	ThousandsTT:        "将结果的数字分组：1,000,000",
	CommonTitle:        "常用数值",
	CommonTT:           "该数值在本类别各单位下的值；右键单击可复制",
	ColumnUnit:         "单位",
	ColumnValue:        "数值",
	Search:             "快速搜索",
	SearchHint:         "输入单位：“千”、“加仑”……",
	SearchTT:           "在所有类别中查找单位",
	Language:           "语言",
	CopyValue:          "复制数值",
	ConvertTo:          "换算为此单位",
	DecimalSeparator:   ".",
	ThousandsSeparator: ",",
	Categories: CategoryStrings{
		Length: "长度", Area: "面积", Volume: "体积", Mass: "质量",
		Temperature: "温度", Speed: "速度", Data: "数据大小",
	},
	Units: UnitStrings{
		Millimetre:   UnitName{"毫米", "mm"},
		Centimetre:   UnitName{"厘米", "cm"},
		Metre:        UnitName{"米", "m"},
		Kilometre:    UnitName{"千米", "km"},
		Inch:         UnitName{"英寸", "in"},
		Foot:         UnitName{"英尺", "ft"},
		Yard:         UnitName{"码", "yd"},
		Mile:         UnitName{"英里", "mi"},
		NauticalMile: UnitName{"海里", "nmi"},

		SquareCentimetre: UnitName{"平方厘米", "cm²"},
		SquareMetre:      UnitName{"平方米", "m²"},
		Hectare:          UnitName{"公顷", "ha"},
		SquareKilometre:  UnitName{"平方千米", "km²"},
		SquareFoot:       UnitName{"平方英尺", "ft²"},
		Acre:             UnitName{"英亩", "ac"},
		SquareMile:       UnitName{"平方英里", "mi²"},

		Millilitre: UnitName{"毫升", "ml"},
		Litre:      UnitName{"升", "l"},
		CubicMetre: UnitName{"立方米", "m³"},
		FluidOunce: UnitName{"液量盎司（美制）", "fl oz"},
		Cup:        UnitName{"杯（美制）", "cup"},
		GallonUS:   UnitName{"加仑（美制）", "gal"},
		GallonUK:   UnitName{"加仑（英制）", "imp gal"},

		Milligram: UnitName{"毫克", "mg"},
		Gram:      UnitName{"克", "g"},
		Kilogram:  UnitName{"千克", "kg"},
		Tonne:     UnitName{"吨", "t"},
		Ounce:     UnitName{"盎司", "oz"},
		Pound:     UnitName{"磅", "lb"},
		Stone:     UnitName{"英石", "st"},

		Celsius:    UnitName{"摄氏度", "°C"},
		Fahrenheit: UnitName{"华氏度", "°F"},
		Kelvin:     UnitName{"开尔文", "K"},

		MetrePerSecond:   UnitName{"米每秒", "m/s"},
		KilometrePerHour: UnitName{"千米每小时", "km/h"},
		MilePerHour:      UnitName{"英里每小时", "mph"},
		Knot:             UnitName{"节", "kn"},
		FootPerSecond:    UnitName{"英尺每秒", "ft/s"},

		Bit:      UnitName{"比特", "bit"},
		Byte:     UnitName{"字节", "B"},
		Kilobyte: UnitName{"千字节", "kB"},
		Megabyte: UnitName{"兆字节", "MB"},
		Gigabyte: UnitName{"吉字节", "GB"},
		Terabyte: UnitName{"太字节", "TB"},
		Kibibyte: UnitName{"二进制千字节", "KiB"},
		Mebibyte: UnitName{"二进制兆字节", "MiB"},
		Gibibyte: UnitName{"二进制吉字节", "GiB"},
		Tebibyte: UnitName{"二进制太字节", "TiB"},
	},
}

var catalog = i18n.NewCatalog(en, map[string]Strings{"ru": ru, "zh": zh})

// T returns the texts in the language of the application.
func T() *Strings {
	return catalog.Get(ui.Language())
}
