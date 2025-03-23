package nuicanvas

// FontMap maps ASCII runes (32 to 126) to their 5x7 bitmap representation
var fixedFontMap = map[byte][]byte{
	0: {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	1: {
		0, 0, 0, 0, 0,
		0, 1, 0, 1, 0,
		0, 0, 0, 0, 0,
		0, 0, 1, 0, 0,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
		0, 0, 0, 0, 0,
	},
	2: {
		1, 1, 1, 1, 1,
		1, 0, 1, 0, 1,
		1, 1, 1, 1, 1,
		1, 1, 0, 1, 1,
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 1, 1, 1, 1,
	},
	3: {
		0, 0, 0, 0, 0,
		0, 1, 0, 1, 0,
		1, 1, 1, 1, 1,
		1, 1, 1, 1, 1,
		0, 1, 1, 1, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 0, 0,
	},
	4: {
		0, 0, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 1, 1, 1, 0,
		1, 1, 1, 1, 1,
		0, 1, 1, 1, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 0, 0,
	},
	5: {
		0, 0, 0, 0, 0,
		0, 0, 1, 0, 0,
		1, 1, 1, 1, 1,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 1, 1, 1, 0,
		0, 0, 0, 0, 0,
	},
	6: {
		0, 0, 1, 0, 0,
		0, 1, 1, 1, 0,
		1, 1, 1, 1, 1,
		1, 1, 1, 1, 1,
		0, 1, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 1, 0,
	},
	7: {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 1, 1, 1, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	8: {
		1, 1, 1, 1, 1,
		1, 1, 1, 1, 1,
		1, 1, 0, 1, 1,
		1, 0, 0, 0, 1,
		1, 1, 0, 1, 1,
		1, 1, 1, 1, 1,
		1, 1, 1, 1, 1,
	},
	9: {
		0, 0, 0, 0, 0,
		0, 0, 1, 0, 0,
		1, 1, 0, 1, 1,
		0, 0, 0, 0, 0,
		1, 1, 0, 1, 1,
		0, 0, 1, 0, 0,
		0, 0, 0, 0, 0,
	},
	10: {
		1, 1, 1, 1, 1,
		1, 1, 1, 1, 1,
		1, 1, 0, 1, 1,
		1, 0, 1, 0, 1,
		1, 1, 0, 1, 1,
		1, 1, 1, 1, 1,
		1, 1, 1, 1, 1,
	},
	11: {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 1,
		0, 1, 0, 1, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
		0, 0, 0, 0, 0,
	},
	12: {
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
		0, 0, 1, 0, 0,
		0, 1, 1, 1, 0,
		0, 0, 1, 0, 0,
	},
	13: {
		0, 0, 0, 1, 1,
		0, 0, 0, 1, 0,
		0, 0, 0, 1, 0,
		0, 0, 1, 1, 0,
		0, 1, 0, 1, 0,
		0, 1, 0, 1, 0,
		0, 0, 1, 0, 0,
	},
	14: {
		0, 0, 0, 1, 1,
		1, 1, 0, 1, 0,
		1, 0, 0, 1, 0,
		0, 0, 1, 1, 0,
		0, 1, 0, 1, 0,
		0, 1, 0, 1, 0,
		0, 0, 1, 0, 0,
	},
	15: {
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
		0, 1, 0, 1, 0,
		0, 1, 0, 1, 0,
		0, 1, 0, 1, 0,
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
	},
	16: {
		0, 1, 0, 0, 0,
		0, 1, 1, 0, 0,
		0, 1, 1, 1, 0,
		0, 1, 1, 1, 1,
		0, 1, 1, 1, 0,
		0, 1, 1, 0, 0,
		0, 1, 0, 0, 0,
	},
	17: {
		0, 0, 0, 1, 0,
		0, 0, 1, 1, 0,
		0, 1, 1, 1, 0,
		1, 1, 1, 1, 0,
		0, 1, 1, 1, 0,
		0, 0, 1, 1, 0,
		0, 0, 0, 1, 0,
	},
	18: {
		0, 0, 1, 0, 0,
		0, 1, 1, 1, 0,
		1, 0, 1, 0, 1,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 0, 0,
	},
	19: {
		0, 0, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		1, 0, 1, 0, 1,
		0, 1, 1, 1, 0,
		0, 0, 1, 0, 0,
	},
	20: {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	21: {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	22: {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	23: {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	24: {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	25: {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	26: {
		0, 0, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 1, 0,
		1, 1, 1, 1, 1,
		0, 0, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 0, 0,
	},
	27: {
		0, 0, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
		1, 1, 1, 1, 1,
		0, 1, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 0, 0,
	},
	28: {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	29: {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	30: {
		0, 0, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 1, 1, 1, 0,
		1, 1, 1, 1, 1,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	31: {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		1, 1, 1, 1, 1,
		0, 1, 1, 1, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 0, 0,
	},
	' ': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	'!': {
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 1, 0, 0,
	},
	'"': {
		0, 1, 0, 1, 0,
		0, 1, 0, 1, 0,
		0, 1, 0, 1, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	'#': {
		0, 1, 0, 1, 0,
		0, 1, 0, 1, 0,
		1, 1, 1, 1, 1,
		0, 1, 0, 1, 0,
		1, 1, 1, 1, 1,
		0, 1, 0, 1, 0,
		0, 1, 0, 1, 0,
	},
	'$': {
		0, 0, 1, 0, 0,
		0, 1, 1, 1, 1,
		1, 0, 1, 0, 0,
		0, 1, 1, 1, 0,
		0, 0, 1, 0, 1,
		1, 1, 1, 1, 0,
		0, 0, 1, 0, 0,
	},
	'%': {
		1, 1, 0, 0, 0,
		1, 1, 0, 0, 1,
		0, 0, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
		1, 0, 0, 1, 1,
		0, 0, 0, 1, 1,
	},
	'&': {
		0, 1, 1, 0, 0,
		1, 0, 0, 1, 0,
		1, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
		1, 0, 1, 0, 1,
		1, 0, 0, 1, 0,
		0, 1, 1, 0, 1,
	},
	'\'': {
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	'(': {
		0, 0, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
		0, 1, 0, 0, 0,
		0, 1, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 1, 0,
	},
	')': {
		0, 1, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 1, 0,
		0, 0, 0, 1, 0,
		0, 0, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
	},
	'*': {
		0, 0, 0, 0, 0,
		0, 0, 1, 0, 0,
		1, 0, 1, 0, 1,
		0, 1, 1, 1, 0,
		1, 0, 1, 0, 1,
		0, 0, 1, 0, 0,
		0, 0, 0, 0, 0,
	},
	'+': {
		0, 0, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		1, 1, 1, 1, 1,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 0, 0,
	},
	',': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 1, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
	},
	'-': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		1, 1, 1, 1, 1,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	'.': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 1, 1, 0, 0,
		0, 1, 1, 0, 0,
	},
	'/': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 1,
		0, 0, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
		1, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	'0': {
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 1, 1,
		1, 0, 1, 0, 1,
		1, 1, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	'1': {
		0, 0, 1, 0, 0,
		0, 1, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 1, 1, 1, 0,
	},
	'2': {
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		0, 0, 0, 0, 1,
		0, 0, 1, 1, 0,
		0, 1, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 1, 1, 1, 1,
	},
	'3': {
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		0, 0, 0, 0, 1,
		0, 0, 1, 1, 0,
		0, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	'4': {
		0, 0, 0, 1, 0,
		0, 0, 1, 1, 0,
		0, 1, 0, 1, 0,
		1, 0, 0, 1, 0,
		1, 1, 1, 1, 1,
		0, 0, 0, 1, 0,
		0, 0, 0, 1, 0,
	},
	'5': {
		1, 1, 1, 1, 1,
		1, 0, 0, 0, 0,
		1, 1, 1, 1, 0,
		0, 0, 0, 0, 1,
		0, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	'6': {
		0, 0, 1, 1, 0,
		0, 1, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	'7': {
		1, 1, 1, 1, 1,
		0, 0, 0, 0, 1,
		0, 0, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
		0, 1, 0, 0, 0,
		0, 1, 0, 0, 0,
	},
	'8': {
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	'9': {
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 1,
		0, 0, 0, 0, 1,
		0, 0, 0, 1, 0,
		0, 1, 1, 0, 0,
	},
	':': {
		0, 0, 0, 0, 0,
		0, 1, 1, 0, 0,
		0, 1, 1, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 1, 1, 0, 0,
		0, 1, 1, 0, 0,
	},
	';': {
		0, 0, 0, 0, 0,
		0, 1, 1, 0, 0,
		0, 1, 1, 0, 0,
		0, 0, 0, 0, 0,
		0, 1, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
	},
	'<': {
		0, 0, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
		1, 0, 0, 0, 0,
		0, 1, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 1, 0,
	},
	'=': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		1, 1, 1, 1, 1,
		0, 0, 0, 0, 0,
		1, 1, 1, 1, 1,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	'>': {
		0, 1, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 1, 0,
		0, 0, 0, 0, 1,
		0, 0, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
	},
	'?': {
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		0, 0, 0, 0, 1,
		0, 0, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 1, 0, 0,
	},
	'@': {
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		0, 0, 0, 0, 1,
		0, 1, 1, 0, 1,
		1, 0, 1, 0, 1,
		1, 0, 1, 0, 1,
		0, 1, 1, 1, 0,
	},
	'A': {
		0, 0, 1, 0, 0,
		0, 1, 0, 1, 0,
		1, 0, 0, 0, 1,
		1, 1, 1, 1, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
	},
	'B': {
		1, 1, 1, 1, 0,
		0, 1, 0, 0, 1,
		0, 1, 0, 0, 1,
		0, 1, 1, 1, 0,
		0, 1, 0, 0, 1,
		0, 1, 0, 0, 1,
		1, 1, 1, 1, 0,
	},
	'C': {
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	'D': {
		1, 1, 1, 1, 0,
		0, 1, 0, 0, 1,
		0, 1, 0, 0, 1,
		0, 1, 0, 0, 1,
		0, 1, 0, 0, 1,
		0, 1, 0, 0, 1,
		1, 1, 1, 1, 0,
	},
	'E': {
		1, 1, 1, 1, 1,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 1, 1, 1, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 1, 1, 1, 1,
	},
	'F': {
		1, 1, 1, 1, 1,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 1, 1, 1, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
	},
	'G': {
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 0,
		1, 0, 0, 1, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 1,
	},
	'H': {
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 1, 1, 1, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
	},
	'I': {
		0, 1, 1, 1, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 1, 1, 1, 0,
	},
	'J': {
		0, 0, 1, 1, 1,
		0, 0, 0, 1, 0,
		0, 0, 0, 1, 0,
		0, 0, 0, 1, 0,
		0, 0, 0, 1, 0,
		1, 0, 0, 1, 0,
		0, 1, 1, 0, 0,
	},
	'K': {
		1, 0, 0, 0, 1,
		1, 0, 0, 1, 0,
		1, 0, 1, 0, 0,
		1, 1, 0, 0, 0,
		1, 0, 1, 0, 0,
		1, 0, 0, 1, 0,
		1, 0, 0, 0, 1,
	},
	'L': {
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 1, 1, 1, 1,
	},
	'M': {
		1, 0, 0, 0, 1,
		1, 1, 0, 1, 1,
		1, 0, 1, 0, 1,
		1, 0, 1, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
	},
	'N': {
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 1, 0, 0, 1,
		1, 0, 1, 0, 1,
		1, 0, 0, 1, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
	},
	'O': {
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	'P': {
		1, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 1, 1, 1, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
	},
	'Q': {
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 1, 0, 1,
		1, 0, 0, 1, 0,
		0, 1, 1, 0, 1,
	},
	'R': {
		1, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 1, 1, 1, 0,
		1, 0, 1, 0, 0,
		1, 0, 0, 1, 0,
		1, 0, 0, 0, 1,
	},
	'S': {
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 0,
		0, 1, 1, 1, 0,
		0, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	'T': {
		1, 1, 1, 1, 1,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
	},
	'U': {
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	'V': {
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 0, 1, 0,
		0, 0, 1, 0, 0,
	},
	'W': {
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 1, 0, 1,
		1, 0, 1, 0, 1,
		1, 0, 1, 0, 1,
		0, 1, 0, 1, 0,
	},
	'X': {
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
	},
	'Y': {
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
	},
	'Z': {
		1, 1, 1, 1, 1,
		0, 0, 0, 0, 1,
		0, 0, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 1, 1, 1, 1,
	},
	'[': {
		0, 1, 1, 1, 0,
		0, 1, 0, 0, 0,
		0, 1, 0, 0, 0,
		0, 1, 0, 0, 0,
		0, 1, 0, 0, 0,
		0, 1, 0, 0, 0,
		0, 1, 1, 1, 0,
	},
	'\\': {
		0, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		0, 1, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 1, 0,
		0, 0, 0, 0, 1,
		0, 0, 0, 0, 0,
	},
	']': {
		0, 1, 1, 1, 0,
		0, 0, 0, 1, 0,
		0, 0, 0, 1, 0,
		0, 0, 0, 1, 0,
		0, 0, 0, 1, 0,
		0, 0, 0, 1, 0,
		0, 1, 1, 1, 0,
	},
	'^': {
		0, 0, 1, 0, 0,
		0, 1, 0, 1, 0,
		1, 0, 0, 0, 1,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	'_': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		1, 1, 1, 1, 1,
	},
	'`': {
		0, 1, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 1, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
	'a': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 1, 1, 1, 0,
		0, 0, 0, 0, 1,
		0, 1, 1, 1, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 1,
	},
	'b': {
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 0, 1, 1, 0,
		1, 1, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 1, 1, 1, 0,
	},
	'c': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	'd': {
		0, 0, 0, 0, 1,
		0, 0, 0, 0, 1,
		0, 1, 1, 0, 1,
		1, 0, 0, 1, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 1,
	},
	'e': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 1, 1, 1, 1,
		1, 0, 0, 0, 0,
		0, 1, 1, 1, 0,
	},
	'f': {
		0, 0, 1, 1, 0,
		0, 1, 0, 0, 1,
		0, 1, 0, 0, 0,
		1, 1, 1, 0, 0,
		0, 1, 0, 0, 0,
		0, 1, 0, 0, 0,
		0, 1, 0, 0, 0,
	},
	'g': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 1, 1, 1, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 1,
		0, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	'h': {
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 0, 1, 1, 0,
		1, 1, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
	},
	'i': {
		0, 0, 1, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 1, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 1, 1, 1, 0,
	},
	'j': {
		0, 0, 0, 0, 1,
		0, 0, 0, 0, 0,
		0, 0, 0, 1, 1,
		0, 0, 0, 0, 1,
		0, 0, 0, 0, 1,
		0, 1, 0, 0, 1,
		0, 0, 1, 1, 0,
	},
	'k': {
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 1, 0,
		1, 0, 1, 0, 0,
		1, 1, 0, 0, 0,
		1, 0, 1, 0, 0,
		1, 0, 0, 1, 0,
	},
	'l': {
		0, 0, 1, 0, 0,
		0, 1, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 1, 1, 1, 0,
	},
	'm': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		1, 1, 0, 1, 0,
		1, 0, 1, 0, 1,
		1, 0, 1, 0, 1,
		1, 0, 1, 0, 1,
		1, 0, 1, 0, 1,
	},
	'n': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		1, 0, 1, 1, 0,
		1, 1, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
	},
	'o': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	'p': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		1, 1, 1, 1, 0,
		1, 0, 0, 0, 1,
		1, 1, 1, 1, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
	},
	'q': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 1, 1, 0, 1,
		1, 0, 0, 1, 1,
		0, 1, 1, 1, 1,
		0, 0, 0, 0, 1,
		0, 0, 0, 0, 1,
	},
	'r': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		1, 0, 1, 1, 0,
		1, 1, 0, 0, 1,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
		1, 0, 0, 0, 0,
	},
	's': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 1, 1, 1, 0,
		1, 0, 0, 0, 0,
		0, 1, 1, 1, 0,
		0, 0, 0, 0, 1,
		1, 1, 1, 1, 0,
	},
	't': {
		0, 1, 0, 0, 0,
		0, 1, 0, 0, 0,
		1, 1, 1, 0, 0,
		0, 1, 0, 0, 0,
		0, 1, 0, 0, 0,
		0, 1, 0, 0, 1,
		0, 0, 1, 1, 0,
	},
	'u': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 1, 1,
		0, 1, 1, 0, 1,
	},
	'v': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 0, 1, 0,
		0, 0, 1, 0, 0,
	},
	'w': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		1, 0, 1, 0, 1,
		1, 0, 1, 0, 1,
		0, 1, 0, 1, 0,
	},
	'x': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		1, 0, 0, 0, 1,
		0, 1, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 1, 0,
		1, 0, 0, 0, 1,
	},
	'y': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		1, 0, 0, 0, 1,
		1, 0, 0, 0, 1,
		0, 1, 1, 1, 1,
		0, 0, 0, 0, 1,
		0, 1, 1, 1, 0,
	},
	'z': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		1, 1, 1, 1, 1,
		0, 0, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
		1, 1, 1, 1, 1,
	},
	'{': {
		0, 0, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 1, 0,
	},
	'|': {
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
	},
	'}': {
		0, 1, 0, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 0, 0, 1, 0,
		0, 0, 1, 0, 0,
		0, 0, 1, 0, 0,
		0, 1, 0, 0, 0,
	},
	'~': {
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 1, 1, 0, 1,
		1, 0, 0, 1, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
	},
}

func GetChar(char byte) []byte {
	return fixedFontMap[char]
}
